package database

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6376/0"
	}

	opts, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Printf("Failed to parse Redis URL: %v. Falling back to default options.", err)
		opts = &redis.Options{
			Addr: "localhost:6376",
		}
	}

	RedisClient = redis.NewClient(opts)

	if err := RedisClient.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully")
}
