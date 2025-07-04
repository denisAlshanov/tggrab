# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

St. Planer is a YouTube Stream Planner - a Go-based microservice for planning, scheduling, and managing YouTube live streams. The service provides a RESTful API that helps content creators organize their streaming schedule, plan content segments, and manage their streaming workflow efficiently.

## Key Architecture Components

- **API Framework**: Gin (with Swagger/OpenAPI 3.0 documentation)
- **Database**: PostgreSQL for stream schedules and metadata
- **Object Storage**: AWS S3 for thumbnails and stream assets
- **YouTube Integration**: YouTube Data API v3 for stream management
- **Authentication**: JWT tokens with API key support

## Common Development Commands

### Project Initialization (if not already done)
```bash
go mod init github.com/denisAlshanov/stPlaner
```

### Dependencies to Install
```bash
# Web framework (choose one)
go get -u github.com/gin-gonic/gin  # OR
go get -u github.com/labstack/echo/v4

# PostgreSQL driver
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool

# AWS SDK
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3

# Swagger
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/gin-swagger  # if using Gin
go get -u github.com/swaggo/echo-swagger # if using Echo

# Environment variables
go get github.com/joho/godotenv

# YouTube API client
go get google.golang.org/api/youtube/v3
go get golang.org/x/oauth2/google

# JWT
go get github.com/golang-jwt/jwt/v5
```

### Build Commands
```bash
# Build the application
go build -o stPlaner cmd/main.go

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
stPlaner/
├── cmd/
│   └── main.go              # Application entry point
├── internal/               # Private application code
│   ├── api/               # API handlers and middleware
│   │   ├── handlers/      # HTTP handlers for each endpoint
│   │   ├── middleware/    # Auth, logging, rate limiting
│   │   └── router/        # Route definitions
│   ├── config/            # Configuration management
│   ├── database/          # MongoDB connection and operations
│   ├── models/            # Data models (Stream, Template, User)
│   ├── services/          # Business logic
│   │   ├── scheduler/     # Stream scheduling logic
│   │   ├── storage/       # S3 operations
│   │   └── youtube/       # YouTube API integration
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

The service implements these main endpoint groups:

### Stream Management
1. `POST /streams/create` - Create new stream plan
2. `GET /streams/list` - List all planned streams
3. `GET /streams/{id}` - Get specific stream details
4. `PUT /streams/{id}` - Update stream plan
5. `DELETE /streams/{id}` - Delete stream plan

### Template Management
1. `POST /templates/create` - Create reusable stream template
2. `GET /templates/list` - List available templates
3. `GET /templates/{id}` - Get template details
4. `PUT /templates/{id}` - Update template
5. `DELETE /templates/{id}` - Delete template

### Calendar & Scheduling
1. `GET /calendar/week` - Get weekly calendar view
2. `GET /calendar/month` - Get monthly calendar view
3. `POST /calendar/schedule` - Schedule recurring streams

### Analytics & Reports
1. `GET /analytics/stream/{id}` - Get stream performance data
2. `GET /reports/monthly` - Generate monthly streaming report

## Database Collections

### streams Table
- Stores stream planning data
- Schedule information (date, time, duration)
- Content segments with timestamps
- Status tracking (planned, live, completed)

### templates Table
- Reusable stream templates
- Default segments and timing
- Category and tag organization

### users Table
- User profiles and preferences
- Authentication credentials
- Time zone settings
- Notification preferences

## Environment Configuration

Key environment variables needed:
- Server configuration (PORT, HOST)
- PostgreSQL connection (POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE)
- AWS S3 credentials and bucket
- YouTube API credentials (YOUTUBE_API_KEY, YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET)
- JWT settings (JWT_SECRET, JWT_EXPIRY)
- Rate limiting parameters

### YouTube Integration
The service uses YouTube Data API v3 for:
1. Creating and scheduling live broadcasts
2. Updating stream metadata
3. Retrieving analytics data
4. Managing stream thumbnails

To enable:
1. Create a project in Google Cloud Console
2. Enable YouTube Data API v3
3. Create OAuth 2.0 credentials
4. Set YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET in environment

## Development Workflow

1. Use structured logging with correlation IDs
2. Implement proper error handling with custom error codes
3. Follow RESTful API design principles
4. Use PostgreSQL transactions for data consistency
5. Implement caching for frequently accessed data
6. Generate Swagger docs after API changes
7. Write unit tests for all service methods

## Testing Guidelines

- Test all API endpoints with different scenarios
- Include integration tests for YouTube API  
- Test scheduling logic with various time zones
- Performance test with multiple concurrent streams
- Test template application and customization
- Use mock YouTube API for unit tests

## Important Considerations

1. **Rate Limiting**: YouTube API has quotas - implement proper quota management
2. **Time Zones**: Handle time zone conversions correctly for global users
3. **Notifications**: Implement reliable notification system for stream reminders
4. **Scalability**: Design for multiple users with concurrent streams
5. **Monitoring**: Track API usage and stream success rates
6. **Data Privacy**: Secure user credentials and streaming data