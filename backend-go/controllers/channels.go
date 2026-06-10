package controllers

import (
	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

// GetChannels returns all live channels
func GetChannels(c *fiber.Ctx) error {
	var channels []models.Channel
	
	// Preload the User info but only return live channels
	database.DB.Preload("User").Where("is_live = ?", true).Find(&channels)
	
	return c.JSON(fiber.Map{
		"channels": channels,
	})
}

// GetChannel details by username or channel name
func GetChannel(c *fiber.Ctx) error {
	name := c.Params("name")
	var channel models.Channel

	if err := database.DB.Preload("User").Where("name = ?", name).First(&channel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Channel not found",
		})
	}

	return c.JSON(fiber.Map{
		"channel": channel,
	})
}

type UpdateChannelRequest struct {
	Name           *string `json:"name"`
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	Category       *string `json:"category"`
	IptvUrls       *string `json:"iptv_urls"`
	ActiveIptvUrl  *string `json:"active_iptv_url"`
	IptvEnabled    *bool   `json:"iptv_enabled"`
	AgendaEvents   *string `json:"agenda_events"`
	WhatsappLink   *string `json:"whatsapp_link"`
	TiktokLink     *string `json:"tiktok_link"`
}

// UpdateChannel updates the user's channel
func UpdateChannel(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req UpdateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found for this user"})
	}

	// Update fields if provided
	if req.Title != nil {
		channel.Name = *req.Title // Or stream title depending on mapping
	}
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Description != nil {
		channel.Description = req.Description
	}
	if req.Category != nil {
		channel.Category = req.Category
	}
	if req.IptvUrls != nil {
		channel.IptvUrls = *req.IptvUrls
	}
	if req.ActiveIptvUrl != nil {
		channel.ActiveIptvUrl = req.ActiveIptvUrl
	}
	if req.IptvEnabled != nil {
		channel.IptvEnabled = *req.IptvEnabled
	}
	if req.AgendaEvents != nil {
		channel.AgendaEvents = *req.AgendaEvents
	}
	if req.WhatsappLink != nil {
		channel.WhatsappLink = req.WhatsappLink
	}
	if req.TiktokLink != nil {
		channel.TiktokLink = req.TiktokLink
	}

	database.DB.Save(&channel)

	return c.JSON(fiber.Map{
		"message": "Channel updated successfully",
		"channel": channel,
	})
}

// GetMyChannel returns the channel of the currently authenticated user
func GetMyChannel(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}
	return c.JSON(channel)
}

