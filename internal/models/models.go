package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID       string             `bson:"post_id" json:"post_id"`
	TelegramLink string             `bson:"telegram_link" json:"telegram_link"`
	ChannelName  string             `bson:"channel_name" json:"channel_name"`
	MessageID    int64              `bson:"message_id" json:"message_id"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	Status       PostStatus         `bson:"status" json:"status"`
	MediaCount   int                `bson:"media_count" json:"media_count"`
	TotalSize    int64              `bson:"total_size" json:"total_size"`
	ErrorMessage string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
}

type PostStatus string

const (
	PostStatusPending    PostStatus = "pending"
	PostStatusProcessing PostStatus = "processing"
	PostStatusCompleted  PostStatus = "completed"
	PostStatusFailed     PostStatus = "failed"
)

type Media struct {
	ID             primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	MediaID        string                 `bson:"media_id" json:"media_id"`
	PostID         string                 `bson:"post_id" json:"post_id"`
	TelegramFileID string                 `bson:"telegram_file_id" json:"telegram_file_id"`
	FileName       string                 `bson:"file_name" json:"file_name"`
	FileType       string                 `bson:"file_type" json:"file_type"`
	FileSize       int64                  `bson:"file_size" json:"file_size"`
	S3Bucket       string                 `bson:"s3_bucket" json:"s3_bucket"`
	S3Key          string                 `bson:"s3_key" json:"s3_key"`
	FileHash       string                 `bson:"file_hash" json:"file_hash"`
	DownloadedAt   time.Time              `bson:"downloaded_at" json:"downloaded_at"`
	Metadata       map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
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
