package telegram

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/tggrab/tggrab/internal/config"
)

type Client struct {
	config *config.TelegramConfig
}

type MediaInfo struct {
	FileID       string
	FileName     string
	FileSize     int64
	MimeType     string
	Type         MediaType
	URL          string      // For web scraper
	TelegramData interface{} // For MTProto (stores *tg.Photo or *tg.Document)
}

type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
)

// NewClient creates a new Telegram client
func NewClient(cfg *config.TelegramConfig) (TelegramClient, error) {
	// If API credentials are provided, use web scraper for reliable public channel access
	if cfg.APIId > 0 && cfg.APIHash != "" {
		fmt.Println("Using web scraper for Telegram integration (can download real media from public channels)")
		return NewWebScraperClient(), nil
	}

	// Fallback to simplified client
	fmt.Println("Warning: API credentials not configured. Using simplified client.")
	return &Client{config: cfg}, nil
}

// For backward compatibility, keep the simplified client methods
func (c *Client) Connect(ctx context.Context) error {
	// For now, return nil to allow the application to start
	// In a production environment, you would implement proper Telegram authentication
	// This is a simplified version for demonstration purposes
	fmt.Println("Warning: Using simplified Telegram client. Switch to MTProto for real media downloads.")
	return nil
}

func (c *Client) ParseTelegramLink(link string) (channelName string, messageID int64, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Handle different Telegram URL formats
	// https://t.me/channel_name/123
	// https://telegram.me/channel_name/123
	if u.Host != "t.me" && u.Host != "telegram.me" {
		return "", 0, fmt.Errorf("not a valid Telegram link")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid Telegram link format")
	}

	channelName = parts[0]
	messageID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid message ID: %w", err)
	}

	return channelName, messageID, nil
}

// ParseTelegramLink is a global function that can be used by any client
func ParseTelegramLink(link string) (channelName string, messageID int64, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Handle different Telegram URL formats
	// https://t.me/channel_name/123
	// https://telegram.me/channel_name/123
	if u.Host != "t.me" && u.Host != "telegram.me" {
		return "", 0, fmt.Errorf("not a valid Telegram link")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid Telegram link format")
	}

	channelName = parts[0]
	messageID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid message ID: %w", err)
	}

	return channelName, messageID, nil
}

func (c *Client) GetMediaFromPost(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error) {
	// This is a simplified implementation for demonstration purposes
	// In a real implementation, you would:
	// 1. Connect to Telegram API
	// 2. Authenticate with the provided credentials
	// 3. Fetch the message from the channel
	// 4. Extract media information

	// Return empty result - this simplified client doesn't fetch real media
	return []MediaInfo{}, fmt.Errorf("simplified client cannot fetch media - configure Telegram API credentials to enable web scraper")
}

func (c *Client) DownloadMedia(ctx context.Context, channelName string, messageID int64, mediaInfo MediaInfo) (io.ReadCloser, error) {
	// This simplified client cannot download media
	return nil, fmt.Errorf("simplified client cannot download media - configure Telegram API credentials to enable web scraper")
}

func (c *Client) Close() error {
	// Clean up any resources if needed
	return nil
}

// Note: This is a simplified implementation for demonstration purposes.
// In a production environment, you would need to:
//
// 1. Implement proper Telegram authentication using the TD library
// 2. Handle session management and storage
// 3. Implement proper error handling and retry logic
// 4. Add support for different types of media (photos, videos, documents)
// 5. Handle rate limiting from Telegram's side
// 6. Implement proper message parsing and media extraction
//
// The current implementation serves as a foundation that allows the rest
// of the application to work while the Telegram integration is being developed.
