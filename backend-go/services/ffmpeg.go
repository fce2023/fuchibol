package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/models"
)

type RestreamProcess struct {
	Cmd        *exec.Cmd
	CancelFunc context.CancelFunc
}

var (
	activeRestreams = make(map[uint]*RestreamProcess)
	mu              sync.Mutex
)

// StartRestream starts the FFmpeg process for a given channel and IPTV URL
func StartRestream(channelID uint, iptvURL string) error {
	// If already running, stop it first to restart with new URL
	mu.Lock()
	_, exists := activeRestreams[channelID]
	mu.Unlock()
	
	if exists {
		StopRestream(channelID)
	}

	mu.Lock()
	defer mu.Unlock()

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = "fuchibol_super_secret_key_change_me_in_prod" // fallback
	}

	// Wait, SRS is at rtmp://srs:1935 internally (if docker compose) or localhost.
	// Since backend runs in docker network, "srs" is the hostname.
	rtmpUrl := fmt.Sprintf("rtmp://srs:1935/live/iptv_%d?vhost=__defaultVhost__&internal_secret=%s", channelID, secretKey)

	ctx, cancel := context.WithCancel(context.Background())

	go ffmpegWorker(ctx, channelID, iptvURL, rtmpUrl)

	activeRestreams[channelID] = &RestreamProcess{
		CancelFunc: cancel,
	}

	return nil
}

// StopRestream kills the FFmpeg process for a given channel
func StopRestream(channelID uint) {
	mu.Lock()
	defer mu.Unlock()

	if process, exists := activeRestreams[channelID]; exists {
		log.Printf("[RESTREAM] Stopping FFmpeg for channel %d", channelID)
		process.CancelFunc()
		delete(activeRestreams, channelID)
	}
}

// StatusRestream returns if the ffmpeg process is active
func StatusRestream(channelID uint) bool {
	mu.Lock()
	defer mu.Unlock()
	_, exists := activeRestreams[channelID]
	return exists
}

func ffmpegWorker(ctx context.Context, channelID uint, reqUrl, rtmpUrl string) {
	retries := 0
	maxRetries := 100

	log.Printf("[RESTREAM] Starting worker for channel %d", channelID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[RESTREAM] Worker loop finished for channel %d (context canceled)", channelID)
			return
		default:
		}

		logFile, err := os.OpenFile(fmt.Sprintf("/tmp/ffmpeg_%d.log", channelID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("[RESTREAM] Failed to open log file: %v", err)
			return
		}

		logFile.WriteString(fmt.Sprintf("\n--- Starting FFmpeg restream for channel %d (Attempt %d) ---\n", channelID, retries+1))

		args := []string{
			"-y",
			// Standard network resilience + intentional massive input buffer for ingest latency
			"-reconnect", "1", "-reconnect_at_eof", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "10",
			"-http_persistent", "1",
			"-thread_queue_size", "10240",
			"-i", reqUrl,
			
			// COPY mode: Zero CPU usage. Bypassing CORS only.
			"-c:v", "copy",
			"-c:a", "copy",
			
			// Sometimes AAC from HLS needs a bitstream filter to go into FLV
			"-bsf:a", "aac_adtstoasc",
			
			// Output format
			"-f", "flv", rtmpUrl,
		}

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		log.Printf("[RESTREAM] Executing FFmpeg for channel %d", channelID)
		
		// Update the active process in map
		mu.Lock()
		if process, exists := activeRestreams[channelID]; exists {
			process.Cmd = cmd
		}
		mu.Unlock()

		err = cmd.Run()
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = -1
			}
		}

		logFile.WriteString(fmt.Sprintf("\n--- FFmpeg exited with code %d ---\n", exitCode))
		logFile.Close()

		select {
		case <-ctx.Done():
			// User manually stopped it
			log.Printf("[RESTREAM] Worker loop finished for channel %d (user stopped)", channelID)
			
			// Update DB to offline if OBS is not live
			var channel models.Channel
			if err := database.DB.Where("id = ?", channelID).First(&channel).Error; err == nil {
				if !channel.IsLive {
					// In python it set is_live = False. But if OBS is live, IsLive is already true
					// Actually we only update if needed.
				}
			}
			return
		default:
		}

		// Process died unexpectedly, retry
		retries++
		if retries >= maxRetries {
			log.Printf("[RESTREAM] Max retries reached for channel %d", channelID)
			StopRestream(channelID) // Clean up map
			return
		}

		log.Printf("[RESTREAM] FFmpeg process died for channel %d, restarting in 5s...", channelID)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
