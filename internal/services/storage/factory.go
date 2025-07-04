package storage

import (
	"fmt"

	"github.com/tggrab/tggrab/internal/config"
)

// NewStorage creates S3 storage
func NewStorage(cfg *config.S3Config) (StorageInterface, error) {
	fmt.Printf("Creating S3 storage (endpoint: %s)\n", cfg.EndpointURL)
	storage, err := NewS3Storage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 storage: %w", err)
	}

	return storage, nil
}