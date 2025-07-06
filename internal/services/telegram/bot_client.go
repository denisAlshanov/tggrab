package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotClient uses Telegram Bot API (requires bot token)
type BotClient struct {
	bot   *tgbotapi.BotAPI
	token string
}

func NewBotClient(token string) (*BotClient, error) {
	if token == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &BotClient{
		bot:   bot,
		token: token,
	}, nil
}

func (c *BotClient) Connect(ctx context.Context) error {
	// Bot API doesn't require connection setup
	me, err := c.bot.GetMe()
	if err != nil {
		return fmt.Errorf("failed to connect to Telegram Bot API: %w", err)
	}
	fmt.Printf("Connected as bot: @%s\n", me.UserName)
	return nil
}

func (c *BotClient) GetMediaFromPost(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error) {
	// For bot API, the channel name needs @ prefix
	if !strings.HasPrefix(channelName, "@") {
		channelName = "@" + channelName
	}

	// Bot API doesn't have a direct method to get message by ID from channel
	// This is a limitation - bot needs to be in the channel and listen to updates

	// For now, return an error explaining the limitation
	return nil, fmt.Errorf("Bot API requires the bot to be added to the channel and cannot fetch historical messages. Use User API instead")
}

func (c *BotClient) ParseTelegramLink(link string) (channelName string, messageID int64, err error) {
	return ParseTelegramLink(link)
}

func (c *BotClient) DownloadMedia(ctx context.Context, channelName string, messageID int64, mediaInfo MediaInfo) (io.ReadCloser, error) {
	// Get file info
	fileConfig := tgbotapi.FileConfig{FileID: mediaInfo.FileID}
	file, err := c.bot.GetFile(fileConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Download file
	fileURL := file.Link(c.token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *BotClient) Close() error {
	// Bot API doesn't need explicit cleanup
	return nil
}
