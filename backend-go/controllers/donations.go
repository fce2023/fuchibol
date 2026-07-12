package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

type ReportDonationRequest struct {
	DonorName     string  `json:"donor_name"`
	Amount        float64 `json:"amount"`
	Method        string  `json:"method"`
	ReferenceCode string  `json:"reference_code"`
	ReceiptUrl    *string `json:"receipt_url"`
}

// ReportDonation registers a pending user donation report
func ReportDonation(c *fiber.Ctx) error {
	channelIDStr := c.Params("id")
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	var req ReportDonationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.DonorName == "" {
		req.DonorName = "Anónimo"
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Amount must be greater than zero"})
	}
	if req.Method != "yape" && req.Method != "paypal" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid donation method"})
	}

	donation := models.Donation{
		ChannelID:     uint(channelID),
		DonorName:     req.DonorName,
		Amount:        req.Amount,
		Method:        req.Method,
		ReferenceCode: req.ReferenceCode,
		ReceiptUrl:    req.ReceiptUrl,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := database.DB.Create(&donation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to record donation report"})
	}

	return c.JSON(fiber.Map{
		"message":  "Donation report submitted successfully. It will be shown in the ranking once approved.",
		"donation": donation,
	})
}

type RankingResult struct {
	DonorName   string  `json:"donor_name"`
	TotalAmount float64 `json:"total_amount"`
}

// GetDonationRanking returns the top 10 approved donors grouped by donor name
func GetDonationRanking(c *fiber.Ctx) error {
	channelIDStr := c.Params("id")
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	var results []RankingResult
	err = database.DB.Model(&models.Donation{}).
		Select("donor_name, SUM(amount) as total_amount").
		Where("channel_id = ? AND status = ?", uint(channelID), "approved").
		Group("donor_name").
		Order("total_amount desc").
		Limit(10).
		Scan(&results).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch donation ranking"})
	}

	return c.JSON(results)
}

// GetAdminDonations fetches all donation reports for the streamer's channel
func GetAdminDonations(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channel, err := GetOrCreateChannel(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	var donations []models.Donation
	if err := database.DB.Where("channel_id = ?", channel.ID).Order("created_at desc").Find(&donations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch donation reports"})
	}

	return c.JSON(donations)
}

type ReviewDonationRequest struct {
	Status string `json:"status"` // approved, rejected
}

// ReviewDonation updates the status of a donation report
func ReviewDonation(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channel, err := GetOrCreateChannel(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	donationIDStr := c.Params("donationId")
	donationID, err := strconv.ParseUint(donationIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid donation ID"})
	}

	var req ReviewDonationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Status != "approved" && req.Status != "rejected" && req.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid status value"})
	}

	var donation models.Donation
	if err := database.DB.Where("id = ? AND channel_id = ?", uint(donationID), channel.ID).First(&donation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Donation report not found"})
	}

	donation.Status = req.Status
	if err := database.DB.Save(&donation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update donation report"})
	}

	return c.JSON(fiber.Map{
		"message":  "Donation report status updated successfully",
		"donation": donation,
	})
}

// UploadDonationReceipt handles receipt image upload
func UploadDonationReceipt(c *fiber.Ctx) error {
	channelIDStr := c.Params("id")
	_, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	file, err := c.FormFile("receipt")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to retrieve file"})
	}

	// Create receipts directory
	uploadDir := "/app/uploads/receipts"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create receipts directory"})
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("receipt_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save receipt file"})
	}

	return c.JSON(fiber.Map{
		"receipt_url": fmt.Sprintf("/uploads/receipts/%s", filename),
	})
}
