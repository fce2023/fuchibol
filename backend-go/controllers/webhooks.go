package controllers

import (
	"log"
	"os"
	"strings"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"fuchibol-backend-go/services"
	"github.com/gofiber/fiber/v2"
)

// SrsWebhookRequest represents the payload from SRS
type SrsWebhookRequest struct {
	Action   string `json:"action"`
	ClientID string `json:"client_id"`
	IP       string `json:"ip"`
	Vhost    string `json:"vhost"`
	App      string `json:"app"`
	TcUrl    string `json:"tcUrl"`
	Stream   string `json:"stream"`
	Param    string `json:"param"`
}

// OnPublish handles SRS on_publish webhook
func OnPublish(c *fiber.Ctx) error {
	var req SrsWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON")
	}

	log.Printf("SRS on_publish: %s from %s (params: %s)", req.Stream, req.IP, req.Param)

	var channel models.Channel

	// Check if it's an internal IPTV stream
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		secret = "fuchibol_super_secret_key_change_me_in_prod"
	}

	if strings.Contains(req.Param, "internal_secret="+secret) && strings.HasPrefix(req.Stream, "iptv_") {
		// Parse channel ID from iptv_{ID}
		idStr := strings.TrimPrefix(req.Stream, "iptv_")
		if err := database.DB.First(&channel, idStr).Error; err != nil {
			log.Printf("Internal IPTV Channel %s not found", idStr)
			return c.SendString("0")
		}
	} else {
		// External OBS stream
		// The panel might send "channel_1?key=live_..."
		// SRS puts "channel_1" in req.Stream and "?key=live_..." in req.Param
		rawKey := ""
		if req.Param != "" {
			params := strings.TrimPrefix(req.Param, "?")
			pairs := strings.Split(params, "&")
			for _, pair := range pairs {
				kv := strings.Split(pair, "=")
				if len(kv) == 2 && kv[0] == "key" {
					rawKey = kv[1]
					break
				}
			}
		}

		if rawKey == "" {
			rawKey = req.Stream
		}

		hashedKey := HashStreamKey(rawKey)
		if err := database.DB.Where("stream_key_hash = ?", hashedKey).First(&channel).Error; err != nil {
			log.Printf("OBS Channel with stream key hash %s not found (raw was: %s)", hashedKey, rawKey)
			return c.Status(401).SendString("Unauthorized")
		}

		// If this is an OBS stream and there's an active IPTV restream, stop it
		if services.StatusRestream(channel.ID) {
			log.Printf("OBS starting for channel %d, stopping IPTV restream", channel.ID)
			services.StopRestream(channel.ID)
		}
	}

	isObsStream := !strings.HasPrefix(req.Stream, "iptv_")

	if isObsStream {
		// Update channel status
		channel.IsLive = true
		channel.ActiveStreamName = req.Stream
		database.DB.Save(&channel)

		// Create a new stream record
		newStream := models.Stream{
			ChannelID: channel.ID,
			Title:     channel.Name + " Live Stream", // Default title
			Status:    "live",
			StartTime: time.Now(),
			CreatedAt: time.Now(),
		}
		database.DB.Create(&newStream)
	} else {
		// Even for IPTV restream, it's good to track the stream name
		channel.ActiveStreamName = req.Stream
		database.DB.Save(&channel)
	}

	// Return "0" to tell SRS it's authorized
	return c.SendString("0")
}

// OnUnpublish handles SRS on_unpublish webhook
func OnUnpublish(c *fiber.Ctx) error {
	var req SrsWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON")
	}

	log.Printf("SRS on_unpublish: %s from %s (params: %s)", req.Stream, req.IP, req.Param)

	var channel models.Channel
	found := false

	// Check if it's an internal IPTV stream
	if strings.HasPrefix(req.Stream, "iptv_") {
		idStr := strings.TrimPrefix(req.Stream, "iptv_")
		if err := database.DB.First(&channel, idStr).Error; err == nil {
			found = true
		}
	} else {
		rawKey := ""
		if req.Param != "" {
			params := strings.TrimPrefix(req.Param, "?")
			pairs := strings.Split(params, "&")
			for _, pair := range pairs {
				kv := strings.Split(pair, "=")
				if len(kv) == 2 && kv[0] == "key" {
					rawKey = kv[1]
					break
				}
			}
		}

		if rawKey == "" {
			rawKey = req.Stream
		}

		hashedKey := HashStreamKey(rawKey)
		if err := database.DB.Where("stream_key_hash = ?", hashedKey).First(&channel).Error; err == nil {
			found = true
		}
	}

	if found {
		// Only mark IsLive as false if it's an OBS stream
		// (We don't want an IPTV stream disconnection to mark the whole channel offline if OBS is still live somehow,
		// though normally they are mutually exclusive)
		isObsStream := !strings.HasPrefix(req.Stream, "iptv_")
		
		if isObsStream {
			channel.IsLive = false
			channel.ActiveStreamName = ""
			database.DB.Save(&channel)

			// Find the active stream and mark it ended
			var activeStream models.Stream
			if err := database.DB.Where("channel_id = ? AND status = ?", channel.ID, "live").Order("created_at desc").First(&activeStream).Error; err == nil {
				activeStream.Status = "ended"
				now := time.Now()
				activeStream.EndTime = &now
				database.DB.Save(&activeStream)
			}
			
			// Auto-restart IPTV if OBS disconnected and IPTV was enabled
			if channel.IptvEnabled && channel.ActiveIptvUrl != nil {
				log.Printf("OBS disconnected for channel %d, auto-restarting IPTV restream", channel.ID)
				services.StartRestream(channel.ID, *channel.ActiveIptvUrl)
			}
		} else {
			// For IPTV unpublish, if we are clearing the stream name, make sure it's the one we expect
			if channel.ActiveStreamName == req.Stream {
				channel.ActiveStreamName = ""
				database.DB.Save(&channel)
			}
		}
	}

	return c.SendString("0")
}
