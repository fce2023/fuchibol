package controllers

import (
	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

// ListUsers gets all users (admin only)
func ListUsers(c *fiber.Ctx) error {
	// TODO: Add Admin role verification middleware here

	var users []models.User
	database.DB.Select("id", "email", "username", "role", "is_banned", "created_at").Find(&users)
	
	return c.JSON(fiber.Map{
		"users": users,
	})
}

// BanUser blocks a user from using the platform
func BanUser(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	user.IsBanned = true
	database.DB.Save(&user)

	return c.JSON(fiber.Map{
		"message": "User banned successfully",
		"user": user.Username,
	})
}
