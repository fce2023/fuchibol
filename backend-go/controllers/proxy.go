package controllers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

// Regexes for parsing m3u8 playlists
var reUri = regexp.MustCompile(`URI="([^"]+)"`)

// Shared HTTP client with connection pooling and timeouts to optimize proxying
var proxyClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100, // Important for high concurrency requests to same IPTV provider
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	Timeout: 15 * time.Second,
}

func resolveUrl(base *url.URL, ref string) string {
	refUrl, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(refUrl).String()
}

func fetchAndProxy(c *fiber.Ctx, targetUrl string) error {
	parsedTarget, err := url.Parse(targetUrl)
	if err != nil {
		return c.Status(400).SendString("Invalid target URL")
	}

	// Try to serve .ts segments from Redis cache if available
	isTS := strings.HasSuffix(strings.ToLower(parsedTarget.Path), ".ts")
	cacheKey := fmt.Sprintf("iptv:ts:%s", base64.URLEncoding.EncodeToString([]byte(targetUrl)))
	
	if isTS && database.RedisClient != nil {
		cachedData, err := database.RedisClient.Get(database.Ctx, cacheKey).Bytes()
		if err == nil && len(cachedData) > 0 {
			c.Set("Access-Control-Allow-Origin", "*")
			c.Set("Content-Type", "video/mp2t")
			c.Set("Content-Length", strconv.Itoa(len(cachedData)))
			c.Set("X-Cache", "HIT")
			return c.Send(cachedData)
		}
	}

	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return c.Status(500).SendString("Failed to create request")
	}

	// IPTV providers often block default Go user agents
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	resp, err := proxyClient.Do(req)
	if err != nil {
		return c.Status(502).SendString(fmt.Sprintf("Bad Gateway: %v", err))
	}

	contentType := resp.Header.Get("Content-Type")

	// Set CORS headers for everything
	c.Set("Access-Control-Allow-Origin", "*")

	// If it's a playlist, we MUST rewrite it to point back to our proxy
	if strings.Contains(strings.ToLower(contentType), "mpegurl") || strings.HasSuffix(strings.ToLower(parsedTarget.Path), ".m3u8") {
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(500).SendString("Error reading playlist body")
		}
		
		bodyStr := string(bodyBytes)

		// If it's a master playlist containing nested streams, flatten it on the fly
		if strings.Contains(bodyStr, "#EXT-X-STREAM-INF") {
			lines := strings.Split(bodyStr, "\n")
			nestedUrlStr := ""
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
					nestedUrlStr = resolveUrl(parsedTarget, trimmed)
					break
				}
			}
			if nestedUrlStr != "" {
				return fetchAndProxy(c, nestedUrlStr)
			}
		}
		
		rewrittenLines := RewritePlaylistContent(bodyStr, parsedTarget, c.BaseURL())
		c.Set("Content-Type", "application/vnd.apple.mpegurl")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendString(rewrittenLines)
	}

	// For .ts files, stream them transparently and cache in Redis if successful
	if isTS && resp.StatusCode == http.StatusOK && database.RedisClient != nil {
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			// Cache in Redis with 60s TTL
			database.RedisClient.Set(database.Ctx, cacheKey, bodyBytes, 60*time.Second)
			c.Status(resp.StatusCode)
			c.Set("Content-Type", "video/mp2t")
			c.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
			c.Set("X-Cache", "MISS")
			return c.Send(bodyBytes)
		}
	}

	// For keys or other files, stream them transparently without modifications
	c.Status(resp.StatusCode)
	c.Set("Content-Type", contentType)
	c.Set("X-Cache", "MISS")
	contentLength := resp.Header.Get("Content-Length")
	if contentLength != "" {
		c.Set("Content-Length", contentLength)
		if size, err := strconv.Atoi(contentLength); err == nil && size >= 0 {
			return c.SendStream(resp.Body, size)
		}
	}
	
	return c.SendStream(resp.Body)
}

// ProxyIPTV handles the initial m3u8 request for a channel
func ProxyIPTV(c *fiber.Ctx) error {
	id := c.Params("id")
	var channel models.Channel
	if err := database.DB.First(&channel, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	if !channel.IptvEnabled || channel.ActiveIptvUrl == nil || *channel.ActiveIptvUrl == "" {
		return c.Status(404).JSON(fiber.Map{"error": "IPTV is not enabled or no URL is set"})
	}

	return fetchAndProxy(c, *channel.ActiveIptvUrl)
}

// ProxyTS handles all subsequent fragment or nested playlist requests
func ProxyTS(c *fiber.Ctx) error {
	encodedUrl := c.Query("url")
	if encodedUrl == "" {
		return c.Status(400).SendString("Missing url parameter")
	}

	decodedBytes, err := base64.URLEncoding.DecodeString(encodedUrl)
	if err != nil {
		return c.Status(400).SendString("Invalid encoded url parameter")
	}

	targetUrl := string(decodedBytes)
	return fetchAndProxy(c, targetUrl)
}

func RewritePlaylistContent(bodyStr string, parsedTarget *url.URL, baseURL string) string {
	var rewrittenLines []string
	lines := strings.Split(bodyStr, "\n")
	baseURL = strings.TrimSuffix(baseURL, "/")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			rewrittenLines = append(rewrittenLines, line)
			continue
		}
		
		// If it's a tag line (starts with #)
		if strings.HasPrefix(trimmed, "#") {
			// Rewrite URIs inside tags (like URI="chunklist.m3u8" or URI="key.key")
			rewrittenLine := reUri.ReplaceAllStringFunc(line, func(match string) string {
				submatch := reUri.FindStringSubmatch(match)
				if len(submatch) > 1 {
					absUrl := resolveUrl(parsedTarget, submatch[1])
					encodedUrl := base64.URLEncoding.EncodeToString([]byte(absUrl))
					return fmt.Sprintf(`URI="%s/api/v1/proxy/ts?url=%s"`, baseURL, encodedUrl)
				}
				return match
			})
			rewrittenLines = append(rewrittenLines, rewrittenLine)
		} else {
			// 1. Rewrite plain URLs (usually .ts or nested .m3u8)
			absUrl := resolveUrl(parsedTarget, trimmed)
			encodedUrl := base64.URLEncoding.EncodeToString([]byte(absUrl))
			newUrl := fmt.Sprintf("%s/api/v1/proxy/ts?url=%s", baseURL, encodedUrl)
			rewrittenLines = append(rewrittenLines, newUrl)
		}
	}
	return strings.Join(rewrittenLines, "\n")
}
