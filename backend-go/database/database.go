package database

import (
	"log"
	"os"

	"fuchibol-backend-go/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback for local development if not in Docker
		dsn = "host=localhost user=fuchibol_user password=fuchibol_password dbname=fuchibol_db port=5435 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("Connected to Database successfully")

	// AutoMigrate the models to ensure tables exist
	// Since we are migrating from Python, tables should already exist, but this ensures schema matches
	log.Println("Running AutoMigrate...")
	err = db.AutoMigrate(
		&models.User{},
		&models.Channel{},
		&models.Stream{},
		&models.Recording{},
		&models.ChatMessage{},
		&models.Follower{},
		&models.BlockedTerm{},
	)
	if err != nil {
		log.Println("AutoMigrate failed (ignoring for backward compatibility with Python DB): \n", err)
	}

	DB = db
}
