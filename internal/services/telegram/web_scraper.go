package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WebScraperClient uses Telegram's web preview to get media info
type WebScraperClient struct {
	httpClient *http.Client
}

type webPreviewResponse struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Video       string `json:"video"`
}

func NewWebScraperClient() *WebScraperClient {
	return &WebScraperClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *WebScraperClient) Connect(ctx context.Context) error {
	// No connection needed for web scraping
	return nil
}

func (c *WebScraperClient) GetMediaFromPost(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error) {
	// Try to get preview data from Telegram's web interface
	previewURL := fmt.Sprintf("https://t.me/%s/%d?embed=1", channelName, messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", previewURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch preview: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch preview: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse HTML to find media
	mediaInfos := c.parseMediaFromHTML(string(body), channelName, messageID)

	if len(mediaInfos) == 0 {
		// Try the widget API
		return c.getMediaFromWidget(ctx, channelName, messageID)
	}

	return mediaInfos, nil
}

func (c *WebScraperClient) getMediaFromWidget(ctx context.Context, channelName string, messageID int64) ([]MediaInfo, error) {
	widgetURL := fmt.Sprintf("https://t.me/%s/%d?embed=1&mode=tme", channelName, messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", widgetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch widget: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return c.parseMediaFromHTML(string(body), channelName, messageID), nil
}

func (c *WebScraperClient) parseMediaFromHTML(html string, channelName string, messageID int64) []MediaInfo {
	var mediaInfos []MediaInfo

	// Look for image URLs
	imgRegex := regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`)
	if matches := imgRegex.FindStringSubmatch(html); len(matches) > 1 {
		mediaInfos = append(mediaInfos, MediaInfo{
			FileID:   fmt.Sprintf("web_image_%s_%d", channelName, messageID),
			FileName: fmt.Sprintf("image_%d.jpg", messageID),
			FileSize: 0, // Unknown from preview
			MimeType: "image/jpeg",
			Type:     MediaTypePhoto,
			URL:      matches[1], // Store URL for downloading
		})
	}

	// Look for video URLs - multiple patterns
	videoPatterns := []string{
		`<meta property="og:video" content="([^"]+)"`,
		`<meta property="og:video:url" content="([^"]+)"`,
		`<meta name="twitter:player:stream" content="([^"]+)"`,
		`<video[^>]+src="([^"]+)"`,
		`data-src="([^"]*\.mp4[^"]*)"`,
		`src="([^"]*\.mp4[^"]*)"`,
		`href="([^"]*\.mp4[^"]*)"`,
	}

	for i, pattern := range videoPatterns {
		videoRegex := regexp.MustCompile(pattern)
		for _, matches := range videoRegex.FindAllStringSubmatch(html, -1) {
			if len(matches) > 1 && !c.isDuplicateURL(mediaInfos, matches[1]) {
				// Determine file extension and mime type from URL
				url := matches[1]
				fileName, mimeType := c.getVideoFileInfo(url, messageID, i)

				mediaInfos = append(mediaInfos, MediaInfo{
					FileID:   fmt.Sprintf("web_video_%s_%d_%d", channelName, messageID, i),
					FileName: fileName,
					FileSize: 0, // Unknown from preview
					MimeType: mimeType,
					Type:     MediaTypeVideo,
					URL:      url,
				})
			}
		}
	}

	// Look for background images in style attributes
	bgRegex := regexp.MustCompile(`background-image:url\('([^']+)'\)`)
	for _, match := range bgRegex.FindAllStringSubmatch(html, -1) {
		if len(match) > 1 && !c.isDuplicateURL(mediaInfos, match[1]) {
			mediaInfos = append(mediaInfos, MediaInfo{
				FileID:   fmt.Sprintf("web_bg_%s_%d_%d", channelName, messageID, len(mediaInfos)),
				FileName: fmt.Sprintf("image_%d_%d.jpg", messageID, len(mediaInfos)),
				FileSize: 0,
				MimeType: "image/jpeg",
				Type:     MediaTypePhoto,
				URL:      match[1],
			})
		}
	}

	return mediaInfos
}

func (c *WebScraperClient) isDuplicateURL(mediaInfos []MediaInfo, url string) bool {
	for _, info := range mediaInfos {
		if info.URL == url {
			return true
		}
	}
	return false
}

func (c *WebScraperClient) ParseTelegramLink(link string) (channelName string, messageID int64, err error) {
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

func (c *WebScraperClient) DownloadMedia(ctx context.Context, channelName string, messageID int64, mediaInfo MediaInfo) (io.ReadCloser, error) {
	if mediaInfo.URL == "" {
		return nil, fmt.Errorf("no URL provided for media download")
	}
	return c.downloadMediaFromURL(ctx, mediaInfo.URL)
}

func (c *WebScraperClient) downloadMediaFromURL(ctx context.Context, mediaURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://t.me/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download media: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *WebScraperClient) Close() error {
	return nil
}

// getVideoFileInfo determines the appropriate filename and mime type for a video URL
func (c *WebScraperClient) getVideoFileInfo(url string, messageID int64, index int) (string, string) {
	// Extract file extension from URL
	urlLower := strings.ToLower(url)

	var extension, mimeType string

	switch {
	case strings.Contains(urlLower, ".mp4"):
		extension = ".mp4"
		mimeType = "video/mp4"
	case strings.Contains(urlLower, ".webm"):
		extension = ".webm"
		mimeType = "video/webm"
	case strings.Contains(urlLower, ".mov"):
		extension = ".mov"
		mimeType = "video/quicktime"
	case strings.Contains(urlLower, ".avi"):
		extension = ".avi"
		mimeType = "video/x-msvideo"
	case strings.Contains(urlLower, ".mkv"):
		extension = ".mkv"
		mimeType = "video/x-matroska"
	case strings.Contains(urlLower, ".flv"):
		extension = ".flv"
		mimeType = "video/x-flv"
	case strings.Contains(urlLower, ".m4v"):
		extension = ".m4v"
		mimeType = "video/x-m4v"
	case strings.Contains(urlLower, ".3gp"):
		extension = ".3gp"
		mimeType = "video/3gpp"
	default:
		// Default to MP4 if we can't determine the type
		extension = ".mp4"
		mimeType = "video/mp4"
	}

	fileName := fmt.Sprintf("video_%d_%d%s", messageID, index, extension)
	return fileName, mimeType
}
