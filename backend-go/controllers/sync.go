package controllers

import (
	"fmt"
	"os"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"github.com/gofiber/fiber/v2"
)

// GetPrimaryChannel returns a main live channel to display on Home
func GetPrimaryChannel(c *fiber.Ctx) error {
	var channel models.Channel
	// Devuelve el primer canal que esté en vivo o con IPTV activado
	if err := database.DB.Preload("User").Where("is_live = ? OR iptv_enabled = ?", true, true).First(&channel).Error; err != nil {
		// Fallback al canal ID 1 si no hay nadie en vivo
		database.DB.Preload("User").First(&channel, 1)
	}
	username := ""
	if channel.User != nil {
		username = channel.User.Username
	}
	return c.JSON(fiber.Map{
		"id":          channel.ID,
		"username":    username,
		"name":        channel.Name,
		"description": channel.Description,
		"category":    channel.Category,
		"is_live":     channel.IsLive || channel.IptvEnabled,
	})
}

// GetChannelByUsername returns a channel given the owner's username
func GetChannelByUsername(c *fiber.Ctx) error {
	username := c.Params("username")
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	var channel models.Channel
	if err := database.DB.Preload("User").Where("user_id = ?", user.ID).First(&channel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}
	return c.JSON(channel)
}

// GetPlaybackURL returns the HLS/WebRTC URL for the video player
func GetPlaybackURL(c *fiber.Ctx) error {
	id := c.Params("id")
	var channel models.Channel
	if err := database.DB.First(&channel, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	hlsBaseUrl := os.Getenv("HLS_BASE_URL")
	if hlsBaseUrl == "" {
		hlsBaseUrl = "/hls"
	}

	var playbackUrl string
	streamName := channel.ActiveStreamName
	if streamName == "" {
		streamName = fmt.Sprintf("channel_%d", channel.ID)
	}

	if channel.IsLive {
		playbackUrl = fmt.Sprintf("%s%s/live/%s.m3u8", c.BaseURL(), hlsBaseUrl, streamName)
	} else if channel.IptvEnabled && channel.ActiveIptvUrl != nil && *channel.ActiveIptvUrl != "" {
		playbackUrl = fmt.Sprintf("%s/api/v1/proxy/m3u8/%d", c.BaseURL(), channel.ID)
	} else {
		playbackUrl = fmt.Sprintf("%s%s/offline/offline.m3u8", c.BaseURL(), hlsBaseUrl)
	}

	streamType := "hls"
	if channel.IsLive {
		streamType = "webrtc"
	}

	return c.JSON(fiber.Map{
		"playback_url": playbackUrl,
		"is_live":      channel.IsLive || channel.IptvEnabled,
		"stream_type":  streamType,
		"hls":          playbackUrl,
		"webrtc":       fmt.Sprintf("webrtc://%s/live/%s", c.Hostname(), streamName),
	})
}

type RestreamStartReq struct {
	IptvUrl string `json:"iptv_url"`
}

// RestreamStatus returns the current status of the restream task
func RestreamStatus(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	// Calculate target (IPTV name) if active
	target := ""
	if channel.IptvEnabled {
		target = "Decodificando señal de forma nativa..."
	}

	isPausedByObs := channel.IsLive && channel.IptvEnabled

	return c.JSON(fiber.Map{
		"is_active_restream": channel.IptvEnabled,
		"is_paused_by_obs":   isPausedByObs,
		"target":             target,
	})
}

// RestreamStart enqueues a background job to start restreaming
func RestreamStart(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req RestreamStartReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	// Update channel status
	channel.IptvEnabled = true
	if req.IptvUrl != "" {
		channel.ActiveIptvUrl = &req.IptvUrl
	}
	database.DB.Save(&channel)

	// Notify all viewers via WebSocket to reload the player
	ChatHub.BroadcastType(channel.ID, "stream_reload", "La señal ha cambiado")

	return c.JSON(fiber.Map{
		"message": "Restream proxy nativo iniciado",
	})
}

// RestreamStop kills the background restream job
func RestreamStop(c *fiber.Ctx) error {
	userID := ExtractUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var channel models.Channel
	if err := database.DB.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Channel not found"})
	}

	channel.IptvEnabled = false
	channel.IsLive = false
	database.DB.Save(&channel)

	return c.JSON(fiber.Map{
		"message": "Restream detenido",
	})
}

// GetFollowing returns channels followed by the user
func GetFollowing(c *fiber.Ctx) error {
        channelID := c.Params("id")
        userID := ExtractUserID(c)

        var followerCount int64
        database.DB.Model(&models.Follower{}).Where("channel_id = ?", channelID).Count(&followerCount)

        isFollowing := false
        if userID != 0 {
                var f models.Follower
                if err := database.DB.Where("follower_id = ? AND channel_id = ?", userID, channelID).First(&f).Error; err == nil {
                        isFollowing = true
                }
        }

        return c.JSON(fiber.Map{
                "is_following":   isFollowing,
                "follower_count": followerCount,
        })
}

// FollowChannel allows a user to follow a channel
func FollowChannel(c *fiber.Ctx) error {
        channelID := c.Params("id")
        userID := ExtractUserID(c)
        if userID == 0 {
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
        }

        var chID uint
        if _, err := fmt.Sscanf(channelID, "%d", &chID); err != nil {
                return c.Status(400).JSON(fiber.Map{"error": "Invalid channel id"})
        }

        f := models.Follower{
                FollowerID: userID,
                ChannelID:  chID,
                CreatedAt:  time.Now(),
        }
        // Save will attempt to insert, if duplicate it might error but that's fine (user is already following)
        database.DB.Create(&f)

        return c.JSON(fiber.Map{"success": true})
}

// UnfollowChannel allows a user to unfollow a channel
func UnfollowChannel(c *fiber.Ctx) error {
        channelID := c.Params("id")
        userID := ExtractUserID(c)
        if userID == 0 {
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
        }

        database.DB.Where("follower_id = ? AND channel_id = ?", userID, channelID).Delete(&models.Follower{})

        return c.JSON(fiber.Map{"success": true})
}

// GetFollowersList returns the users who follow a channel
func GetFollowersList(c *fiber.Ctx) error {
	channelID := c.Params("id")
	var followers []models.Follower
	
	if err := database.DB.Preload("Follower").Where("channel_id = ?", channelID).Find(&followers).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch followers"})
	}

	var results []fiber.Map
	for _, f := range followers {
		if f.Follower != nil {
			results = append(results, fiber.Map{
				"id":       f.Follower.ID,
				"username": f.Follower.Username,
			})
		}
	}
	
	// Si la lista está vacía, devuelve un array vacío en vez de null
	if results == nil {
		results = []fiber.Map{}
	}

	return c.JSON(results)
}

