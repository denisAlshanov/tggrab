package downloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type Downloader struct {
	db        *database.PostgresDB
	storage   storage.StorageInterface
	telegram  telegram.TelegramClient
	config    *config.DownloadConfig
	semaphore chan struct{}
	mu        sync.Mutex
}

func NewDownloader(db *database.PostgresDB, storage storage.StorageInterface, telegram telegram.TelegramClient, cfg *config.DownloadConfig) *Downloader {
	return &Downloader{
		db:        db,
		storage:   storage,
		telegram:  telegram,
		config:    cfg,
		semaphore: make(chan struct{}, cfg.MaxConcurrentDownloads),
	}
}

func (d *Downloader) ProcessPost(ctx context.Context, link string) (*models.Post, error) {
	// Parse the Telegram link
	channelName, messageID, err := d.telegram.ParseTelegramLink(link)
	if err != nil {
		return nil, utils.NewInvalidLinkError(link)
	}

	postID := fmt.Sprintf("%s_%d", channelName, messageID)

	// Check if post already exists (deduplication)
	existingPost, err := d.getPostByID(ctx, postID)
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
		PostID:       postID,
		TelegramLink: link,
		ChannelName:  channelName,
		MessageID:    messageID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       models.PostStatusPending,
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
	go d.downloadPostMedia(context.Background(), post)

	return post, nil
}

func (d *Downloader) downloadPostMedia(ctx context.Context, post *models.Post) {
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

			if err := d.downloadAndStoreMedia(ctx, post, info); err != nil {
				errorChan <- err
				utils.LogError(ctx, "Failed to download media", err, utils.Fields{
					"media_id": info.FileID,
					"post_id":  post.PostID,
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

func (d *Downloader) downloadAndStoreMedia(ctx context.Context, post *models.Post, mediaInfo telegram.MediaInfo) error {
	mediaID := generateMediaID(post.PostID, mediaInfo.FileID)

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
	s3Key := fmt.Sprintf("%s/%s/%s", post.ChannelName, post.PostID, mediaInfo.FileName)

	// Upload to S3
	metadata := map[string]string{
		"post_id":   post.PostID,
		"media_id":  mediaID,
		"file_name": mediaInfo.FileName,
	}

	if err := d.storage.UploadWithMetadata(ctx, s3Key, teeReader, mediaInfo.MimeType, metadata); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Calculate final hash
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Save media metadata
	media := &models.Media{
		MediaID:        mediaID,
		PostID:         post.PostID,
		TelegramFileID: mediaInfo.FileID,
		FileName:       mediaInfo.FileName,
		FileType:       mediaInfo.MimeType,
		FileSize:       mediaInfo.FileSize,
		S3Bucket:       d.storage.BucketName(),
		S3Key:          s3Key,
		FileHash:       hash,
		DownloadedAt:   time.Now(),
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

func (d *Downloader) getPostByID(ctx context.Context, postID string) (*models.Post, error) {
	return d.db.GetPostByID(ctx, postID)
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

func generateMediaID(postID, fileID string) string {
	return fmt.Sprintf("%s_%s", postID, fileID)
}
