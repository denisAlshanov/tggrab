package downloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
	"github.com/denisAlshanov/stPlaner/internal/services/youtube"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type Downloader struct {
	db        *database.PostgresDB
	storage   storage.StorageInterface
	telegram  telegram.TelegramClient
	youtube   youtube.YouTubeClient
	config    *config.DownloadConfig
	semaphore chan struct{}
	mu        sync.Mutex
}

func NewDownloader(db *database.PostgresDB, storage storage.StorageInterface, telegram telegram.TelegramClient, youtube youtube.YouTubeClient, cfg *config.DownloadConfig) *Downloader {
	return &Downloader{
		db:        db,
		storage:   storage,
		telegram:  telegram,
		youtube:   youtube,
		config:    cfg,
		semaphore: make(chan struct{}, cfg.MaxConcurrentDownloads),
	}
}

func (d *Downloader) ProcessPost(ctx context.Context, link string) (*models.Post, error) {
	// Auto-detect if this is a YouTube or Telegram URL
	if d.youtube.IsYouTubeURL(link) {
		return d.processYouTubePost(ctx, link)
	} else {
		return d.processTelegramPost(ctx, link)
	}
}

func (d *Downloader) processTelegramPost(ctx context.Context, link string) (*models.Post, error) {
	// Parse the Telegram link
	channelName, messageID, err := d.telegram.ParseTelegramLink(link)
	if err != nil {
		return nil, utils.NewInvalidLinkError(link)
	}

	contentID := fmt.Sprintf("%s_%d", channelName, messageID)

	// Check if post already exists (deduplication)
	existingPost, err := d.getPostByContentID(ctx, contentID)
	if err == nil && existingPost != nil {
		// Post already exists
		if existingPost.Status == models.PostStatusCompleted {
			return existingPost, nil
		}
		// If processing or failed, we might want to retry
		if existingPost.Status == models.PostStatusFailed {
			// Reset status to retry
			existingPost.Status = models.PostStatusPending
		}
	}

	// Create or update post
	post := &models.Post{
		ContentID:           contentID,
		TelegramLink:        link,
		ChannelName:         channelName,
		OriginalChannelName: channelName, // Store original channel name
		MessageID:           messageID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		Status:              models.PostStatusPending,
	}

	if existingPost != nil {
		post.ID = existingPost.ID
		post.CreatedAt = existingPost.CreatedAt
	}

	// Save post to database
	if err := d.savePost(ctx, post); err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	// Process asynchronously
	go d.downloadTelegramMedia(context.Background(), post)

	return post, nil
}

func (d *Downloader) processYouTubePost(ctx context.Context, link string) (*models.Post, error) {
	// Parse YouTube URL to get video ID
	videoID, err := d.youtube.ParseYouTubeURL(link)
	if err != nil {
		return nil, utils.NewInvalidLinkError(link)
	}

	contentID := fmt.Sprintf("youtube_%s", videoID)

	// Check if post already exists (deduplication)
	existingPost, err := d.getPostByContentID(ctx, contentID)
	if err == nil && existingPost != nil {
		// Post already exists
		if existingPost.Status == models.PostStatusCompleted {
			return existingPost, nil
		}
		// If processing or failed, we might want to retry
		if existingPost.Status == models.PostStatusFailed {
			// Reset status to retry
			existingPost.Status = models.PostStatusPending
		}
	}

	// Get video info for initial metadata
	videoInfo, err := d.youtube.GetVideoInfo(ctx, videoID)
	if err != nil {
		return nil, utils.NewInvalidLinkError(link)
	}

	// Create or update post
	post := &models.Post{
		ContentID:           contentID,
		TelegramLink:        link, // Store original YouTube link here
		ChannelName:         videoInfo.Author,
		OriginalChannelName: videoInfo.Author, // Store original YouTube channel name
		MessageID:           0,                // YouTube doesn't have message IDs
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		Status:              models.PostStatusPending,
	}

	if existingPost != nil {
		post.ID = existingPost.ID
		post.CreatedAt = existingPost.CreatedAt
	}

	// Save post to database
	if err := d.savePost(ctx, post); err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	// Process asynchronously
	go d.downloadYouTubeMedia(context.Background(), post, videoID)

	return post, nil
}

func (d *Downloader) downloadTelegramMedia(ctx context.Context, post *models.Post) {
	// Update status to processing
	post.Status = models.PostStatusProcessing
	if err := d.updatePostStatus(ctx, post); err != nil {
		utils.LogError(ctx, "Failed to update post status", err)
		return
	}

	// Get media from Telegram
	mediaInfos, err := d.telegram.GetMediaFromPost(ctx, post.ChannelName, post.MessageID)
	if err != nil {
		utils.LogError(ctx, "Failed to get media from Telegram", err)
		post.Status = models.PostStatusFailed
		errMsg := err.Error()
		post.ErrorMessage = &errMsg
		d.updatePostStatus(ctx, post)
		return
	}

	// Download each media file
	var wg sync.WaitGroup
	var totalSize int64
	var mediaCount int
	errorChan := make(chan error, len(mediaInfos))

	for _, mediaInfo := range mediaInfos {
		wg.Add(1)
		go func(info telegram.MediaInfo) {
			defer wg.Done()

			// Acquire semaphore
			d.semaphore <- struct{}{}
			defer func() { <-d.semaphore }()

			if err := d.downloadAndStoreTelegramMedia(ctx, post, info); err != nil {
				errorChan <- err
				utils.LogError(ctx, "Failed to download media", err, utils.Fields{
					"media_id":   info.FileID,
					"content_id": post.ContentID,
				})
			} else {
				d.mu.Lock()
				totalSize += info.FileSize
				mediaCount++
				d.mu.Unlock()
			}
		}(mediaInfo)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	var hasError bool
	for err := range errorChan {
		if err != nil {
			hasError = true
			break
		}
	}

	// Update post status
	if hasError {
		post.Status = models.PostStatusFailed
		errMsg := "Some media files failed to download"
		post.ErrorMessage = &errMsg
	} else {
		post.Status = models.PostStatusCompleted
		post.MediaCount = mediaCount
		post.TotalSize = totalSize
	}

	post.UpdatedAt = time.Now()
	if err := d.updatePostStatus(ctx, post); err != nil {
		utils.LogError(ctx, "Failed to update post status", err)
	}
}

func (d *Downloader) downloadAndStoreTelegramMedia(ctx context.Context, post *models.Post, mediaInfo telegram.MediaInfo) error {
	mediaID := generateMediaID(post.ContentID, mediaInfo.FileID)

	// Check if media already exists (deduplication by hash)
	existingMedia, err := d.getMediaByID(ctx, mediaID)
	if err == nil && existingMedia != nil {
		// Media already downloaded
		return nil
	}

	// Download media from Telegram
	reader, err := d.telegram.DownloadMedia(ctx, post.ChannelName, post.MessageID, mediaInfo)
	if err != nil {
		return fmt.Errorf("failed to download media: %w", err)
	}
	defer reader.Close()

	// Calculate hash while reading
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	// Generate S3 key
	s3Key := fmt.Sprintf("%s/%s/%s", post.ChannelName, post.ContentID, mediaInfo.FileName)

	// Upload to S3
	metadata := map[string]string{
		"content_id": post.ContentID,
		"media_id":   mediaID,
		"file_name":  mediaInfo.FileName,
	}

	if err := d.storage.UploadWithMetadata(ctx, s3Key, teeReader, mediaInfo.MimeType, metadata); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Calculate final hash
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Save media metadata
	media := &models.Media{
		MediaID:          mediaID,
		ContentID:        post.ContentID,
		TelegramFileID:   mediaInfo.FileID,
		FileName:         mediaInfo.FileName,
		OriginalFileName: mediaInfo.FileName, // Store original filename from Telegram
		FileType:         mediaInfo.MimeType,
		FileSize:         mediaInfo.FileSize,
		S3Bucket:         d.storage.BucketName(),
		S3Key:            s3Key,
		FileHash:         hash,
		DownloadedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"type": string(mediaInfo.Type),
		},
	}

	if err := d.saveMedia(ctx, media); err != nil {
		// Try to clean up S3
		d.storage.Delete(ctx, s3Key)
		return fmt.Errorf("failed to save media metadata: %w", err)
	}

	return nil
}

func (d *Downloader) getPostByContentID(ctx context.Context, contentID string) (*models.Post, error) {
	return d.db.GetPostByContentID(ctx, contentID)
}

func (d *Downloader) savePost(ctx context.Context, post *models.Post) error {
	if post.ID == uuid.Nil {
		return d.db.CreatePost(ctx, post)
	} else {
		return d.db.UpdatePost(ctx, post)
	}
}

func (d *Downloader) updatePostStatus(ctx context.Context, post *models.Post) error {
	post.UpdatedAt = time.Now()
	return d.db.UpdatePost(ctx, post)
}

func (d *Downloader) getMediaByID(ctx context.Context, mediaID string) (*models.Media, error) {
	return d.db.GetMediaByID(ctx, mediaID)
}

func (d *Downloader) saveMedia(ctx context.Context, media *models.Media) error {
	return d.db.CreateMedia(ctx, media)
}

func (d *Downloader) downloadYouTubeMedia(ctx context.Context, post *models.Post, videoID string) {
	// Update status to processing
	post.Status = models.PostStatusProcessing
	if err := d.updatePostStatus(ctx, post); err != nil {
		utils.LogError(ctx, "Failed to update post status", err)
		return
	}

	// Download YouTube video
	err := d.downloadAndStoreYouTubeMedia(ctx, post, videoID)
	if err != nil {
		utils.LogError(ctx, "Failed to download YouTube video", err)
		post.Status = models.PostStatusFailed
		errMsg := err.Error()
		post.ErrorMessage = &errMsg
		d.updatePostStatus(ctx, post)
		return
	}

	// Update post status to completed
	post.Status = models.PostStatusCompleted
	post.MediaCount = 1 // YouTube videos are single files
	post.UpdatedAt = time.Now()
	if err := d.updatePostStatus(ctx, post); err != nil {
		utils.LogError(ctx, "Failed to update post status", err)
	}
}

func (d *Downloader) downloadAndStoreYouTubeMedia(ctx context.Context, post *models.Post, videoID string) error {
	mediaID := generateMediaID(post.ContentID, videoID)

	// Check if media already exists (deduplication)
	existingMedia, err := d.getMediaByID(ctx, mediaID)
	if err == nil && existingMedia != nil {
		// Media already downloaded
		return nil
	}

	// Download video from YouTube
	reader, videoInfo, err := d.youtube.DownloadVideo(ctx, videoID, "best")
	if err != nil {
		return fmt.Errorf("failed to download YouTube video: %w", err)
	}
	defer reader.Close()

	// Calculate hash while reading
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	// Generate filename and S3 key
	originalFileName := fmt.Sprintf("%s.mp4", videoInfo.Title) // Store original title as filename
	fileName := d.sanitizeFileName(originalFileName)           // Sanitized filename for file system
	s3Key := fmt.Sprintf("youtube/%s/%s", post.ContentID, fileName)

	// Upload to S3
	metadata := map[string]string{
		"content_id": post.ContentID,
		"media_id":   mediaID,
		"video_id":   videoID,
		"file_name":  fileName,
		"title":      videoInfo.Title,
		"author":     videoInfo.Author,
		"duration":   videoInfo.Duration,
		"quality":    videoInfo.Quality,
		"platform":   "youtube",
	}

	if err := d.storage.UploadWithMetadata(ctx, s3Key, teeReader, videoInfo.MimeType, metadata); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Calculate final hash
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Save media metadata
	media := &models.Media{
		MediaID:          mediaID,
		ContentID:        post.ContentID,
		TelegramFileID:   videoID, // Store YouTube video ID here
		FileName:         fileName,
		OriginalFileName: originalFileName, // Store original YouTube video title as filename
		FileType:         videoInfo.MimeType,
		FileSize:         videoInfo.FileSize,
		S3Bucket:         d.storage.BucketName(),
		S3Key:            s3Key,
		FileHash:         hash,
		DownloadedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"platform":      "youtube",
			"video_id":      videoID,
			"title":         videoInfo.Title,
			"author":        videoInfo.Author,
			"duration":      videoInfo.Duration,
			"quality":       videoInfo.Quality,
			"description":   videoInfo.Description,
			"thumbnail_url": videoInfo.ThumbnailURL,
		},
	}

	if err := d.saveMedia(ctx, media); err != nil {
		// Try to clean up S3
		d.storage.Delete(ctx, s3Key)
		return fmt.Errorf("failed to save media metadata: %w", err)
	}

	// Update post with total size
	d.mu.Lock()
	post.TotalSize = videoInfo.FileSize
	d.mu.Unlock()

	return nil
}

func (d *Downloader) sanitizeFileName(filename string) string {
	// Remove or replace invalid characters for file names
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Ensure the filename has .mp4 extension
	if !strings.HasSuffix(strings.ToLower(sanitized), ".mp4") {
		// Remove existing extension if any
		sanitized = strings.TrimSuffix(sanitized, filepath.Ext(sanitized))
		sanitized += ".mp4"
	}

	// Limit filename length
	if len(sanitized) > 200 {
		ext := filepath.Ext(sanitized)
		base := sanitized[:200-len(ext)]
		sanitized = base + ext
	}

	return sanitized
}

func generateMediaID(contentID, fileID string) string {
	return fmt.Sprintf("%s_%s", contentID, fileID)
}
