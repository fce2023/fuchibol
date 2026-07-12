package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ChatMessagePayload defines the incoming JSON structure
type ChatMessagePayload struct {
	ID        uint   `json:"id,omitempty"`
	Type      string `json:"type,omitempty"` // "chat", "viewers"
	ChannelID uint   `json:"channel_id"`
	Content   string `json:"content"`
	UserID    uint   `json:"user_id"` // Normally from JWT, passing directly for simplicity in this demo
	Username  string `json:"username"`
	Viewers   int    `json:"viewers,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	rooms      map[uint]map[*websocket.Conn]bool
	register   chan *clientRegistration
	unregister chan *clientRegistration
	broadcast  chan ChatMessagePayload
	mu         sync.RWMutex
}

type clientRegistration struct {
	conn      *websocket.Conn
	channelID uint
}

var ChatHub = Hub{
	rooms:      make(map[uint]map[*websocket.Conn]bool),
	register:   make(chan *clientRegistration),
	unregister: make(chan *clientRegistration),
	broadcast:  make(chan ChatMessagePayload),
}

func broadcastViewers(h *Hub, channelID uint) {
	h.mu.RLock()
	count := len(h.rooms[channelID])
	h.mu.RUnlock()
	
	// Sync to Redis for the Python restreamer optimization
	if database.RedisClient != nil {
		database.RedisClient.Set(database.Ctx, fmt.Sprintf("channel:%d:viewers", channelID), count, 0)
	}

	msg := ChatMessagePayload{
		Type:      "viewers",
		ChannelID: channelID,
		Viewers:   count,
	}
	
	for conn := range h.rooms[channelID] {
		conn.WriteJSON(msg)
	}
}

// BroadcastType sends a system-level event to all clients in a specific channel
func (h *Hub) BroadcastType(channelID uint, msgType string, content string) {
	msg := ChatMessagePayload{
		Type:      msgType,
		ChannelID: channelID,
		Content:   content,
	}
	h.broadcast <- msg
}

// Run starts the chat hub loop
func (h *Hub) Run() {
	for {
		select {
		case req := <-h.register:
			h.mu.Lock()
			if h.rooms[req.channelID] == nil {
				h.rooms[req.channelID] = make(map[*websocket.Conn]bool)
			}
			h.rooms[req.channelID][req.conn] = true
			h.mu.Unlock()
			log.Printf("Nuevo cliente WebSocket conectado al canal %d", req.channelID)
			broadcastViewers(h, req.channelID)

		case req := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.rooms[req.channelID][req.conn]; ok {
				delete(h.rooms[req.channelID], req.conn)
				req.conn.Close()
				log.Printf("Cliente desconectado del canal %d", req.channelID)
			}
			h.mu.Unlock()
			broadcastViewers(h, req.channelID)

		case msg := <-h.broadcast:
			// Guardar el mensaje en BD
			if msg.Type == "" || msg.Type == "chat" {
				dbMsg := models.ChatMessage{
					ChannelID: msg.ChannelID,
					UserID:    msg.UserID,
					Content:   msg.Content,
					Timestamp: time.Now(),
				}
				if err := database.DB.Create(&dbMsg).Error; err == nil {
					msg.ID = dbMsg.ID
				} else {
					log.Printf("Error guardando mensaje en BD: %v", err)
				}
			}

			// Enviar (broadcast) a todos los usuarios conectados a esta sala/canal
			h.mu.RLock()
			for conn := range h.rooms[msg.ChannelID] {
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Printf("Error enviando mensaje por WS: %v", err)
					conn.Close()
					// Unregister will be handled by the read loop error
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WebsocketHandler upgrades the HTTP connection to a WebSocket
func WebsocketHandler(c *websocket.Conn) {
	channelIDStr := c.Query("channel")
	channelID64, _ := strconv.ParseUint(channelIDStr, 10, 32)
	channelID := uint(channelID64)

	if channelID == 0 {
		c.Close()
		return
	}

	req := &clientRegistration{conn: c, channelID: channelID}
	ChatHub.register <- req

	defer func() {
		ChatHub.unregister <- req
	}()

	// Infinite loop to read incoming messages
	for {
		messageType, msgBytes, err := c.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage {
			var payload ChatMessagePayload
			if err := json.Unmarshal(msgBytes, &payload); err == nil {
				payload.ChannelID = channelID
				// Send to Hub for broadcasting
				ChatHub.broadcast <- payload
			}
		}
	}
}

// GetChatHistory fetches the last 50 chat messages for a channel
func GetChatHistory(c *fiber.Ctx) error {
	channelIDStr := c.Params("id")
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	var dbMessages []models.ChatMessage
	err = database.DB.Preload("User").
		Where("channel_id = ? AND is_deleted = ?", uint(channelID), false).
		Order("timestamp desc").
		Limit(50).
		Find(&dbMessages).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch chat history",
		})
	}

	// Reverse the order so they are chronologically ordered (oldest first)
	for i, j := 0, len(dbMessages)-1; i < j; i, j = i+1, j-1 {
		dbMessages[i], dbMessages[j] = dbMessages[j], dbMessages[i]
	}

	return c.JSON(dbMessages)
}

// ClearChatHistory deletes (soft-deletes) all messages for a channel
func ClearChatHistory(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channelIDStr := c.Params("id")
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	// Retrieve the channel to check permissions
	var channel models.Channel
	if err := database.DB.Where("id = ?", uint(channelID)).First(&channel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	// Fetch user details to check role
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	// Check authorization: must be the channel owner or an admin
	if channel.UserID != userID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: You do not own this channel"})
	}

	// Soft delete all active messages for this channel
	err = database.DB.Model(&models.ChatMessage{}).
		Where("channel_id = ? AND is_deleted = ?", uint(channelID), false).
		Updates(map[string]interface{}{
			"is_deleted":      true,
			"deleted_by":      userID,
			"deletion_reason": "Cleared by admin",
		}).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to clear chat history",
		})
	}

	// Broadcast chat_cleared event to notify connected clients in real-time
	ChatHub.BroadcastType(uint(channelID), "chat_cleared", "El chat ha sido limpiado por el moderador.")

	return c.JSON(fiber.Map{
		"message": "Chat cleared successfully",
	})
}

// DeleteChatMessage deletes (soft-deletes) a single message
func DeleteChatMessage(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	channelIDStr := c.Params("id")
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid channel ID"})
	}

	msgIDStr := c.Params("msgId")
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
	}

	// Fetch message
	var msg models.ChatMessage
	if err := database.DB.Where("id = ? AND channel_id = ?", uint(msgID), uint(channelID)).First(&msg).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Message not found"})
	}

	// Fetch channel to verify ownership
	var channel models.Channel
	if err := database.DB.Where("id = ?", uint(channelID)).First(&channel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Channel not found"})
	}

	// Fetch requesting user to verify roles
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	// Check permission: channel owner, message sender, admin, or mod
	isOwner := channel.UserID == userID
	isSender := msg.UserID == userID
	isAdminOrMod := user.Role == "admin" || user.Role == "mod"
	if !isOwner && !isSender && !isAdminOrMod {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Cannot delete this message"})
	}

	// Soft delete the message
	err = database.DB.Model(&msg).Updates(map[string]interface{}{
		"is_deleted":      true,
		"deleted_by":      userID,
		"deletion_reason": "Deleted by moderator/admin",
	}).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete message"})
	}

	// Notify connected websocket clients
	msgPayload := ChatMessagePayload{
		Type:      "message_deleted",
		ChannelID: uint(channelID),
		Content:   strconv.FormatUint(msgID, 10),
	}
	ChatHub.broadcast <- msgPayload

	return c.JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}



