package controllers

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/websocket/v2"
)

// ChatMessagePayload defines the incoming JSON structure
type ChatMessagePayload struct {
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
	defer h.mu.RUnlock()
	count := len(h.rooms[channelID])
	
	msg := ChatMessagePayload{
		Type:      "viewers",
		ChannelID: channelID,
		Viewers:   count,
	}
	
	for conn := range h.rooms[channelID] {
		conn.WriteJSON(msg)
	}
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
			// Guardar el mensaje en BD de forma asíncrona sin bloquear el chat
			if msg.Type == "" || msg.Type == "chat" {
				go func(m ChatMessagePayload) {
					dbMsg := models.ChatMessage{
						ChannelID: m.ChannelID,
						UserID:    m.UserID,
						Content:   m.Content,
					}
					database.DB.Create(&dbMsg)
				}(msg)
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
