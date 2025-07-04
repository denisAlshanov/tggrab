package models

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	PostID       string     `json:"post_id" db:"post_id"`
	TelegramLink string     `json:"telegram_link" db:"telegram_link"`
	ChannelName  string     `json:"channel_name" db:"channel_name"`
	MessageID    int64      `json:"message_id" db:"message_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	Status       PostStatus `json:"status" db:"status"`
	MediaCount   int        `json:"media_count" db:"media_count"`
	TotalSize    int64      `json:"total_size" db:"total_size"`
	ErrorMessage *string    `json:"error_message,omitempty" db:"error_message"`
}

type PostStatus string

const (
	PostStatusPending    PostStatus = "pending"
	PostStatusProcessing PostStatus = "processing"
	PostStatusCompleted  PostStatus = "completed"
	PostStatusFailed     PostStatus = "failed"
)

type Media struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	MediaID        string                 `json:"media_id" db:"media_id"`
	PostID         string                 `json:"post_id" db:"post_id"`
	TelegramFileID string                 `json:"telegram_file_id" db:"telegram_file_id"`
	FileName       string                 `json:"file_name" db:"file_name"`
	FileType       string                 `json:"file_type" db:"file_type"`
	FileSize       int64                  `json:"file_size" db:"file_size"`
	S3Bucket       string                 `json:"s3_bucket" db:"s3_bucket"`
	S3Key          string                 `json:"s3_key" db:"s3_key"`
	FileHash       string                 `json:"file_hash" db:"file_hash"`
	DownloadedAt   time.Time              `json:"downloaded_at" db:"downloaded_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

type PaginationOptions struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`
}

type PostListResponse struct {
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Links []PostListItem `json:"links"`
}

type PostListItem struct {
	PostID     string     `json:"post_id"`
	Link       string     `json:"link"`
	AddedAt    time.Time  `json:"added_at"`
	MediaCount int        `json:"media_count"`
	Status     PostStatus `json:"status"`
}

type MediaListResponse struct {
	PostID     string          `json:"post_id"`
	Link       string          `json:"link"`
	MediaFiles []MediaListItem `json:"media_files"`
}

type MediaListItem struct {
	MediaID    string    `json:"media_id"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	UploadDate time.Time `json:"upload_date"`
}

type AddPostRequest struct {
	Link string `json:"link" binding:"required"`
}

type AddPostResponse struct {
	Status           string     `json:"status"`
	Message          string     `json:"message"`
	PostID           string     `json:"post_id"`
	MediaCount       int        `json:"media_count"`
	ProcessingStatus PostStatus `json:"processing_status"`
}

type GetLinkListRequest struct {
	Link string `json:"link" binding:"required"`
}

type GetLinkMediaRequest struct {
	Link    string `json:"link" binding:"required"`
	MediaID string `json:"media_id" binding:"required"`
}

type GetLinkMediaURIRequest struct {
	Link          string `json:"link" binding:"required"`
	MediaID       string `json:"media_id" binding:"required"`
	ExpiryMinutes int    `json:"expiry_minutes,omitempty"`
}

type GetLinkMediaURIResponse struct {
	MediaID   string    `json:"media_id"`
	S3URL     string    `json:"s3_url"`
	ExpiresAt time.Time `json:"expires_at"`
}