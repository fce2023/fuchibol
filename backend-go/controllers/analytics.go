package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

type GeoResponse struct {
	Status  string  `json:"status"`
	Country string  `json:"country"`
	City    string  `json:"city"`
	Query   string  `json:"query"`
}

func resolveGeo(ip string) (string, string) {
	if ip == "127.0.0.1" || ip == "::1" || ip == "" {
		return "Local", "Development"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,city", ip))
	if err != nil {
		return "Unknown", "Unknown"
	}
	defer resp.Body.Close()

	var geo GeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil || geo.Status != "success" {
		return "Unknown", "Unknown"
	}

	return geo.Country, geo.City
}

// TrackView registers a viewer hit for a channel
func TrackView(c *fiber.Ctx) error {
	type Request struct {
		ChannelID uint `json:"channel_id"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	ip := c.IP()
	
	// Handle proxy headers with priority for real client IP
	if cf := c.Get("CF-Connecting-IP"); cf != "" {
		ip = cf
	} else if trueClient := c.Get("True-Client-IP"); trueClient != "" {
		ip = trueClient
	} else if ff := c.Get("X-Forwarded-For"); ff != "" {
		// X-Forwarded-For can contain multiple IPs separated by commas
		parts := strings.Split(ff, ",")
		ip = strings.TrimSpace(parts[0])
	} else if realIP := c.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}

	country, city := resolveGeo(ip)

	log := models.AnalyticsLog{
		ChannelID: req.ChannelID,
		IPAddress: ip,
		Country:   country,
		City:      city,
		UserAgent: c.Get("User-Agent"),
	}

	database.DB.Create(&log)

	return c.JSON(fiber.Map{"status": "tracked"})
}

// GetAnalyticsSummary returns stats for the current user's channel
func GetAnalyticsSummary(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	// 1. Total Views
	var totalViews int64
	database.DB.Model(&models.AnalyticsLog{}).Where("channel_id = ?", channel.ID).Count(&totalViews)

	// 2. Top Countries
	type CountryStat struct {
		Country string `json:"country"`
		Count   int64  `json:"count"`
	}
	topCountries := []CountryStat{}
	database.DB.Model(&models.AnalyticsLog{}).
		Select("country, count(*) as count").
		Where("channel_id = ?", channel.ID).
		Group("country").
		Order("count DESC").
		Limit(5).
		Scan(&topCountries)

	// 3. Views last 24h
	var viewsLast24h int64
	yesterday := time.Now().Add(-24 * time.Hour)
	database.DB.Model(&models.AnalyticsLog{}).
		Where("channel_id = ? AND created_at > ?", channel.ID, yesterday).
		Count(&viewsLast24h)

	return c.JSON(fiber.Map{
		"total_views":    totalViews,
		"views_last_24h": viewsLast24h,
		"top_countries":  topCountries,
	})
}
