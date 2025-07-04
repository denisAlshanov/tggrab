package telegram

import (
	"context"
	"io"
)

// TelegramClient defines the interface for Telegram clients
type TelegramClient interface {
	Connect(ctx context.Context) error
	ParseTelegramLink(link string) (channelName string, messageID int64, err error)
	GetMediaFromPost(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error)
	DownloadMedia(ctx context.Context, channelName string, messageID int64, mediaInfo MediaInfo) (io.ReadCloser, error)
	Close() error
}
