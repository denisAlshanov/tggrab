package youtube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

type Client struct {
	client     *youtube.Client
	httpClient *http.Client
}

// NewClient creates a new YouTube client
func NewClient() *Client {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	ytClient := &youtube.Client{
		HTTPClient: httpClient,
	}

	return &Client{
		client:     ytClient,
		httpClient: httpClient,
	}
}

// IsYouTubeURL checks if the provided URL is a valid YouTube URL
func (c *Client) IsYouTubeURL(url string) bool {
	patterns := []string{
		`^https?://(www\.)?youtube\.com/watch\?v=[\w-]+`,
		`^https?://(www\.)?youtube\.com/embed/[\w-]+`,
		`^https?://youtu\.be/[\w-]+`,
		`^https?://(www\.)?youtube\.com/v/[\w-]+`,
		`^https?://(m\.)?youtube\.com/watch\?v=[\w-]+`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}
	return false
}

// ParseYouTubeURL extracts video ID from YouTube URL
func (c *Client) ParseYouTubeURL(url string) (string, error) {
	patterns := []string{
		`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/|youtube\.com/v/)([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract video ID from YouTube URL: %s", url)
}

// GetVideoInfo retrieves video metadata
func (c *Client) GetVideoInfo(ctx context.Context, videoID string) (*VideoInfo, error) {
	video, err := c.client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// Find the best video format for metadata
	videoFormat := c.getBestVideoFormat(video.Formats, "best")
	if videoFormat == nil {
		return nil, fmt.Errorf("no suitable video format found")
	}

	// Find the best audio format for metadata
	audioFormat := c.getBestAudioFormat(video.Formats)
	if audioFormat == nil {
		return nil, fmt.Errorf("no suitable audio format found")
	}

	// Estimate combined file size (rough approximation)
	estimatedSize := videoFormat.ContentLength + audioFormat.ContentLength

	info := &VideoInfo{
		ID:          video.ID,
		Title:       video.Title,
		Description: video.Description,
		Duration:    video.Duration.String(),
		Author:      video.Author,
		FileSize:    estimatedSize, // This will be updated with actual size after merging
		Quality:     videoFormat.Quality,
		Format:      "video/mp4",
		MimeType:    "video/mp4",
	}

	// Get thumbnail URL
	if len(video.Thumbnails) > 0 {
		info.ThumbnailURL = video.Thumbnails[0].URL
	}

	return info, nil
}

// DownloadVideo downloads video and audio streams separately, merges them using FFmpeg, and returns the merged file
func (c *Client) DownloadVideo(ctx context.Context, videoID string, quality string) (io.ReadCloser, *VideoInfo, error) {
	video, err := c.client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get video: %w", err)
	}

	// Find the best video format (video only)
	videoFormat := c.getBestVideoFormat(video.Formats, quality)
	if videoFormat == nil {
		return nil, nil, fmt.Errorf("no suitable video format found")
	}

	// Find the best audio format
	audioFormat := c.getBestAudioFormat(video.Formats)
	if audioFormat == nil {
		return nil, nil, fmt.Errorf("no suitable audio format found")
	}

	// Create temporary directory for processing
	tempDir, err := os.MkdirTemp("", "youtube_download_*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Download video and audio streams
	videoPath := filepath.Join(tempDir, "video.mp4")
	audioPath := filepath.Join(tempDir, "audio.mp4")
	outputPath := filepath.Join(tempDir, "merged.mp4")

	// Download video stream
	if err := c.downloadStream(ctx, video, videoFormat, videoPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, nil, fmt.Errorf("failed to download video stream: %w", err)
	}

	// Download audio stream
	if err := c.downloadStream(ctx, video, audioFormat, audioPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, nil, fmt.Errorf("failed to download audio stream: %w", err)
	}

	// Merge video and audio using FFmpeg
	if err := c.mergeVideoAudio(ctx, videoPath, audioPath, outputPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, nil, fmt.Errorf("failed to merge video and audio: %w", err)
	}

	// Get file size of merged file
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, nil, fmt.Errorf("failed to get merged file info: %w", err)
	}

	// Open the merged file for reading
	mergedFile, err := os.Open(outputPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, nil, fmt.Errorf("failed to open merged file: %w", err)
	}

	// Create a wrapper that cleans up temp directory when closed
	wrapper := &tempFileWrapper{
		file:    mergedFile,
		tempDir: tempDir,
	}

	info := &VideoInfo{
		ID:          video.ID,
		Title:       video.Title,
		Description: video.Description,
		Duration:    video.Duration.String(),
		Author:      video.Author,
		FileSize:    fileInfo.Size(),
		Quality:     videoFormat.Quality,
		Format:      "video/mp4",
		MimeType:    "video/mp4",
	}

	// Get thumbnail URL
	if len(video.Thumbnails) > 0 {
		info.ThumbnailURL = video.Thumbnails[0].URL
	}

	return wrapper, info, nil
}

// getBestVideoFormat selects the best video-only format
func (c *Client) getBestVideoFormat(formats youtube.FormatList, preferredQuality string) *youtube.Format {
	var bestFormat *youtube.Format
	var bestQuality int
	targetQuality := c.parseQuality(preferredQuality)

	for _, format := range formats {
		// Only consider video formats (no audio)
		if format.MimeType == "" || !strings.Contains(format.MimeType, "video") {
			continue
		}

		// Skip formats with audio
		if format.AudioChannels > 0 {
			continue
		}

		// Prefer mp4 container
		if !strings.Contains(format.MimeType, "mp4") {
			continue
		}

		// Parse quality (height)
		quality := c.parseQuality(format.Quality)

		// If target quality specified, try to match it
		if targetQuality > 0 {
			if quality == targetQuality {
				return &format
			}
			// Find closest match
			if bestFormat == nil || abs(quality-targetQuality) < abs(bestQuality-targetQuality) {
				bestFormat = &format
				bestQuality = quality
			}
		} else {
			// No target quality, get highest quality
			if bestFormat == nil || quality > bestQuality {
				bestFormat = &format
				bestQuality = quality
			}
		}
	}

	// Fallback to any video format if no mp4 found
	if bestFormat == nil {
		for _, format := range formats {
			if format.MimeType != "" && strings.Contains(format.MimeType, "video") && format.AudioChannels == 0 {
				return &format
			}
		}
	}

	return bestFormat
}

// getBestAudioFormat selects the best audio-only format
func (c *Client) getBestAudioFormat(formats youtube.FormatList) *youtube.Format {
	var bestFormat *youtube.Format
	var bestBitrate int

	for _, format := range formats {
		// Only consider audio formats
		if format.MimeType == "" || !strings.Contains(format.MimeType, "audio") {
			continue
		}

		// Prefer mp4/m4a container
		if strings.Contains(format.MimeType, "mp4") || strings.Contains(format.MimeType, "m4a") {
			if bestFormat == nil || format.Bitrate > bestBitrate {
				bestFormat = &format
				bestBitrate = format.Bitrate
			}
		}
	}

	// Fallback to any audio format
	if bestFormat == nil {
		for _, format := range formats {
			if format.MimeType != "" && strings.Contains(format.MimeType, "audio") {
				if bestFormat == nil || format.Bitrate > bestBitrate {
					bestFormat = &format
					bestBitrate = format.Bitrate
				}
			}
		}
	}

	return bestFormat
}

// downloadStream downloads a stream to a file
func (c *Client) downloadStream(ctx context.Context, video *youtube.Video, format *youtube.Format, outputPath string) error {
	stream, _, err := c.client.GetStreamContext(ctx, video, format)
	if err != nil {
		return fmt.Errorf("failed to get stream: %w", err)
	}
	defer stream.Close()

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return fmt.Errorf("failed to write stream to file: %w", err)
	}

	return nil
}

// mergeVideoAudio merges video and audio files using FFmpeg
func (c *Client) mergeVideoAudio(ctx context.Context, videoPath, audioPath, outputPath string) error {
	// Check if FFmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// FFmpeg command to merge video and audio
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy", // Copy video stream without re-encoding
		"-c:a", "aac", // Encode audio to AAC
		"-strict", "experimental",
		"-y", // Overwrite output file
		outputPath,
	)

	// Run FFmpeg command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return nil
}

// parseQuality extracts numeric quality from quality string (e.g., "720p" -> 720)
func (c *Client) parseQuality(quality string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(quality)
	if len(matches) > 1 {
		if q, err := strconv.Atoi(matches[1]); err == nil {
			return q
		}
	}
	return 0
}

// tempFileWrapper wraps a file and cleans up temp directory when closed
type tempFileWrapper struct {
	file    *os.File
	tempDir string
}

func (w *tempFileWrapper) Read(p []byte) (n int, err error) {
	return w.file.Read(p)
}

func (w *tempFileWrapper) Close() error {
	err := w.file.Close()
	os.RemoveAll(w.tempDir) // Clean up temp directory
	return err
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
