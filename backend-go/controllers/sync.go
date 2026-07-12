package controllers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
	"fuchibol-backend-go/services"
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
		playbackUrl = fmt.Sprintf("%s/live/%s.m3u8", hlsBaseUrl, streamName)
	} else if channel.IptvEnabled && channel.ActiveIptvUrl != nil && *channel.ActiveIptvUrl != "" {
		playbackUrl = fmt.Sprintf("/api/v1/proxy/m3u8/%d", channel.ID)
	} else {
		playbackUrl = fmt.Sprintf("%s/offline/offline.m3u8", hlsBaseUrl)
	}

	streamType := "hls"
	if channel.IsLive {
		// Use HLS by default for OBS streams to prioritize stability.
		// Set OBS_STREAM_TECH=webrtc in environment variables to revert to WebRTC.
		if os.Getenv("OBS_STREAM_TECH") == "webrtc" {
			streamType = "webrtc"
		}
	}

	return c.JSON(fiber.Map{
		"playback_url": playbackUrl,
		"is_live":      channel.IsLive || channel.IptvEnabled,
		"stream_type":  streamType,
		"hls":          playbackUrl,
		"webrtc":       fmt.Sprintf("webrtc://%s/live/%s", c.Hostname(), streamName),
	})
}

type StreamState struct {
	LastSignalType      string
	SequenceOffset      int
	LastRawSequence     int
	LastSegmentCount    int
	Discontinuities     []int
}

var (
	stateMutex    sync.Mutex
	channelStates = make(map[uint]*StreamState)
)

var (
	reMediaSeq = regexp.MustCompile(`#EXT-X-MEDIA-SEQUENCE:(\d+)`)
	reExtInf   = regexp.MustCompile(`#EXTINF:`)
)

func GetUnifiedManifest(c *fiber.Ctx) error {
	id := c.Params("id")
	var channel models.Channel
	if err := database.DB.First(&channel, id).Error; err != nil {
		return c.Status(404).SendString("Channel not found")
	}

	// Public base URL (scheme+host) as seen behind the reverse proxy, used to
	// build absolute segment URLs for strict players (Chromecast/Smart TV).
	baseURL := strings.TrimSuffix(c.BaseURL(), "/")

	// Prefix used to make bare (relative) OBS segment filenames absolute.
	// Defaults to the raw SRS HLS location; switched to the clean location
	// when the Chromecast-friendly playlist is available.
	obsSegmentPrefix := baseURL + "/hls/live/"

	// Cast mode: the Chromecast/Smart-TV receiver (CAF/Shaka) is stricter than
	// hls.js and rejects our virtual-sequence manifest (huge MEDIA-SEQUENCE and
	// injected discontinuities). For casting we serve the clean ffmpeg playlist
	// verbatim (standard sequence, absolute segment URLs, no state machine) —
	// exactly the plain shape that the working IPTV playlist has.
	if c.Query("cast") == "1" && (channel.IsLive || channel.IptvEnabled) {
		// Pick the clean stream name for the active source. OBS is keyed by its
		// SRS stream name; IPTV by channel_<id>.
		cleanName := fmt.Sprintf("channel_%d", channel.ID)
		if channel.IsLive {
			if channel.ActiveStreamName != "" {
				cleanName = channel.ActiveStreamName
			}
			// Lazily (re)start the OBS clean pipeline if needed.
			if !services.CleanHLSReady(cleanName) {
				services.StartCleanHLS(cleanName)
			}
		}

		// Prefer the clean playlist (SEI-stripped, plain sequence, absolute URLs).
		cleanPath := fmt.Sprintf("%s/%s.m3u8", services.CleanHLSDir, cleanName)
		if contentBytes, fileErr := os.ReadFile(cleanPath); fileErr == nil && len(contentBytes) > 0 {
			lines := strings.Split(string(contentBytes), "\n")
			for i, line := range lines {
				t := strings.TrimSpace(line)
				if t != "" && !strings.HasPrefix(t, "#") &&
					!strings.HasPrefix(t, "http") && !strings.HasPrefix(t, "/") {
					lines[i] = baseURL + "/hls/live_clean/" + t
				}
			}
			c.Set("Content-Type", "application/vnd.apple.mpegurl")
			c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Set("Access-Control-Allow-Origin", "*")
			return c.SendString(strings.Join(lines, "\n"))
		}

		// Fallback for IPTV: serve the plain proxy-rewritten playlist (absolute
		// proxied URLs) — the shape that already casts fine — never the
		// virtual-sequence manifest, which strict receivers reject.
		if !channel.IsLive && channel.IptvEnabled && channel.ActiveIptvUrl != nil && *channel.ActiveIptvUrl != "" {
			if pl, plErr := downloadAndRewriteIPTV(channel.ID, *channel.ActiveIptvUrl, baseURL); plErr == nil {
				c.Set("Content-Type", "application/vnd.apple.mpegurl")
				c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Set("Access-Control-Allow-Origin", "*")
				return c.SendString(pl)
			}
		}
		// OBS clean not ready yet: fall through to normal handling meanwhile.
	}

	// 1. Determine active signal type
	currentSignalType := "offline"
	var rawPlaylist string
	var err error

	if channel.IsLive {
		currentSignalType = "obs"
		// Read OBS playlist from SRS HLS directory
		streamName := channel.ActiveStreamName
		if streamName == "" {
			streamName = fmt.Sprintf("channel_%d", channel.ID)
		}

		// Prefer the clean HLS (SEI stripped) that plays on Chromecast/Smart TV.
		// If the backend restarted while OBS was already live, lazily (re)start
		// the ffmpeg — it's idempotent — and fall back to raw SRS meanwhile.
		if !services.CleanHLSReady(streamName) {
			services.StartCleanHLS(streamName)
		}

		cleanPath := fmt.Sprintf("%s/%s.m3u8", services.CleanHLSDir, streamName)
		rawPath := fmt.Sprintf("/app/srs_hls/live/%s.m3u8", streamName)

		if contentBytes, fileErr := os.ReadFile(cleanPath); fileErr == nil && len(contentBytes) > 0 {
			rawPlaylist = string(contentBytes)
			obsSegmentPrefix = baseURL + "/hls/live_clean/"
		} else if contentBytes, fileErr := os.ReadFile(rawPath); fileErr == nil {
			rawPlaylist = string(contentBytes)
			obsSegmentPrefix = baseURL + "/hls/live/"
		} else {
			// Fallback to offline if neither file exists yet
			currentSignalType = "offline"
		}
	}

	if currentSignalType == "offline" && channel.IptvEnabled && channel.ActiveIptvUrl != nil && *channel.ActiveIptvUrl != "" {
		currentSignalType = "iptv"
		// Download and rewrite IPTV playlist
		rawPlaylist, err = downloadAndRewriteIPTV(channel.ID, *channel.ActiveIptvUrl, c.BaseURL())
		if err != nil {
			// Fallback to offline if IPTV fetch fails
			currentSignalType = "offline"
		}
	}

	if currentSignalType == "offline" {
		// Read offline loop playlist
		path := "/app/hls_offline/offline.m3u8"
		contentBytes, fileErr := os.ReadFile(path)
		if fileErr == nil {
			rawPlaylist = string(contentBytes)
			// Rewrite segments in offline playlist to be served relative to /hls/offline/
			rawPlaylist = strings.ReplaceAll(rawPlaylist, "offline-", "/hls/offline/offline-")
		} else {
			return c.Status(404).SendString("No signal available and offline loop missing")
		}
	}

	// 2. State management for smooth transitions
	stateMutex.Lock()
	state, exists := channelStates[channel.ID]
	if !exists {
		state = &StreamState{
			LastSignalType:   currentSignalType,
			SequenceOffset:   0,
			LastRawSequence:  0,
			LastSegmentCount: 3, // Default fallback count
			Discontinuities:  []int{},
		}
		channelStates[channel.ID] = state
	}

	// Parse raw sequence number
	rawSeq := 0
	seqSubmatch := reMediaSeq.FindStringSubmatch(rawPlaylist)
	if len(seqSubmatch) > 1 {
		rawSeq, _ = strconv.Atoi(seqSubmatch[1])
	}

	// Rebase the virtual sequence when the source playlist changes signal type
	// OR when its raw MEDIA-SEQUENCE goes backwards (the underlying ffmpeg/SRS
	// restarted and reset its numbering). Without the second case the virtual
	// sequence would jump backwards and the player would replay old fragments.
	if state.LastSignalType != currentSignalType || rawSeq < state.LastRawSequence {
		// Calculate the virtual sequence of the last segment served in the previous playlist
		lastVirtualSeqStart := state.LastRawSequence + state.SequenceOffset
		lastVirtualSeqEnd := lastVirtualSeqStart + state.LastSegmentCount - 1

		// The new virtual sequence starts exactly after the last segment served
		nextVirtualSeq := lastVirtualSeqEnd + 1
		state.SequenceOffset = nextVirtualSeq - rawSeq

		// Record discontinuity at the transition boundary
		state.Discontinuities = append(state.Discontinuities, nextVirtualSeq)
		state.LastSignalType = currentSignalType
	}

	state.LastRawSequence = rawSeq
	virtualSeq := rawSeq + state.SequenceOffset
	stateMutex.Unlock()

	// 3. Rewrite playlist with virtual sequence number and discontinuities
	lines := strings.Split(rawPlaylist, "\n")
	var rewrittenLines []string
	segmentIndex := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#EXT-X-MEDIA-SEQUENCE:") {
			rewrittenLines = append(rewrittenLines, fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d", virtualSeq))
			continue
		}

		if strings.HasPrefix(trimmed, "#EXTINF:") {
			currentSegVirtualSeq := virtualSeq + segmentIndex
			// Check if this segment needs a discontinuity tag injected before it
			for _, dSeq := range state.Discontinuities {
				if dSeq == currentSegVirtualSeq {
					rewrittenLines = append(rewrittenLines, "#EXT-X-DISCONTINUITY")
					break
				}
			}
			segmentIndex++
		}

		// Ensure segment URIs are fully-qualified absolute URLs. SRS emits bare
		// relative filenames (e.g. "channel_1-342-....ts"). Strict players like
		// the Chromecast receiver (Shaka) can load the manifest but fail to play
		// unless segments are absolute — matching the IPTV path, which already
		// emits absolute URLs and casts correctly.
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
			!strings.HasPrefix(trimmed, "http") && !strings.HasPrefix(trimmed, "/") {
			line = obsSegmentPrefix + trimmed
		}

		rewrittenLines = append(rewrittenLines, line)
	}

	// Clean up old discontinuities that have slided out of the window
	stateMutex.Lock()
	var activeDiscontinuities []int
	for _, dSeq := range state.Discontinuities {
		if dSeq >= virtualSeq {
			activeDiscontinuities = append(activeDiscontinuities, dSeq)
		}
	}
	state.Discontinuities = activeDiscontinuities
	// Save segment count of currently served playlist
	state.LastSegmentCount = segmentIndex
	stateMutex.Unlock()

	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return c.SendString(strings.Join(rewrittenLines, "\n"))
}

func downloadAndRewriteIPTV(channelID uint, targetUrl string, baseURL string) (string, error) {
	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")

	resp, err := proxyClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	parsedTarget, err := url.Parse(targetUrl)
	if err != nil {
		return "", err
	}

	bodyStr := string(bodyBytes)
	
	// If it's a master playlist, fetch the first nested stream
	if strings.Contains(bodyStr, "#EXT-X-STREAM-INF") {
		lines := strings.Split(bodyStr, "\n")
		nestedUrlStr := ""
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				nestedUrlStr = resolveUrl(parsedTarget, trimmed)
				break
			}
		}
		if nestedUrlStr != "" {
			return downloadAndRewriteIPTV(channelID, nestedUrlStr, baseURL)
		}
	}

	rewrittenPlaylist := RewritePlaylistContent(bodyStr, parsedTarget, baseURL)
	return rewrittenPlaylist, nil
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

