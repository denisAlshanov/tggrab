package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
	"github.com/denisAlshanov/stPlaner/internal/services/youtube"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type MediaHandler struct {
	db       *database.PostgresDB
	storage  storage.StorageInterface
	telegram telegram.TelegramClient
	youtube  youtube.YouTubeClient
}

func NewMediaHandler(db *database.PostgresDB, storage storage.StorageInterface, telegram telegram.TelegramClient, youtube youtube.YouTubeClient) *MediaHandler {
	return &MediaHandler{
		db:       db,
		storage:  storage,
		telegram: telegram,
		youtube:  youtube,
	}
}

// GetLinkList godoc
// @Summary Get media files from a specific post
// @Description Get list of all media files from a specific Telegram post or YouTube video
// @Tags media
// @Accept json
// @Produce json
// @Param request body models.GetLinkListRequest true "Content ID for post"
// @Success 200 {object} models.MediaListResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/media/links [post]
// @Security BearerAuth
func (h *MediaHandler) GetLinkList(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.GetLinkListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Find the post by content ID
	post, err := h.db.GetPostByContentID(ctx, req.ContentID)
	if err != nil {
		utils.LogError(ctx, "Failed to find post", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	if post == nil {
		h.errorResponse(c, utils.NewPostNotFoundError(req.ContentID))
		return
	}

	// Find media files for this post
	mediaFiles, err := h.db.GetMediaByContentID(ctx, req.ContentID)
	if err != nil {
		utils.LogError(ctx, "Failed to find media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}

	// Convert to response format
	mediaList := make([]models.MediaListItem, len(mediaFiles))
	for i, media := range mediaFiles {
		mediaList[i] = models.MediaListItem{
			MediaID:    media.MediaID,
			FileName:   media.FileName,
			FileType:   media.FileType,
			FileSize:   media.FileSize,
			UploadDate: media.DownloadedAt,
		}
	}

	response := models.MediaListResponse{
		ContentID:  post.ContentID,
		Link:       post.TelegramLink,
		MediaFiles: mediaList,
	}

	c.JSON(http.StatusOK, response)
}

// GetLinkMedia godoc
// @Summary Download specific media file
// @Description Download specific media file from a post as binary stream. Supports range requests for video files to enable streaming and seeking.
// @Tags media
// @Accept json
// @Produce application/octet-stream
// @Param request body models.GetLinkMediaRequest true "Media download request"
// @Param Range header string false "Range header for partial content (e.g., bytes=0-1023)"
// @Success 200 {file} binary "Full file download"
// @Success 206 {file} binary "Partial content (range request)"
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 416 {object} map[string]interface{} "Range Not Satisfiable"
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/media/get [post]
// @Security BearerAuth
func (h *MediaHandler) GetLinkMedia(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.GetLinkMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Find the media file
	media, err := h.db.GetMediaByID(ctx, req.MediaID)
	if err != nil {
		utils.LogError(ctx, "Failed to find media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	if media == nil {
		h.errorResponse(c, utils.NewMediaNotFoundError(req.MediaID))
		return
	}

	// Get file metadata for proper handling
	metadata, err := h.storage.GetMetadata(ctx, media.S3Key)
	if err != nil {
		utils.LogError(ctx, "Failed to get S3 metadata", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}

	contentLengthStr := metadata["ContentLength"]
	fileSize, err := strconv.ParseInt(contentLengthStr, 10, 64)
	if err != nil {
		utils.LogError(ctx, "Invalid content length in metadata", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}

	// Check if this is a video file that might need range support
	isVideo := strings.HasPrefix(media.FileType, "video/")

	// Handle range requests for video files
	if isVideo {
		h.handleVideoStream(c, ctx, media, fileSize, req.MediaID)
	} else {
		h.handleFileStream(c, ctx, media, fileSize, req.MediaID)
	}
}

// handleVideoStream handles video files with range request support
func (h *MediaHandler) handleVideoStream(c *gin.Context, ctx context.Context, media *models.Media, fileSize int64, mediaID string) {
	// Parse Range header for video streaming
	rangeHeader := c.GetHeader("Range")

	if rangeHeader != "" {
		// Handle range request (e.g., "bytes=0-1023")
		h.handleRangeRequest(c, ctx, media, fileSize, rangeHeader, mediaID)
	} else {
		// Full file request
		h.streamFullVideo(c, ctx, media, fileSize, mediaID)
	}
}

// handleFileStream handles regular files (images, documents)
func (h *MediaHandler) handleFileStream(c *gin.Context, ctx context.Context, media *models.Media, fileSize int64, mediaID string) {
	// Set response headers
	c.Header("Content-Type", media.FileType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", media.FileName))
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
	c.Header("Accept-Ranges", "bytes")

	// Download from S3
	reader, err := h.storage.Download(ctx, media.S3Key)
	if err != nil {
		utils.LogError(ctx, "Failed to download from S3", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}
	defer reader.Close()

	// Stream the file
	written, err := io.Copy(c.Writer, reader)
	if err != nil {
		utils.LogError(ctx, "Failed to stream file", err, utils.Fields{
			"bytes_written": written,
			"media_id":      mediaID,
		})
		return
	}

	utils.LogInfo(ctx, "Successfully streamed file", utils.Fields{
		"media_id":      mediaID,
		"bytes_written": written,
		"file_name":     media.FileName,
	})
}

// streamFullVideo streams the entire video file
func (h *MediaHandler) streamFullVideo(c *gin.Context, ctx context.Context, media *models.Media, fileSize int64, mediaID string) {
	// Set video-specific headers
	c.Header("Content-Type", media.FileType)
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", media.FileName))

	// Add cache headers for video
	c.Header("Cache-Control", "public, max-age=3600")

	// Download from S3
	reader, err := h.storage.Download(ctx, media.S3Key)
	if err != nil {
		utils.LogError(ctx, "Failed to download video from S3", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}
	defer reader.Close()

	// Stream the video
	written, err := io.Copy(c.Writer, reader)
	if err != nil {
		utils.LogError(ctx, "Failed to stream video", err, utils.Fields{
			"bytes_written": written,
			"media_id":      mediaID,
		})
		return
	}

	utils.LogInfo(ctx, "Successfully streamed video", utils.Fields{
		"media_id":      mediaID,
		"bytes_written": written,
		"file_name":     media.FileName,
	})
}

// handleRangeRequest handles HTTP range requests for video streaming
func (h *MediaHandler) handleRangeRequest(c *gin.Context, ctx context.Context, media *models.Media, fileSize int64, rangeHeader string, mediaID string) {
	// Parse range header (e.g., "bytes=0-1023" or "bytes=1024-")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")

	if len(parts) != 2 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	var start, end int64
	var err error

	// Parse start
	if parts[0] != "" {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	// Parse end
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end >= fileSize {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	} else {
		// If end is not specified, serve to the end of file
		end = fileSize - 1
	}

	// Validate range
	if start > end || start >= fileSize {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1

	// Set partial content headers
	c.Header("Content-Type", media.FileType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", media.FileName))
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusPartialContent)

	// For now, we'll stream the full file and skip to the range
	// In a production system, you'd want to implement range requests in S3
	reader, err := h.storage.Download(ctx, media.S3Key)
	if err != nil {
		utils.LogError(ctx, "Failed to download video for range request", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}
	defer reader.Close()

	// Skip to start position
	if start > 0 {
		_, err = io.CopyN(io.Discard, reader, start)
		if err != nil {
			utils.LogError(ctx, "Failed to seek to range start", err)
			return
		}
	}

	// Stream the requested range
	written, err := io.CopyN(c.Writer, reader, contentLength)
	if err != nil && err != io.EOF {
		utils.LogError(ctx, "Failed to stream video range", err, utils.Fields{
			"bytes_written": written,
			"media_id":      mediaID,
			"range_start":   start,
			"range_end":     end,
		})
		return
	}

	utils.LogInfo(ctx, "Successfully streamed video range", utils.Fields{
		"media_id":      mediaID,
		"bytes_written": written,
		"range_start":   start,
		"range_end":     end,
		"file_name":     media.FileName,
	})
}

// GetLinkMediaURI godoc
// @Summary Get S3 pre-signed URL for media
// @Description Get direct S3 link for specific media with configurable expiration
// @Tags media
// @Accept json
// @Produce json
// @Param request body models.GetLinkMediaURIRequest true "Media URI request"
// @Success 200 {object} models.GetLinkMediaURIResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/media/getDirect [post]
// @Security BearerAuth
func (h *MediaHandler) GetLinkMediaURI(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.GetLinkMediaURIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Default expiry time
	expiryMinutes := req.ExpiryMinutes
	if expiryMinutes <= 0 {
		expiryMinutes = 60 // 1 hour default
	}

	// Find the media file
	media, err := h.db.GetMediaByID(ctx, req.MediaID)
	if err != nil {
		utils.LogError(ctx, "Failed to find media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	if media == nil {
		h.errorResponse(c, utils.NewMediaNotFoundError(req.MediaID))
		return
	}

	// Generate pre-signed URL
	expiry := time.Duration(expiryMinutes) * time.Minute
	presignedURL, err := h.storage.GeneratePresignedURL(ctx, media.S3Key, expiry)
	if err != nil {
		utils.LogError(ctx, "Failed to generate presigned URL", err)
		h.errorResponse(c, utils.NewS3Error(err))
		return
	}

	response := models.GetLinkMediaURIResponse{
		MediaID:   media.MediaID,
		S3URL:     presignedURL,
		ExpiresAt: time.Now().Add(expiry),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateLinkMedia godoc
// @Summary Update media metadata
// @Description Update media file metadata including filename and custom metadata
// @Tags media
// @Accept json
// @Produce json
// @Param request body models.UpdateMediaRequest true "Media update request"
// @Success 200 {object} models.UpdateMediaResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/media/get [put]
// @Security BearerAuth
func (h *MediaHandler) UpdateLinkMedia(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.UpdateMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Find the media file
	media, err := h.db.GetMediaByID(ctx, req.MediaID)
	if err != nil {
		utils.LogError(ctx, "Failed to find media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	if media == nil {
		h.errorResponse(c, utils.NewMediaNotFoundError(req.MediaID))
		return
	}

	// Update fields if provided
	if req.FileName != nil {
		media.FileName = *req.FileName
	}
	if req.OriginalFileName != nil {
		media.OriginalFileName = *req.OriginalFileName
	}
	if req.Metadata != nil {
		// Merge metadata instead of replacing completely
		if media.Metadata == nil {
			media.Metadata = make(map[string]interface{})
		}
		for key, value := range req.Metadata {
			media.Metadata[key] = value
		}
	}

	// Update in database
	if err := h.db.UpdateMedia(ctx, media); err != nil {
		utils.LogError(ctx, "Failed to update media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}

	response := models.UpdateMediaResponse{
		Status:  "success",
		Message: "Media updated successfully",
		MediaID: media.MediaID,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteLinkMedia godoc
// @Summary Delete media file
// @Description Delete media file from database and S3 storage
// @Tags media
// @Accept json
// @Produce json
// @Param request body models.DeleteMediaRequest true "Media delete request"
// @Success 200 {object} models.DeleteMediaResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/media/get [delete]
// @Security BearerAuth
func (h *MediaHandler) DeleteLinkMedia(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.DeleteMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Find the media file first to get S3 info
	media, err := h.db.GetMediaByID(ctx, req.MediaID)
	if err != nil {
		utils.LogError(ctx, "Failed to find media", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	if media == nil {
		h.errorResponse(c, utils.NewMediaNotFoundError(req.MediaID))
		return
	}

	// Delete from S3 first
	if err := h.storage.Delete(ctx, media.S3Key); err != nil {
		utils.LogError(ctx, "Failed to delete from S3", err, utils.Fields{
			"s3_key":   media.S3Key,
			"media_id": req.MediaID,
		})
		// Continue with database deletion even if S3 deletion fails
		// Log the error but don't fail the entire operation
	}

	// Delete from database
	if err := h.db.DeleteMedia(ctx, req.MediaID); err != nil {
		utils.LogError(ctx, "Failed to delete media from database", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}

	response := models.DeleteMediaResponse{
		Status:  "success",
		Message: "Media deleted successfully",
		MediaID: req.MediaID,
	}

	c.JSON(http.StatusOK, response)
}

func (h *MediaHandler) errorResponse(c *gin.Context, err *utils.AppError) {
	c.JSON(err.StatusCode, gin.H{
		"error":      err,
		"request_id": c.GetString("request_id"),
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}
