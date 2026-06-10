package workers

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/hibiken/asynq"
)

const (
	TypeRestreamStart = "restream:start"
)

// RestreamPayload represents the payload for the restream task
type RestreamPayload struct {
	StreamID   uint   `json:"stream_id"`
	TargetURL  string `json:"target_url"`
	StreamName string `json:"stream_name"`
}

var Client *asynq.Client

// getRedisOpt parses the redis URL
func getRedisOpt() asynq.RedisClientOpt {
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		// Fallback for local development
		redisUrl = "redis://localhost:6376/0"
	}

	// Simple parser (asynq.ParseRedisURI handles full URIs well)
	opt, err := asynq.ParseRedisURI(redisUrl)
	if err != nil {
		// Fallback to simple host port if it fails
		hostPort := strings.Replace(redisUrl, "redis://", "", 1)
		hostPort = strings.Split(hostPort, "/")[0]
		return asynq.RedisClientOpt{Addr: hostPort}
	}
	return opt.(asynq.RedisClientOpt)
}

// InitAsynqClient initializes the Asynq Client used by the web API to enqueue tasks
func InitAsynqClient() {
	Client = asynq.NewClient(getRedisOpt())
	log.Println("Asynq Client connected to Redis")
}

// EnqueueRestreamTask sends a task to the background queue
func EnqueueRestreamTask(streamID uint, targetURL, streamName string) error {
	payload, err := json.Marshal(RestreamPayload{
		StreamID:   streamID,
		TargetURL:  targetURL,
		StreamName: streamName,
	})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeRestreamStart, payload)
	info, err := Client.Enqueue(task)
	if err != nil {
		return err
	}
	log.Printf("Enqueued background task: id=%s queue=%s", info.ID, info.Queue)
	return nil
}

// HandleRestreamStartTask actually processes the background job (executed by the worker server)
func HandleRestreamStartTask(ctx context.Context, t *asynq.Task) error {
	var p RestreamPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err // Return err to retry later
	}

	log.Printf("[Celery Replacement] Starting heavy restream job for %s to %s", p.StreamName, p.TargetURL)
	// Aquí ejecutaríamos FFmpeg vía os.Exec para enviar el video a YouTube/Twitch
	// de forma idéntica a como lo hacía Celery en Python.
	
	return nil
}

// RunWorkerServer starts the consumer that processes background tasks
func RunWorkerServer() {
	srv := asynq.NewServer(
		getRedisOpt(),
		asynq.Config{
			Concurrency: 10,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeRestreamStart, HandleRestreamStartTask)

	log.Println("Starting Asynq Background Worker (Celery Replacement)...")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
