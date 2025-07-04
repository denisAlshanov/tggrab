package storage

import (
	"context"
	"io"
	"time"
)

// StorageInterface defines the common interface for storage backends
type StorageInterface interface {
	BucketName() string
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	UploadWithMetadata(ctx context.Context, key string, data io.Reader, contentType string, metadata map[string]string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	GetMetadata(ctx context.Context, key string) (map[string]string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// CompletedPart represents a completed multipart upload part
type CompletedPart struct {
	ETag       *string
	PartNumber *int32
}

// MultipartStorageInterface extends StorageInterface for multipart uploads
type MultipartStorageInterface interface {
	StorageInterface
	InitiateMultipartUpload(ctx context.Context, key string, contentType string) (string, error)
	UploadPart(ctx context.Context, key string, uploadID string, partNumber int32, data io.Reader) (*CompletedPart, error)
	CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []CompletedPart) error
	AbortMultipartUpload(ctx context.Context, key string, uploadID string) error
}