package controllers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	LogoUrl        *string `json:"logo_url"`
	YapeNumber     *string `json:"yape_number"`
	PaypalLink     *string `json:"paypal_link"`
	DonationMessage *string `json:"donation_message"`
	DonationLongMessage *string `json:"donation_long_message"`
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

	channel, err := GetOrCreateChannel(userID)
	if err != nil {
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
	if req.LogoUrl != nil {
		channel.LogoUrl = req.LogoUrl
	}
	if req.YapeNumber != nil {
		channel.YapeNumber = req.YapeNumber
	}
	if req.PaypalLink != nil {
		channel.PaypalLink = req.PaypalLink
	}
	if req.DonationMessage != nil {
		channel.DonationMessage = req.DonationMessage
	}
	if req.DonationLongMessage != nil {
		channel.DonationLongMessage = req.DonationLongMessage
	}

	database.DB.Save(channel)

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
	channel, err := GetOrCreateChannel(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}
	return c.JSON(channel)
}


// UploadLogo handles logo image upload, converts to webp, and saves it
func UploadLogo(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channel, err := GetOrCreateChannel(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to retrieve file"})
	}

	// Create directories if they don't exist
	uploadDir := "/app/uploads/logos"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create directory"})
	}

	// Temporary file path for the uploaded image
	tempPath := filepath.Join(uploadDir, fmt.Sprintf("temp_%d_%s", channel.ID, file.Filename))
	
	// Target WebP file path
	webpFilename := fmt.Sprintf("channel_%d.webp", channel.ID)
	webpPath := filepath.Join(uploadDir, webpFilename)

	// Save the uploaded file temporarily
	if err := c.SaveFile(file, tempPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	// Convert to WebP using cwebp
	cmd := exec.Command("cwebp", "-q", "80", tempPath, "-o", webpPath)
	if err := cmd.Run(); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process image optimization (cwebp)"})
	}

	// Clean up temp file after successful conversion
	os.Remove(tempPath)

	// Update Database
	logoUrl := fmt.Sprintf("/uploads/logos/%s", webpFilename)
	channel.LogoUrl = &logoUrl
	database.DB.Save(channel)

	return c.JSON(fiber.Map{
		"message": "Logo uploaded successfully",
		"logo_url": logoUrl,
	})
}
