# Telegram Media Downloader Service

A Go-based microservice that automatically downloads and stores media content from Telegram posts. The service provides a RESTful API for managing Telegram post links, downloading associated media files, and serving them to users on demand.

## Features

- **Automated Media Download**: Download media from Telegram posts using MTProto API
- **RESTful API**: Clean API endpoints for post management and media access
- **Video Streaming**: Support for HTTP range requests and progressive video playback
- **Cloud Storage**: AWS S3 integration for scalable media storage
- **Deduplication**: Intelligent content deduplication to save storage space
- **Rate Limiting**: Built-in protection against API abuse
- **Swagger Documentation**: Auto-generated API documentation

## Architecture

- **Backend**: Go with Gin web framework
- **Database**: MongoDB for metadata storage
- **Storage**: AWS S3 for media files
- **Telegram Integration**: MTProto User API via gotd/td library
- **Documentation**: Swagger/OpenAPI 3.0

## Quick Start

### Prerequisites

- Go 1.19+
- MongoDB
- AWS S3 bucket
- Telegram API credentials

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
| POST | `/add` | Add new Telegram post for processing |
| GET | `/getList` | Retrieve list of processed posts |
| POST | `/getLinkList` | Get media files from specific post |
| POST | `/getLinkMedia` | Download specific media file |
| POST | `/getLinkMediaURI` | Get S3 pre-signed URL |

### Video Streaming

The service supports advanced video streaming capabilities:

- **Range Requests**: HTTP range headers for partial content delivery
- **Progressive Loading**: Start playback before full download
- **Multiple Formats**: Support for .mp4, .avi, .mov, .mkv, .webm, .flv, .m4v, .3gp, .gif
- **Adaptive Streaming**: Optimized for video players and seeking

### API Documentation

Interactive Swagger documentation is available at `/swagger/index.html` when running the service.

## Configuration

### Environment Variables

```bash
# Server Configuration
PORT=8080
HOST=localhost

# Database
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=stPlaner

# AWS S3
AWS_REGION=us-east-1
AWS_S3_BUCKET=your-bucket-name
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# Telegram API
TELEGRAM_API_ID=your-api-id
TELEGRAM_API_HASH=your-api-hash

# Authentication & Rate Limiting
API_KEY=your-api-key
RATE_LIMIT_RPM=100
```

### Telegram API Setup

1. Visit https://my.telegram.org
2. Create a new application
3. Get your API ID and API Hash
4. Add them to your environment configuration

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
│   │   ├── downloader/    # Media download logic
│   │   ├── storage/       # S3 operations
│   │   └── telegram/      # Telegram client
│   └── utils/             # Utility functions
├── docs/                   # Swagger documentation
├── scripts/                # Build and deployment scripts
├── .env.example           # Environment template
├── Dockerfile             # Docker configuration
└── docker-compose.yml     # Local development setup
```

## Database Schema

### Posts Collection
- Stores Telegram post metadata
- Tracks processing status
- Contains deduplication information

### Media Collection
- Individual media file information
- Maps to S3 storage locations
- Contains file metadata (size, type, hash)

## Deployment

### Docker

```bash
# Build image
docker build -t stPlaner .

# Run container
docker run -p 8080:8080 --env-file .env stPlaner
```

### AWS Deployment

The service is designed to run on AWS with:
- ECS/EKS for container orchestration
- RDS or MongoDB Atlas for database
- S3 for media storage
- CloudFront for CDN (optional)

## Monitoring & Health Checks

- Health check endpoint: `GET /health`
- Structured logging with correlation IDs
- Prometheus metrics (configurable)
- AWS CloudWatch integration

## Security Considerations

- API key authentication
- Rate limiting protection
- S3 pre-signed URLs with expiration
- Input validation and sanitization
- No sensitive data in logs

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