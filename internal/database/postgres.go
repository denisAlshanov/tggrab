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
			content_id VARCHAR(255) UNIQUE NOT NULL,
			telegram_link VARCHAR(500) UNIQUE NOT NULL,
			channel_name VARCHAR(255) NOT NULL,
			original_channel_name VARCHAR(255) NOT NULL,
			message_id BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			status post_status DEFAULT 'pending',
			media_count INTEGER DEFAULT 0,
			total_size BIGINT DEFAULT 0,
			error_message TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_posts_content_id ON posts(content_id);
		CREATE INDEX IF NOT EXISTS idx_posts_channel_name ON posts(channel_name);
		CREATE INDEX IF NOT EXISTS idx_posts_original_channel_name ON posts(original_channel_name);
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
			content_id VARCHAR(255) NOT NULL,
			telegram_file_id VARCHAR(500) NOT NULL,
			file_name VARCHAR(500) NOT NULL,
			original_file_name VARCHAR(500) NOT NULL,
			file_type VARCHAR(100) NOT NULL,
			file_size BIGINT NOT NULL,
			s3_bucket VARCHAR(255) NOT NULL,
			s3_key VARCHAR(500) NOT NULL,
			file_hash VARCHAR(255) NOT NULL,
			downloaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			metadata JSONB,
			FOREIGN KEY (content_id) REFERENCES posts(content_id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_media_media_id ON media(media_id);
		CREATE INDEX IF NOT EXISTS idx_media_content_id ON media(content_id);
		CREATE INDEX IF NOT EXISTS idx_media_file_hash ON media(file_hash);
		CREATE INDEX IF NOT EXISTS idx_media_telegram_file_id ON media(telegram_file_id);
		CREATE INDEX IF NOT EXISTS idx_media_original_file_name ON media(original_file_name);
	`

	if _, err := p.pool.Exec(ctx, createMediaTable); err != nil {
		return fmt.Errorf("failed to create media table: %w", err)
	}

	// Run migrations for existing databases
	if err := p.runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
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

// runMigrations handles database schema migrations
func (p *PostgresDB) runMigrations(ctx context.Context) error {
	// Migration 1: Add original_channel_name column to posts table if it doesn't exist
	addOriginalChannelName := `
		DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						   WHERE table_name = 'posts' AND column_name = 'original_channel_name') THEN
				ALTER TABLE posts ADD COLUMN original_channel_name VARCHAR(255);
				-- Set original_channel_name to channel_name for existing records
				UPDATE posts SET original_channel_name = channel_name WHERE original_channel_name IS NULL;
				-- Make the column NOT NULL after updating existing records
				ALTER TABLE posts ALTER COLUMN original_channel_name SET NOT NULL;
				-- Create index
				CREATE INDEX IF NOT EXISTS idx_posts_original_channel_name ON posts(original_channel_name);
			END IF;
		END $$;
	`

	if _, err := p.pool.Exec(ctx, addOriginalChannelName); err != nil {
		return fmt.Errorf("failed to add original_channel_name column: %w", err)
	}

	// Migration 2: Add original_file_name column to media table if it doesn't exist
	addOriginalFileName := `
		DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						   WHERE table_name = 'media' AND column_name = 'original_file_name') THEN
				ALTER TABLE media ADD COLUMN original_file_name VARCHAR(500);
				-- Set original_file_name to file_name for existing records
				UPDATE media SET original_file_name = file_name WHERE original_file_name IS NULL;
				-- Make the column NOT NULL after updating existing records
				ALTER TABLE media ALTER COLUMN original_file_name SET NOT NULL;
				-- Create index
				CREATE INDEX IF NOT EXISTS idx_media_original_file_name ON media(original_file_name);
			END IF;
		END $$;
	`

	if _, err := p.pool.Exec(ctx, addOriginalFileName); err != nil {
		return fmt.Errorf("failed to add original_file_name column: %w", err)
	}

	// Migration 3: Rename post_id to content_id in posts table
	renamePostIDToContentID := `
		DO $$ 
		BEGIN
			-- Check if content_id column exists, if not rename post_id
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						   WHERE table_name = 'posts' AND column_name = 'content_id') THEN
				-- Rename the column
				ALTER TABLE posts RENAME COLUMN post_id TO content_id;
				-- Drop old index and create new one
				DROP INDEX IF EXISTS idx_posts_post_id;
				CREATE INDEX IF NOT EXISTS idx_posts_content_id ON posts(content_id);
			END IF;
		END $$;
	`
	
	if _, err := p.pool.Exec(ctx, renamePostIDToContentID); err != nil {
		return fmt.Errorf("failed to rename post_id to content_id: %w", err)
	}

	// Migration 4: Rename post_id to content_id in media table
	renameMediaPostIDToContentID := `
		DO $$ 
		BEGIN
			-- Check if content_id column exists in media table, if not rename post_id
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						   WHERE table_name = 'media' AND column_name = 'content_id') THEN
				-- Drop foreign key constraint first
				ALTER TABLE media DROP CONSTRAINT IF EXISTS media_post_id_fkey;
				-- Rename the column
				ALTER TABLE media RENAME COLUMN post_id TO content_id;
				-- Drop old index and create new one
				DROP INDEX IF EXISTS idx_media_post_id;
				CREATE INDEX IF NOT EXISTS idx_media_content_id ON media(content_id);
				-- Add new foreign key constraint
				ALTER TABLE media ADD CONSTRAINT media_content_id_fkey 
					FOREIGN KEY (content_id) REFERENCES posts(content_id) ON DELETE CASCADE;
			END IF;
		END $$;
	`
	
	if _, err := p.pool.Exec(ctx, renameMediaPostIDToContentID); err != nil {
		return fmt.Errorf("failed to rename post_id to content_id in media table: %w", err)
	}

	return nil
}

// Post operations
func (p *PostgresDB) CreatePost(ctx context.Context, post *models.Post) error {
	post.ID = uuid.New()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	query := `
		INSERT INTO posts (id, content_id, telegram_link, channel_name, original_channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err := p.pool.QueryRow(ctx, query,
		post.ID, post.ContentID, post.TelegramLink, post.ChannelName, post.OriginalChannelName, post.MessageID,
		post.CreatedAt, post.UpdatedAt, post.Status, post.MediaCount, post.TotalSize,
		post.ErrorMessage,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)

	return err
}

func (p *PostgresDB) GetPostByContentID(ctx context.Context, contentID string) (*models.Post, error) {
	post := &models.Post{}
	query := `
		SELECT id, content_id, telegram_link, channel_name, original_channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message
		FROM posts WHERE content_id = $1`

	err := p.pool.QueryRow(ctx, query, contentID).Scan(
		&post.ID, &post.ContentID, &post.TelegramLink, &post.ChannelName, &post.OriginalChannelName, &post.MessageID,
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
		SELECT id, content_id, telegram_link, channel_name, original_channel_name, message_id, 
			created_at, updated_at, status, media_count, total_size, error_message
		FROM posts WHERE telegram_link = $1`

	err := p.pool.QueryRow(ctx, query, link).Scan(
		&post.ID, &post.ContentID, &post.TelegramLink, &post.ChannelName, &post.OriginalChannelName, &post.MessageID,
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
			telegram_link = $2, channel_name = $3, original_channel_name = $4, message_id = $5,
			status = $6, media_count = $7, total_size = $8, error_message = $9
		WHERE content_id = $1`

	_, err := p.pool.Exec(ctx, query,
		post.ContentID, post.TelegramLink, post.ChannelName, post.OriginalChannelName, post.MessageID,
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
		SELECT id, content_id, telegram_link, channel_name, original_channel_name, message_id, 
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
			&post.ID, &post.ContentID, &post.TelegramLink, &post.ChannelName, &post.OriginalChannelName, &post.MessageID,
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
		INSERT INTO media (id, media_id, content_id, telegram_file_id, file_name, original_file_name,
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, downloaded_at`

	err = p.pool.QueryRow(ctx, query,
		media.ID, media.MediaID, media.ContentID, media.TelegramFileID, media.FileName, media.OriginalFileName,
		media.FileType, media.FileSize, media.S3Bucket, media.S3Key, media.FileHash,
		media.DownloadedAt, metadataJSON,
	).Scan(&media.ID, &media.DownloadedAt)

	return err
}

func (p *PostgresDB) GetMediaByID(ctx context.Context, mediaID string) (*models.Media, error) {
	media := &models.Media{}
	var metadataJSON []byte

	query := `
		SELECT id, media_id, content_id, telegram_file_id, file_name, original_file_name,
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE media_id = $1`

	err := p.pool.QueryRow(ctx, query, mediaID).Scan(
		&media.ID, &media.MediaID, &media.ContentID, &media.TelegramFileID, &media.FileName, &media.OriginalFileName,
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

func (p *PostgresDB) GetMediaByContentID(ctx context.Context, contentID string) ([]models.Media, error) {
	query := `
		SELECT id, media_id, content_id, telegram_file_id, file_name, original_file_name,
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE content_id = $1
		ORDER BY downloaded_at ASC`

	rows, err := p.pool.Query(ctx, query, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		var metadataJSON []byte

		err := rows.Scan(
			&media.ID, &media.MediaID, &media.ContentID, &media.TelegramFileID, &media.FileName, &media.OriginalFileName,
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
		SELECT id, media_id, content_id, telegram_file_id, file_name, original_file_name,
			file_type, file_size, s3_bucket, s3_key, file_hash, downloaded_at, metadata
		FROM media WHERE file_hash = $1
		LIMIT 1`

	err := p.pool.QueryRow(ctx, query, hash).Scan(
		&media.ID, &media.MediaID, &media.ContentID, &media.TelegramFileID, &media.FileName, &media.OriginalFileName,
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

func (p *PostgresDB) UpdateMedia(ctx context.Context, media *models.Media) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(media.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE media SET 
			file_name = $2, original_file_name = $3, metadata = $4
		WHERE media_id = $1`

	_, err = p.pool.Exec(ctx, query,
		media.MediaID, media.FileName, media.OriginalFileName, metadataJSON,
	)
	return err
}

func (p *PostgresDB) DeleteMedia(ctx context.Context, mediaID string) error {
	query := `DELETE FROM media WHERE media_id = $1`

	result, err := p.pool.Exec(ctx, query, mediaID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no media found with ID: %s", mediaID)
	}

	return nil
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
