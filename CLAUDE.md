# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Telegram Media Downloader Service - a Go-based microservice that automatically downloads and stores media content from Telegram posts. The service provides a RESTful API for managing Telegram post links, downloading associated media files, and serving them to users on demand.

## Key Architecture Components

- **API Framework**: Gin (with Swagger/OpenAPI 3.0 documentation)
- **Database**: MongoDB for metadata storage
- **Object Storage**: AWS S3 for media files
- **Telegram Integration**: MTProto User API using gotd/td library for real media downloads

## Common Development Commands

### Project Initialization (if not already done)
```bash
go mod init github.com/username/tggrab
```

### Dependencies to Install
```bash
# Web framework (choose one)
go get -u github.com/gin-gonic/gin  # OR
go get -u github.com/labstack/echo/v4

# MongoDB driver
go get go.mongodb.org/mongo-driver/mongo

# AWS SDK
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3

# Swagger
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/gin-swagger  # if using Gin
go get -u github.com/swaggo/echo-swagger # if using Echo

# Environment variables
go get github.com/joho/godotenv

# Telegram client (choose based on approach)
go get -u github.com/gotd/td  # MTProto client
```

### Build Commands
```bash
# Build the application
go build -o tggrab cmd/main.go

# Run with race detector during development
go run -race cmd/main.go

# Generate Swagger documentation
swag init -g cmd/main.go
```

### Testing Commands
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific test
go test -run TestFunctionName ./internal/...

# Run with verbose output
go test -v ./...
```

### Linting and Formatting
```bash
# Format code
go fmt ./...

# Run linter (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run

# Run go vet
go vet ./...
```

## Project Structure

The recommended Go project structure for this service:

```
tggrab/
├── cmd/
│   └── main.go              # Application entry point
├── internal/               # Private application code
│   ├── api/               # API handlers and middleware
│   │   ├── handlers/      # HTTP handlers for each endpoint
│   │   ├── middleware/    # Auth, logging, rate limiting
│   │   └── router/        # Route definitions
│   ├── config/            # Configuration management
│   ├── database/          # MongoDB connection and operations
│   ├── models/            # Data models (Post, Media)
│   ├── services/          # Business logic
│   │   ├── downloader/    # Telegram media download logic
│   │   ├── storage/       # S3 operations
│   │   └── telegram/      # Telegram client wrapper
│   └── utils/             # Utility functions
├── pkg/                    # Public packages (if any)
├── docs/                   # Swagger generated documentation
├── scripts/                # Build and deployment scripts
├── tests/                  # Integration tests
│   └── e2e/               # End-to-end tests
├── .env.example           # Example environment variables
├── Dockerfile             # Docker configuration
├── docker-compose.yml     # Local development setup
└── go.mod                 # Go module file
```

## API Endpoints Overview

The service implements 5 main endpoints:

1. `POST /add` - Add new Telegram post for processing
2. `GET /getList` - Retrieve list of processed posts
3. `POST /getLinkList` - Get media files from specific post
4. `POST /getLinkMedia` - Download specific media file (supports range requests for video streaming)
5. `POST /getLinkMediaURI` - Get S3 pre-signed URL

### Video Processing & Streaming Support

The service includes comprehensive video handling:

#### Video Detection:
- **MIME Type Recognition**: Detects `video/*` content types
- **File Extension Analysis**: Supports .mp4, .avi, .mov, .mkv, .webm, .flv, .m4v, .3gp, .gif
- **Telegram Attributes**: Uses `DocumentAttributeVideo` and `DocumentAttributeAnimated`
- **Multi-Pattern Matching**: Web scraper detects videos from multiple HTML patterns

#### Video Streaming:
- **Range Requests**: Supports HTTP range headers for partial content delivery
- **Adaptive Streaming**: Optimized for video players and seeking
- **Progressive Loading**: Start playback before full download
- **Cache Headers**: Proper caching for video content
- **Content-Length Accuracy**: Uses S3 metadata for precise file sizes

## Database Collections

### posts Collection
- Stores Telegram post metadata
- Tracks processing status
- Contains deduplication information

### media Collection
- Stores individual media file information
- Maps to S3 storage locations
- Contains file metadata (size, type, hash)

## Environment Configuration

Key environment variables needed:
- Server configuration (PORT, HOST)
- MongoDB connection (MONGO_URI, MONGO_DATABASE)
- AWS S3 credentials and bucket
- Telegram API credentials (TELEGRAM_API_ID, TELEGRAM_API_HASH)
- Authentication settings
- Rate limiting parameters

### Telegram Integration
The service uses Telegram's web scraper for real media downloads from public channels. To enable:
1. Get API credentials from https://my.telegram.org
2. Set TELEGRAM_API_ID and TELEGRAM_API_HASH in environment
3. No authentication required - works immediately
4. Downloads real media files from public Telegram channels

## Development Workflow

1. Check for existing media before downloading (deduplication)
2. Use structured logging with correlation IDs
3. Implement proper error handling with custom error codes
4. Follow S3 key structure: `channel_name/post_id/filename`
5. Use MongoDB transactions for consistency
6. Implement retry logic for Telegram API calls
7. Generate Swagger docs after API changes

## Testing Guidelines

- Test error scenarios thoroughly
- Include integration tests for API endpoints  
- Performance test with large media files
- Test rate limiting and authentication
- Use real S3 storage for testing

## Important Considerations

1. **Rate Limiting**: Telegram has strict rate limits - implement proper backoff strategies
2. **Storage Costs**: Monitor S3 usage and implement lifecycle policies
3. **Security**: Never expose S3 credentials, use pre-signed URLs with expiration
4. **Scalability**: Design for horizontal scaling with proper connection pooling
5. **Monitoring**: Implement health checks and comprehensive logging