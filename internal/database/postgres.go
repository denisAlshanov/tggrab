package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/models"
)

type PostgresDB struct {
	pool *pgxpool.Pool
	db   *sql.DB
}

func NewPostgresDB(cfg *config.PostgresConfig) (*PostgresDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create sql.DB for compatibility
	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection config: %w", err)
	}

	db := stdlib.OpenDB(*connConfig)

	pgdb := &PostgresDB{
		pool: pool,
		db:   db,
	}

	// Create tables if they don't exist
	if err := pgdb.createTables(ctx); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return pgdb, nil
}

func (p *PostgresDB) createTables(ctx context.Context) error {
	// Create custom type for post status
	createTypeQuery := `
		DO $$ BEGIN
			CREATE TYPE post_status AS ENUM ('pending', 'processing', 'completed', 'failed');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;`

	if _, err := p.pool.Exec(ctx, createTypeQuery); err != nil {
		return fmt.Errorf("failed to create post_status type: %w", err)
	}

	// Create posts table
	createPostsTable := `
		CREATE TABLE IF NOT EXISTS posts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			post_id VARCHAR(255) UNIQUE NOT NULL,
			telegram_link VARCHAR(500) UNIQUE NOT NULL,
			channel_name VARCHAR(255) NOT NULL,
			message_id BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			status post_status DEFAULT 'pending',
			media_count INTEGER DEFAULT 0,
			total_size BIGINT DEFAULT 0,
			error_message TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_posts_post_id ON posts(post_id);
		CREATE INDEX IF NOT EXISTS idx_posts_channel_name ON posts(channel_name);
		CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
	`

	if _, err := p.pool.Exec(ctx, createPostsTable); err != nil {
		return fmt.Errorf("failed to create posts table: %w", err)
	}

	// Create media table
	createMediaTable := `
		CREATE TABLE IF NOT EXISTS media (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			media_id VARCHAR(255) UNIQUE NOT NULL,
			post_id VARCHAR(255) NOT NULL,
			telegram_file_id VARCHAR(500) NOT NULL,
			file_name VARCHAR(500) NOT NULL,
			file_type VARCHAR(100) NOT NULL,
			file_size BIGINT NOT NULL,
			s3_bucket VARCHAR(255) NOT NULL,
			s3_key VARCHAR(500) NOT NULL,
			file_hash VARCHAR(255) NOT NULL,
			downloaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			metadata JSONB,
			FOREIGN KEY (post_id) REFERENCES posts(post_id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_media_media_id ON media(media_id);
		CREATE INDEX IF NOT EXISTS idx_media_post_id ON media(post_id);
		CREATE INDEX IF NOT EXISTS idx_media_file_hash ON media(file_hash);
		CREATE INDEX IF NOT EXISTS idx_media_telegram_file_id ON media(telegram_file_id);
	`

	if _, err := p.pool.Exec(ctx, createMediaTable); err != nil {
		return fmt.Errorf("failed to create media table: %w", err)
	}

	// Create update trigger for updated_at
	createTrigger := `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';

		DROP TRIGGER IF EXISTS update_posts_updated_at ON posts;
		CREATE TRIGGER update_posts_updated_at BEFORE UPDATE ON posts
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	if _, err := p.pool.Exec(ctx, createTrigger); err != nil {
		return fmt.Errorf("failed to create update trigger: %w", err)
	}

	return nil
}

// Post operations
func (p *PostgresDB) CreatePost(ctx context.Context, post *models.Post) error {
	post.ID = uuid.New()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	query := `
		INSERT INTO posts (id, post_id, telegram_link, channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err := p.pool.QueryRow(ctx, query,
		post.ID, post.PostID, post.TelegramLink, post.ChannelName, post.MessageID,
		post.CreatedAt, post.UpdatedAt, post.Status, post.MediaCount, post.TotalSize,
		post.ErrorMessage,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)

	return err
}

func (p *PostgresDB) GetPostByID(ctx context.Context, postID string) (*models.Post, error) {
	post := &models.Post{}
	query := `
		SELECT id, post_id, telegram_link, channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message
		FROM posts WHERE post_id = $1`

	err := p.pool.QueryRow(ctx, query, postID).Scan(
		&post.ID, &post.PostID, &post.TelegramLink, &post.ChannelName, &post.MessageID,
		&post.CreatedAt, &post.UpdatedAt, &post.Status, &post.MediaCount, &post.TotalSize,
		&post.ErrorMessage,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return post, err
}

func (p *PostgresDB) GetPostByLink(ctx context.Context, link string) (*models.Post, error) {
	post := &models.Post{}
	query := `
		SELECT id, post_id, telegram_link, channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message
		FROM posts WHERE telegram_link = $1`

	err := p.pool.QueryRow(ctx, query, link).Scan(
		&post.ID, &post.PostID, &post.TelegramLink, &post.ChannelName, &post.MessageID,
		&post.CreatedAt, &post.UpdatedAt, &post.Status, &post.MediaCount, &post.TotalSize,
		&post.ErrorMessage,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return post, err
}

func (p *PostgresDB) UpdatePost(ctx context.Context, post *models.Post) error {
	query := `
		UPDATE posts SET 
			telegram_link = $2, channel_name = $3, message_id = $4,
			status = $5, media_count = $6, total_size = $7, error_message = $8
		WHERE post_id = $1`

	_, err := p.pool.Exec(ctx, query,
		post.PostID, post.TelegramLink, post.ChannelName, post.MessageID,
		post.Status, post.MediaCount, post.TotalSize, post.ErrorMessage,
	)
	return err
}

func (p *PostgresDB) ListPosts(ctx context.Context, opts models.PaginationOptions) ([]models.Post, int, error) {
	// Set defaults
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.Limit

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM posts`
	if err := p.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get posts
	query := `
		SELECT id, post_id, telegram_link, channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message
		FROM posts 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := p.pool.Query(ctx, query, opts.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(
			&post.ID, &post.PostID, &post.TelegramLink, &post.ChannelName, &post.MessageID,
			&post.CreatedAt, &post.UpdatedAt, &post.Status, &post.MediaCount, &post.TotalSize,
			&post.ErrorMessage,
		)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	return posts, total, nil
}

// Media operations
func (p *PostgresDB) CreateMedia(ctx context.Context, media *models.Media) error {
	media.ID = uuid.New()
	media.DownloadedAt = time.Now()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(media.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO media (id, media_id, post_id, telegram_file_id, file_name, 
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, downloaded_at`

	err = p.pool.QueryRow(ctx, query,
		media.ID, media.MediaID, media.PostID, media.TelegramFileID, media.FileName,
		media.FileType, media.FileSize, media.S3Bucket, media.S3Key, media.FileHash,
		media.DownloadedAt, metadataJSON,
	).Scan(&media.ID, &media.DownloadedAt)

	return err
}

func (p *PostgresDB) GetMediaByID(ctx context.Context, mediaID string) (*models.Media, error) {
	media := &models.Media{}
	var metadataJSON []byte

	query := `
		SELECT id, media_id, post_id, telegram_file_id, file_name, 
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE media_id = $1`

	err := p.pool.QueryRow(ctx, query, mediaID).Scan(
		&media.ID, &media.MediaID, &media.PostID, &media.TelegramFileID, &media.FileName,
		&media.FileType, &media.FileSize, &media.S3Bucket, &media.S3Key, &media.FileHash,
		&media.DownloadedAt, &metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &media.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return media, nil
}

func (p *PostgresDB) GetMediaByPostID(ctx context.Context, postID string) ([]models.Media, error) {
	query := `
		SELECT id, media_id, post_id, telegram_file_id, file_name, 
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE post_id = $1
		ORDER BY downloaded_at ASC`

	rows, err := p.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		var metadataJSON []byte

		err := rows.Scan(
			&media.ID, &media.MediaID, &media.PostID, &media.TelegramFileID, &media.FileName,
			&media.FileType, &media.FileSize, &media.S3Bucket, &media.S3Key, &media.FileHash,
			&media.DownloadedAt, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &media.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		mediaList = append(mediaList, media)
	}

	return mediaList, nil
}

func (p *PostgresDB) GetMediaByHash(ctx context.Context, hash string) (*models.Media, error) {
	media := &models.Media{}
	var metadataJSON []byte

	query := `
		SELECT id, media_id, post_id, telegram_file_id, file_name, 
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE file_hash = $1
		LIMIT 1`

	err := p.pool.QueryRow(ctx, query, hash).Scan(
		&media.ID, &media.MediaID, &media.PostID, &media.TelegramFileID, &media.FileName,
		&media.FileType, &media.FileSize, &media.S3Bucket, &media.S3Key, &media.FileHash,
		&media.DownloadedAt, &metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &media.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return media, nil
}

// Transaction support
func (p *PostgresDB) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Health check
func (p *PostgresDB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return p.pool.Ping(ctx)
}

// Close connection
func (p *PostgresDB) Close() {
	p.pool.Close()
	p.db.Close()
}