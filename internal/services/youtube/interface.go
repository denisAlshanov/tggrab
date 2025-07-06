package youtube

import (
	"context"
	"io"
)

// YouTubeClient interface for YouTube operations
type YouTubeClient interface {
	// ParseYouTubeURL extracts video ID from YouTube URL
	ParseYouTubeURL(url string) (string, error)

	// GetVideoInfo retrieves video metadata
	GetVideoInfo(ctx context.Context, videoID string) (*VideoInfo, error)

	// DownloadVideo downloads video content as a reader
	DownloadVideo(ctx context.Context, videoID string, quality string) (io.ReadCloser, *VideoInfo, error)

	// IsYouTubeURL checks if the provided URL is a valid YouTube URL
	IsYouTubeURL(url string) bool
}

// VideoInfo contains YouTube video metadata
type VideoInfo struct {
	ID           string
	Title        string
	Description  string
	Duration     string
	Author       string
	ThumbnailURL string
	FileSize     int64
	Quality      string
	Format       string
	MimeType     string
}
