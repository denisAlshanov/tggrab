// Package main provides the entry point for the Telegram Media Grabber service.
// @title Telegram Media Grabber API
// @version 1.0
// @description A Go-based microservice that automatically downloads and stores media content from Telegram posts.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API key authentication

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/denisAlshanov/stPlaner/docs" // Import for swagger docs
	"github.com/denisAlshanov/stPlaner/internal/api/handlers"
	"github.com/denisAlshanov/stPlaner/internal/api/router"
	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/services/downloader"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger := utils.GetLogger()
	logger.Info("Starting Telegram Media Grabber service")

	// Initialize database
	db, err := database.NewMongoDB(&cfg.MongoDB)
	if err != nil {
		logger.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Initialize S3 storage
	s3Storage, err := storage.NewStorage(&cfg.S3)
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize Telegram client
	telegramClient, err := telegram.NewClient(&cfg.Telegram)
	if err != nil {
		logger.Fatalf("Failed to initialize Telegram client: %v", err)
	}

	// Connect to Telegram synchronously to ensure authentication
	logger.Info("Connecting to Telegram...")
	if err := telegramClient.Connect(context.Background()); err != nil {
		logger.Errorf("Failed to connect to Telegram: %v", err)
		logger.Info("Telegram connection failed - service will run with limited functionality")
	} else {
		logger.Info("Successfully connected to Telegram")
	}

	// Initialize downloader service
	downloaderService := downloader.NewDownloader(db, s3Storage, telegramClient, &cfg.Download)

	// Initialize handlers
	postHandler := handlers.NewPostHandler(db, downloaderService)
	mediaHandler := handlers.NewMediaHandler(db, s3Storage, telegramClient)
	healthHandler := handlers.NewHealthHandler(db, s3Storage)

	// Initialize router
	r := router.NewRouter(cfg, postHandler, mediaHandler, healthHandler)

	// Start server
	go func() {
		logger.Infof("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := r.Start(); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close database connection
	if err := db.Close(ctx); err != nil {
		logger.Errorf("Failed to close database connection: %v", err)
	}

	// Close Telegram client
	if err := telegramClient.Close(); err != nil {
		logger.Errorf("Failed to close Telegram client: %v", err)
	}

	logger.Info("Server shutdown complete")
}
