package controllers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

// Regexes for parsing m3u8 playlists
var reUri = regexp.MustCompile(`URI="([^"]+)"`)

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

	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return c.Status(500).SendString("Failed to create request")
	}

	// IPTV providers often block default Go user agents
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(502).SendString(fmt.Sprintf("Bad Gateway: %v", err))
	}

	contentType := resp.Header.Get("Content-Type")

	// Set CORS headers for everything
	c.Set("Access-Control-Allow-Origin", "*")

	// If it's a playlist, we MUST rewrite it to point back to our proxy
	if strings.Contains(strings.ToLower(contentType), "mpegurl") || strings.HasSuffix(parsedTarget.Path, ".m3u8") {
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

	// For .ts files or keys, stream them transparently without modifications
	c.Status(resp.StatusCode)
	c.Set("Content-Type", contentType)
	contentLength := resp.Header.Get("Content-Length")
	if contentLength != "" {
		c.Set("Content-Length", contentLength)
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
					return fmt.Sprintf(`URI="/api/v1/proxy/ts?url=%s"`, encodedUrl)
				}
				return match
			})
			rewrittenLines = append(rewrittenLines, rewrittenLine)
		} else {
			// 1. Rewrite plain URLs (usually .ts or nested .m3u8)
			absUrl := resolveUrl(parsedTarget, trimmed)
			encodedUrl := base64.URLEncoding.EncodeToString([]byte(absUrl))
			newUrl := fmt.Sprintf("/api/v1/proxy/ts?url=%s", encodedUrl)
			rewrittenLines = append(rewrittenLines, newUrl)
		}
	}
	return strings.Join(rewrittenLines, "\n")
}
