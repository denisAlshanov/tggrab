package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	S3       S3Config
	Telegram TelegramConfig
	API      APIConfig
	Download DownloadConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Timeout  time.Duration
}

type S3Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	EndpointURL     string
}

type TelegramConfig struct {
	APIId       int
	APIHash     string
	SessionFile string
}

type APIConfig struct {
	APIKey            string
	JWTSecret         string
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

type DownloadConfig struct {
	MaxConcurrentDownloads int
	DownloadTimeout        time.Duration
	MaxFileSize            int64
}

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
	Profile          string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	cfg := &Config{}

	// Server configuration
	cfg.Server.Port = getEnv("SERVER_PORT", "8080")
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")

	// PostgreSQL configuration
	cfg.Postgres.Host = getEnv("POSTGRES_HOST", "localhost")
	cfg.Postgres.Port = getEnvInt("POSTGRES_PORT", 5432)
	cfg.Postgres.User = getEnvRequired("POSTGRES_USER")
	cfg.Postgres.Password = getEnvRequired("POSTGRES_PASSWORD")
	cfg.Postgres.Database = getEnv("POSTGRES_DATABASE", "stplaner")
	cfg.Postgres.SSLMode = getEnv("POSTGRES_SSLMODE", "disable")
	pgTimeout, err := time.ParseDuration(getEnv("POSTGRES_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_TIMEOUT: %w", err)
	}
	cfg.Postgres.Timeout = pgTimeout

	// S3 configuration
	cfg.S3.Region = getEnv("AWS_REGION", "us-east-1")
	cfg.S3.BucketName = getEnvRequired("S3_BUCKET_NAME")
	cfg.S3.EndpointURL = getEnv("AWS_ENDPOINT_URL", "") // Optional for LocalStack
	cfg.S3.AccessKeyID = getEnvRequired("AWS_ACCESS_KEY_ID")
	cfg.S3.SecretAccessKey = getEnvRequired("AWS_SECRET_ACCESS_KEY")

	// Telegram configuration
	apiId, err := strconv.Atoi(getEnvRequired("TELEGRAM_API_ID"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}
	cfg.Telegram.APIId = apiId
	cfg.Telegram.APIHash = getEnvRequired("TELEGRAM_API_HASH")
	cfg.Telegram.SessionFile = getEnv("TELEGRAM_SESSION_FILE", "session.db")

	// API configuration
	cfg.API.APIKey = getEnvRequired("API_KEY")
	cfg.API.JWTSecret = getEnv("JWT_SECRET", "dev-jwt-secret-change-in-production-must-be-at-least-32-chars")
	cfg.API.RateLimitRequests = getEnvInt("RATE_LIMIT_REQUESTS", 100)
	rateLimitWindow, err := time.ParseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_WINDOW: %w", err)
	}
	cfg.API.RateLimitWindow = rateLimitWindow

	// Download configuration
	cfg.Download.MaxConcurrentDownloads = getEnvInt("MAX_CONCURRENT_DOWNLOADS", 5)
	downloadTimeout, err := time.ParseDuration(getEnv("DOWNLOAD_TIMEOUT", "300s"))
	if err != nil {
		return nil, fmt.Errorf("invalid DOWNLOAD_TIMEOUT: %w", err)
	}
	cfg.Download.DownloadTimeout = downloadTimeout
	cfg.Download.MaxFileSize = getEnvInt64("MAX_FILE_SIZE", 2*1024*1024*1024) // 2GB default

	// CORS configuration
	cfg.CORS = loadCORSConfig()

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(strings.TrimSpace(value), ",")
	}
	return defaultValue
}

// loadCORSConfig loads CORS configuration based on profile or custom settings
func loadCORSConfig() CORSConfig {
	profile := getEnv("CORS_PROFILE", "custom")
	
	switch profile {
	case "development":
		return getDevelopmentCORSConfig()
	case "production":
		return getProductionCORSConfig()
	default:
		return getCustomCORSConfig()
	}
}

// getDevelopmentCORSConfig returns permissive CORS settings for development
func getDevelopmentCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled: getEnvBool("CORS_ENABLED", true),
		AllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3000",
			"http://localhost:3001", 
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:8080",
		}),
		AllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH",
		}),
		AllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-API-Key",
		}),
		ExposedHeaders: getEnvStringSlice("CORS_EXPOSED_HEADERS", []string{
			"X-Total-Count", "X-Page", "X-Per-Page",
		}),
		AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getEnvInt("CORS_MAX_AGE", 86400),
		Profile:          "development",
	}
}

// getProductionCORSConfig returns secure CORS settings for production
func getProductionCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled: getEnvBool("CORS_ENABLED", true),
		AllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{
			"https://app.stplaner.com",
		}),
		AllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		}),
		AllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{
			"Origin", "Content-Type", "Accept", "Authorization",
		}),
		ExposedHeaders: getEnvStringSlice("CORS_EXPOSED_HEADERS", []string{
			"X-Total-Count",
		}),
		AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getEnvInt("CORS_MAX_AGE", 3600),
		Profile:          "production",
	}
}

// getCustomCORSConfig returns CORS settings from individual environment variables
func getCustomCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled: getEnvBool("CORS_ENABLED", true),
		AllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3000",
		}),
		AllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		}),
		AllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{
			"Origin", "Content-Type", "Accept", "Authorization",
		}),
		ExposedHeaders: getEnvStringSlice("CORS_EXPOSED_HEADERS", []string{}),
		AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getEnvInt("CORS_MAX_AGE", 3600),
		Profile:          "custom",
	}
}
