package main

import (
	"log"

	"fuchibol-backend-go/controllers"
	"fuchibol-backend-go/database"
	"fuchibol-backend-go/workers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Initialize Database
	database.ConnectDB()

	// Initialize Background Jobs Client (Celery queue equivalent)
	workers.InitAsynqClient()

	// Arrancar el Hub de Chat (WebSockets) en segundo plano
	go controllers.ChatHub.Run()

	// Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName: "Fuchibol API (Go)",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, HEAD, PUT, DELETE, PATCH",
	}))

	// Health Check Route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Fuchibol Go Backend is running!",
		})
	})

	// API v1 Group
	api := app.Group("/api/v1")
	
	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", controllers.Register)
	auth.Post("/login", controllers.Login)
	auth.Get("/me", controllers.Me)
	auth.Post("/rotate-stream-key", controllers.RotateStreamKey)

	// Channels Routes
	channels := api.Group("/channels")
	channels.Get("/", controllers.GetChannels)
	channels.Get("/primary", controllers.GetPrimaryChannel)
	channels.Get("/by-username/:username", controllers.GetChannelByUsername)
	channels.Get("/:id/following", controllers.GetFollowing)
	channels.Post("/:id/follow", controllers.FollowChannel)
	channels.Post("/:id/unfollow", controllers.UnfollowChannel)
	channels.Get("/:id/followers_list", controllers.GetFollowersList)
	channels.Get("/me", controllers.GetMyChannel)
	channels.Get("/:name", controllers.GetChannel)
	channels.Patch("/me", controllers.UpdateChannel)

	// Streams / Playback
	streams := api.Group("/streams")
	streams.Get("/playback/:id", controllers.GetPlaybackURL)

	// Restream
	restream := api.Group("/restream")
	restream.Get("/status", controllers.RestreamStatus)
	restream.Post("/start", controllers.RestreamStart)
	restream.Post("/stop", controllers.RestreamStop)
        
	// Proxy (Native IPTV)
	proxy := api.Group("/proxy")
	proxy.Get("/m3u8/:id", controllers.ProxyIPTV)
	proxy.Get("/ts", controllers.ProxyTS)

	// Webhooks (SRS)
	webhooks := streams.Group("/webhook")
	webhooks.Post("/on_publish", controllers.OnPublish)
	webhooks.Post("/on_unpublish", controllers.OnUnpublish)

	// Admin
	admin := api.Group("/admin")
	admin.Get("/users", controllers.ListUsers)
	admin.Post("/users/:id/ban", controllers.BanUser)

	// WebSockets Middleware (Asegura que es un upgrade de WS real)
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSockets Route
	app.Get("/ws/chat", websocket.New(controllers.WebsocketHandler))

	api.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	// Start server on port 8000
	log.Println("Starting Server on port 8000")
	log.Fatal(app.Listen(":8000"))
}
