package controllers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(getEnv("SECRET_KEY", "fuchibol_super_secret_key_change_me_in_prod"))

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func generateStreamKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return "live_" + hex.EncodeToString(bytes)
}

func HashStreamKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ExtractUserID helper to parse JWT from Authorization header
func ExtractUserID(c *fiber.Ctx) uint {
	tokenString := c.Get("Authorization")
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	} else {
		return 0
	}

	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if idFloat, ok := claims["user_id"].(float64); ok {
			return uint(idFloat)
		}
	}
	return 0
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register registers a new user
func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user := models.User{
		Email:          req.Email,
		Username:       req.Username,
		HashedPassword: string(hashedPassword),
		Role:           "user",
	}

	if result := database.DB.Create(&user); result.Error != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email or Username already exists"})
	}

	// Create an empty channel for the user with an initial stream key
	rawKey := generateStreamKey()
	channel := models.Channel{
		UserID:        user.ID,
		Name:          user.Username + " Channel",
		StreamKeyHash: HashStreamKey(rawKey),
		KeyExpiresAt:  time.Now().AddDate(0, 3, 0), // 90 days
		UpdatedAt:     time.Now(),
	}
	if err := database.DB.Create(&channel).Error; err != nil {
		log.Printf("Error creating channel during registration: %v", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}

// Login authenticates a user
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var user models.User
	if err := database.DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not login"})
	}

	return c.JSON(fiber.Map{
		"access_token": t,
		"token_type":   "bearer",
	})
}

// Me returns the current authenticated user
func Me(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	if err := database.DB.Select("id", "username", "email", "role").First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

// RotateStreamKey generates a new stream key for OBS
func RotateStreamKey(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channel, err := GetOrCreateChannel(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	rawKey := generateStreamKey()
	channel.StreamKeyHash = HashStreamKey(rawKey)
	channel.KeyExpiresAt = time.Now().AddDate(0, 3, 0) // 90 days

	database.DB.Save(channel)

	return c.JSON(fiber.Map{
		"stream_key": rawKey,
		"expires_at": channel.KeyExpiresAt,
	})
}

// GetOrCreateChannel retrieves the user's channel or creates one if it doesn't exist (self-healing)
func GetOrCreateChannel(userID uint) (*models.Channel, error) {
	var channel models.Channel
	err := database.DB.Where("user_id = ?", userID).First(&channel).Error
	if err == nil {
		return &channel, nil
	}

	// Channel doesn't exist, create it
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	rawKey := generateStreamKey()
	channel = models.Channel{
		UserID:        userID,
		Name:          user.Username + " Channel",
		StreamKeyHash: HashStreamKey(rawKey),
		KeyExpiresAt:  time.Now().AddDate(0, 3, 0), // 90 days
		UpdatedAt:     time.Now(),
	}

	if err := database.DB.Create(&channel).Error; err != nil {
		return nil, err
	}

	return &channel, nil
}
