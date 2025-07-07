// Package main provides the entry point for the St. Planer YouTube Stream Planning service.
// @title St. Planer - YouTube Stream Planner API
// @version 1.0
// @description A Go-based microservice for planning and scheduling YouTube live streams. Helps content creators organize their streaming schedule and manage content.
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

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token authentication. Format: "Bearer {token}"

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
	"github.com/denisAlshanov/stPlaner/internal/api/middleware"
	"github.com/denisAlshanov/stPlaner/internal/api/router"
	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/services/auth"
	"github.com/denisAlshanov/stPlaner/internal/services/downloader"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
	"github.com/denisAlshanov/stPlaner/internal/services/youtube"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger := utils.GetLogger()
	logger.Info("Starting St. Planer - YouTube Stream Planner service")

	// Validate and log CORS configuration
	middleware.ValidateCORSConfig(&cfg.CORS)

	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Postgres)
	if err != nil {
		logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
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

	// Initialize YouTube client
	youtubeClient := youtube.NewClient()
	logger.Info("YouTube client initialized")

	// Initialize downloader service
	downloaderService := downloader.NewDownloader(db, s3Storage, telegramClient, youtubeClient, &cfg.Download)

	// Initialize authentication services
	jwtConfig := auth.JWTConfig{
		SecretKey:            cfg.API.JWTSecret,
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour, // 7 days
		Issuer:               "stPlaner",
	}
	
	jwtService := auth.NewJWTService(jwtConfig)
	sessionService := auth.NewSessionService(db, jwtService)
	
	// Initialize Google OIDC service
	googleConfig := auth.GoogleOIDCConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),
		Scopes:       []string{"openid", "email", "profile"},
	}
	googleService := auth.NewGoogleOIDCService(googleConfig, db, sessionService)

	// Initialize handlers
	postHandler := handlers.NewPostHandler(db, downloaderService)
	mediaHandler := handlers.NewMediaHandler(db, s3Storage, telegramClient, youtubeClient)
	healthHandler := handlers.NewHealthHandler(db, s3Storage)
	showHandler := handlers.NewShowHandler(db)
	eventHandler := handlers.NewEventHandler(db)
	guestHandler := handlers.NewGuestHandler(db)
	blockHandler := handlers.NewBlockHandler(db)
	userHandler := handlers.NewUserHandler(db)
	roleHandler := handlers.NewRoleHandler(db)
	authHandler := handlers.NewAuthHandlers(db, jwtService, sessionService, googleService)

	// Initialize router
	r := router.NewRouter(cfg, postHandler, mediaHandler, healthHandler, showHandler, eventHandler, guestHandler, blockHandler, userHandler, roleHandler, authHandler, jwtService, sessionService)

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

	// Close database connection
	db.Close()

	// Close Telegram client
	if err := telegramClient.Close(); err != nil {
		logger.Errorf("Failed to close Telegram client: %v", err)
	}

	logger.Info("Server shutdown complete")
}
