package telegram

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"

	"github.com/tggrab/tggrab/internal/config"
)

// MTProtoClient implements Telegram client using MTProto protocol
type MTProtoClient struct {
	client      *telegram.Client
	api         *tg.Client
	cfg         *config.TelegramConfig
	downloader  *downloader.Downloader
	isConnected bool
}

func NewMTProtoClient(cfg *config.TelegramConfig) (*MTProtoClient, error) {
	// Create session storage
	sessionStorage := &session.FileStorage{
		Path: cfg.SessionFile,
	}

	// Create client
	client := telegram.NewClient(cfg.APIId, cfg.APIHash, telegram.Options{
		SessionStorage: sessionStorage,
		Logger:         nil, // You can add logger here
	})

	return &MTProtoClient{
		client:      client,
		cfg:         cfg,
		downloader:  downloader.NewDownloader(),
		isConnected: false,
	}, nil
}

func (c *MTProtoClient) Connect(ctx context.Context) error {
	if c.isConnected {
		return nil
	}

	// For service applications, we don't need user authentication
	// We can use the API credentials directly for public channel access
	errChan := make(chan error, 1)

	go func() {
		err := c.client.Run(ctx, func(ctx context.Context) error {
			c.api = c.client.API()
			c.isConnected = true
			fmt.Println("MTProto client connected successfully (service mode)")
			return nil
		})
		errChan <- err
	}()

	// Wait for connection with timeout
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *MTProtoClient) GetMediaFromPost(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error) {
	if !c.isConnected || c.api == nil {
		return []MediaInfo{}, fmt.Errorf("MTProto client not connected")
	}

	// For public channels, try to resolve without authentication
	// This works for public channels and bots
	resolved, err := c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: channelName,
	})
	if err != nil {
		// If resolution fails due to auth issues, suggest using web scraper
		return nil, fmt.Errorf("failed to resolve channel %s: %w (this may require user authentication for private channels - consider using web scraper for public content)", channelName, err)
	}

	if len(resolved.Chats) == 0 {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	// Get channel info
	var channelID int64
	var accessHash int64

	switch chat := resolved.Chats[0].(type) {
	case *tg.Channel:
		channelID = chat.ID
		accessHash = chat.AccessHash
	default:
		return nil, fmt.Errorf("resolved entity is not a channel")
	}

	// Get messages from channel
	messages, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		OffsetID: int(messageID) + 1,
		Limit:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Extract media from messages
	var mediaInfos []MediaInfo

	switch msgs := messages.(type) {
	case *tg.MessagesMessages:
		for _, msg := range msgs.Messages {
			mediaInfos = append(mediaInfos, c.extractMediaFromMessage(msg)...)
		}
	case *tg.MessagesMessagesSlice:
		for _, msg := range msgs.Messages {
			mediaInfos = append(mediaInfos, c.extractMediaFromMessage(msg)...)
		}
	case *tg.MessagesChannelMessages:
		for _, msg := range msgs.Messages {
			mediaInfos = append(mediaInfos, c.extractMediaFromMessage(msg)...)
		}
	}

	return mediaInfos, nil
}

func (c *MTProtoClient) extractMediaFromMessage(msg tg.MessageClass) []MediaInfo {
	var mediaInfos []MediaInfo

	message, ok := msg.(*tg.Message)
	if !ok || message.Media == nil {
		return mediaInfos
	}

	switch media := message.Media.(type) {
	case *tg.MessageMediaPhoto:
		if photo, ok := media.Photo.(*tg.Photo); ok {
			info := MediaInfo{
				FileID:       fmt.Sprintf("photo_%d", photo.ID),
				FileName:     fmt.Sprintf("photo_%d.jpg", photo.ID),
				FileSize:     c.getPhotoSize(photo),
				MimeType:     "image/jpeg",
				Type:         MediaTypePhoto,
				TelegramData: photo, // Store for downloading
			}
			mediaInfos = append(mediaInfos, info)
		}

	case *tg.MessageMediaDocument:
		if doc, ok := media.Document.(*tg.Document); ok {
			// Enhanced video detection and processing
			mediaType := c.getDocumentType(doc)
			fileName := c.getDocumentFileName(doc)
			mimeType := doc.MimeType

			// Special handling for video files
			if mediaType == MediaTypeVideo {
				// Ensure proper video file extension
				if !strings.HasSuffix(fileName, ".mp4") && !strings.HasSuffix(fileName, ".mov") &&
					!strings.HasSuffix(fileName, ".avi") && !strings.HasSuffix(fileName, ".mkv") {
					// If no video extension, add .mp4 as default
					if mimeType == "video/mp4" || mimeType == "" {
						fileName = fmt.Sprintf("video_%d.mp4", doc.ID)
						mimeType = "video/mp4"
					} else if strings.HasPrefix(mimeType, "video/") {
						// Extract extension from mime type
						ext := strings.TrimPrefix(mimeType, "video/")
						fileName = fmt.Sprintf("video_%d.%s", doc.ID, ext)
					}
				}
			}

			info := MediaInfo{
				FileID:       fmt.Sprintf("document_%d", doc.ID),
				FileName:     fileName,
				FileSize:     doc.Size,
				MimeType:     mimeType,
				Type:         mediaType,
				TelegramData: doc, // Store for downloading
			}
			mediaInfos = append(mediaInfos, info)
		}

	// Handle video notes (round videos)
	case *tg.MessageMediaContact:
		// This shouldn't contain media, but keeping for completeness

	case *tg.MessageMediaGame:
		// Games might contain media, but typically not downloadable

	case *tg.MessageMediaGeo:
		// Geographic data, not media files

	case *tg.MessageMediaVenue:
		// Venue data, not media files

	case *tg.MessageMediaWebPage:
		// Web pages might contain embedded media, but complex to extract

	case *tg.MessageMediaPoll:
		// Polls don't contain media files

	case *tg.MessageMediaDice:
		// Dice animations, not downloadable media

	default:
		// Log unknown media types for debugging
		fmt.Printf("Unknown media type in message: %T\n", media)
	}

	return mediaInfos
}

func (c *MTProtoClient) DownloadMedia(ctx context.Context, channelName string, messageID int64, mediaInfo MediaInfo) (io.ReadCloser, error) {
	if !c.isConnected || c.api == nil {
		return nil, fmt.Errorf("MTProto client not connected")
	}

	var location tg.InputFileLocationClass

	switch data := mediaInfo.TelegramData.(type) {
	case *tg.Photo:
		// Get the largest photo size
		if len(data.Sizes) > 0 {
			var largest *tg.PhotoSize
			for _, size := range data.Sizes {
				if photoSize, ok := size.(*tg.PhotoSize); ok {
					if largest == nil || photoSize.Size > largest.Size {
						largest = photoSize
					}
				}
			}
			if largest != nil {
				location = &tg.InputPhotoFileLocation{
					ID:            data.ID,
					AccessHash:    data.AccessHash,
					FileReference: data.FileReference,
					ThumbSize:     largest.Type,
				}
			}
		}

	case *tg.Document:
		location = &tg.InputDocumentFileLocation{
			ID:            data.ID,
			AccessHash:    data.AccessHash,
			FileReference: data.FileReference,
		}
	}

	if location == nil {
		return nil, fmt.Errorf("unable to create download location")
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Download in background
	go func() {
		defer pw.Close()

		_, err := c.downloader.Download(c.api, location).Stream(ctx, pw)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("download failed: %w", err))
		}
	}()

	return pr, nil
}

func (c *MTProtoClient) getPhotoSize(photo *tg.Photo) int64 {
	var maxSize int64
	for _, size := range photo.Sizes {
		switch s := size.(type) {
		case *tg.PhotoSize:
			if int64(s.Size) > maxSize {
				maxSize = int64(s.Size)
			}
		case *tg.PhotoSizeProgressive:
			if int64(s.Sizes[len(s.Sizes)-1]) > maxSize {
				maxSize = int64(s.Sizes[len(s.Sizes)-1])
			}
		}
	}
	return maxSize
}

func (c *MTProtoClient) getDocumentFileName(doc *tg.Document) string {
	for _, attr := range doc.Attributes {
		if filename, ok := attr.(*tg.DocumentAttributeFilename); ok {
			return filename.FileName
		}
	}

	// Generate filename based on mime type
	ext := ".bin"
	if doc.MimeType != "" {
		parts := strings.Split(doc.MimeType, "/")
		if len(parts) == 2 {
			ext = "." + parts[1]
		}
	}
	return fmt.Sprintf("document_%d%s", doc.ID, ext)
}

func (c *MTProtoClient) getDocumentType(doc *tg.Document) MediaType {
	// Check mime type first (most reliable)
	if strings.HasPrefix(doc.MimeType, "video/") {
		return MediaTypeVideo
	}
	if strings.HasPrefix(doc.MimeType, "image/") {
		return MediaTypePhoto
	}

	// Check attributes for video indicators
	for _, attr := range doc.Attributes {
		switch attr.(type) {
		case *tg.DocumentAttributeVideo:
			// This is definitely a video
			return MediaTypeVideo
		case *tg.DocumentAttributeAnimated:
			// Animated files (GIFs) - treat as video for streaming purposes
			return MediaTypeVideo
		case *tg.DocumentAttributeImageSize:
			// Has image dimensions but check if it's actually a video
			if c.isVideoByExtension(doc) {
				return MediaTypeVideo
			}
			return MediaTypePhoto
		case *tg.DocumentAttributeFilename:
			// Check file extension as fallback
			if c.isVideoByExtension(doc) {
				return MediaTypeVideo
			}
		}
	}

	// Final fallback: check file extension
	if c.isVideoByExtension(doc) {
		return MediaTypeVideo
	}

	return MediaTypeDocument
}

// isVideoByExtension checks if the document is a video based on file extension
func (c *MTProtoClient) isVideoByExtension(doc *tg.Document) bool {
	fileName := c.getDocumentFileName(doc)
	fileName = strings.ToLower(fileName)

	videoExtensions := []string{
		".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv",
		".webm", ".m4v", ".3gp", ".ogv", ".ts", ".mts",
		".gif", // Animated GIFs can be treated as videos
	}

	for _, ext := range videoExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}

	return false
}

func (c *MTProtoClient) Close() error {
	// Client will be closed when context is cancelled
	c.isConnected = false
	return nil
}

func (c *MTProtoClient) ParseTelegramLink(link string) (channelName string, messageID int64, err error) {
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
