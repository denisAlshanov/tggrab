# St. Planer - YouTube Stream Planner

A Go-based microservice for planning and scheduling YouTube live streams. St. Planer provides a RESTful API for managing stream schedules, planning content, and organizing streaming workflows for content creators.

## Features

- **Stream Scheduling**: Plan and schedule YouTube live streams in advance
- **Content Planning**: Organize stream topics, segments, and timing
- **Calendar Integration**: View and manage streaming schedule
- **Template Management**: Create reusable stream templates
- **Notification System**: Get reminders for upcoming streams
- **Analytics Integration**: Track stream performance and viewer engagement
- **Swagger Documentation**: Auto-generated API documentation

## Architecture

- **Backend**: Go with Gin web framework
- **Database**: PostgreSQL for stream data and schedules
- **Storage**: AWS S3 for stream assets and thumbnails
- **YouTube Integration**: YouTube Data API v3
- **Documentation**: Swagger/OpenAPI 3.0

## Quick Start

### Prerequisites

- Go 1.19+
- PostgreSQL
- AWS S3 bucket
- YouTube API credentials

### Installation

1. Clone the repository:
```bash
git clone https://github.com/denisAlshanov/stPlaner
cd stPlaner
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run the service:
```bash
go run cmd/main.go
```

### Docker Setup

For local development with all dependencies:

```bash
docker-compose up -d
```

## API Endpoints

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/media/grab` | Add new Telegram post for processing |
| GET | `/api/v1/media/list` | Retrieve list of processed posts |
| POST | `/api/v1/media/links` | Get media files from specific post |
| POST | `/api/v1/media/get` | Download specific media file |
| POST | `/api/v1/media/getDirect` | Get S3 pre-signed URL |

### Stream Planning Features

The service supports comprehensive stream planning:

- **Time Zones**: Support for multiple time zones
- **Recurring Streams**: Set up weekly/monthly recurring streams
- **Content Segments**: Plan stream segments with timestamps
- **Collaboration**: Share stream plans with team members

### API Documentation

Interactive Swagger documentation is available at `/swagger/index.html` when running the service.

## Configuration

### Environment Variables

```bash
# Server Configuration
PORT=8080
HOST=localhost

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=stplaner
POSTGRES_PASSWORD=stplaner_password
POSTGRES_DATABASE=stplaner
POSTGRES_SSLMODE=disable

# AWS S3
AWS_REGION=us-east-1
AWS_S3_BUCKET=your-bucket-name
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# YouTube API
YOUTUBE_API_KEY=your-api-key
YOUTUBE_CLIENT_ID=your-client-id
YOUTUBE_CLIENT_SECRET=your-client-secret

# Authentication & Rate Limiting
API_KEY=your-api-key
RATE_LIMIT_RPM=100
```

### YouTube API Setup

1. Visit [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable YouTube Data API v3
4. Create credentials (OAuth 2.0 Client ID)
5. Add credentials to your environment configuration

## Development

### Build Commands

```bash
# Build the application
go build -o stPlaner cmd/main.go

# Run with race detector
go run -race cmd/main.go

# Generate Swagger docs
swag init -g cmd/main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run go vet
go vet ./...
```

## Project Structure

```
stPlaner/
├── cmd/
│   └── main.go              # Application entry point
├── internal/               # Private application code
│   ├── api/               # API handlers and middleware
│   │   ├── handlers/      # HTTP handlers
│   │   ├── middleware/    # Auth, logging, rate limiting
│   │   └── router/        # Route definitions
│   ├── config/            # Configuration management
│   ├── database/          # MongoDB operations
│   ├── models/            # Data models
│   ├── services/          # Business logic
│   │   ├── scheduler/     # Stream scheduling logic
│   │   ├── storage/       # S3 operations
│   │   └── youtube/       # YouTube API client
│   └── utils/             # Utility functions
├── docs/                   # Swagger documentation
├── scripts/                # Build and deployment scripts
├── .env.example           # Environment template
├── Dockerfile             # Docker configuration
└── docker-compose.yml     # Local development setup
```

## Database Schema

### streams Table
- Stores stream planning data
- Schedule information
- Content segments and timing

### templates Table
- Reusable stream templates
- Default settings and segments
- Category organization

### users Table
- User profiles and preferences
- Authentication data
- Time zone settings

## Deployment

### Docker

```bash
# Build image
docker build -t stPlaner .

# Run container
docker run -p 8080:8080 --env-file .env stPlaner
```

### Cloud Deployment

The service is designed to run on:
- AWS ECS/EKS for container orchestration
- Google Cloud Run for serverless deployment
- PostgreSQL (managed or self-hosted)
- S3 for asset storage

## Monitoring & Health Checks

- Health check endpoint: `GET /health`
- Structured logging with correlation IDs
- Prometheus metrics (configurable)
- Cloud monitoring integration

## Security Considerations

- API key authentication
- OAuth 2.0 for YouTube integration
- Rate limiting protection
- Input validation and sanitization
- Secure credential storage

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

[Add your license information here]

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation
- Review API documentation at `/swagger/index.html`