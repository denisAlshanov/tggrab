package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/utils"
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

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// runMigrations handles database schema migrations with proper version tracking
func (p *PostgresDB) runMigrations(ctx context.Context) error {
	// First, create migrations table if it doesn't exist
	if err := p.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Define all migrations
	migrations := []Migration{
		{
			Version:     1,
			Description: "Add original_channel_name column to posts table",
			SQL: `
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
			`,
		},
		{
			Version:     2,
			Description: "Add original_file_name column to media table",
			SQL: `
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
			`,
		},
		{
			Version:     3,
			Description: "Rename post_id to content_id in posts table",
			SQL: `
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
			`,
		},
		{
			Version:     4,
			Description: "Rename post_id to content_id in media table",
			SQL: `
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
			`,
		},
		{
			Version:     5,
			Description: "Add shows table for YouTube show planning",
			SQL: `
				-- Create custom types for shows
				DO $$ BEGIN
					CREATE TYPE repeat_pattern AS ENUM ('none', 'daily', 'weekly', 'biweekly', 'monthly', 'custom');
				EXCEPTION
					WHEN duplicate_object THEN null;
				END $$;

				DO $$ BEGIN
					CREATE TYPE show_status AS ENUM ('active', 'paused', 'completed', 'cancelled');
				EXCEPTION
					WHEN duplicate_object THEN null;
				END $$;

				-- Create shows table
				CREATE TABLE IF NOT EXISTS shows (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					show_name VARCHAR(255) NOT NULL,
					youtube_key VARCHAR(500) NOT NULL,
					additional_key VARCHAR(500),
					zoom_meeting_url VARCHAR(500),
					zoom_meeting_id VARCHAR(100),
					zoom_passcode VARCHAR(50),
					start_time TIME NOT NULL,
					length_minutes INTEGER NOT NULL CHECK (length_minutes > 0),
					first_event_date DATE NOT NULL,
					repeat_pattern repeat_pattern NOT NULL DEFAULT 'none',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					status show_status DEFAULT 'active',
					user_id UUID NOT NULL,
					metadata JSONB,
					
					-- Constraints
					CONSTRAINT valid_first_event_date CHECK (first_event_date >= CURRENT_DATE),
					CONSTRAINT valid_length CHECK (length_minutes BETWEEN 1 AND 1440),
					CONSTRAINT valid_zoom_url CHECK (zoom_meeting_url IS NULL OR zoom_meeting_url ~ '^https://.*\.zoom\.us/.*')
				);

				-- Create indexes
				CREATE INDEX IF NOT EXISTS idx_shows_user_id ON shows(user_id);
				CREATE INDEX IF NOT EXISTS idx_shows_status ON shows(status);
				CREATE INDEX IF NOT EXISTS idx_shows_first_event_date ON shows(first_event_date);
				CREATE INDEX IF NOT EXISTS idx_shows_show_name ON shows(LOWER(show_name));
				CREATE INDEX IF NOT EXISTS idx_shows_zoom_meeting_id ON shows(zoom_meeting_id) WHERE zoom_meeting_id IS NOT NULL;

				-- Create update trigger for shows
				DROP TRIGGER IF EXISTS update_shows_updated_at ON shows;
				CREATE TRIGGER update_shows_updated_at BEFORE UPDATE ON shows
					FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
			`,
		},
		{
			Version:     6,
			Description: "Add advanced scheduling configuration for shows",
			SQL: `
				DO $$ 
				BEGIN
					-- Add scheduling_config column
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
								   WHERE table_name = 'shows' AND column_name = 'scheduling_config') THEN
						ALTER TABLE shows ADD COLUMN scheduling_config JSONB;
						
						-- Create index for scheduling queries
						CREATE INDEX idx_shows_scheduling_config ON shows USING GIN (scheduling_config);
						
						-- Migrate existing shows to new format
						-- For weekly and biweekly shows, use the weekday of first_event_date
						UPDATE shows SET scheduling_config = jsonb_build_object(
							'weekdays', ARRAY[EXTRACT(DOW FROM first_event_date)::int]
						) WHERE repeat_pattern IN ('weekly', 'biweekly') AND scheduling_config IS NULL;
						
						-- For monthly shows, use the day of month approach
						UPDATE shows SET scheduling_config = jsonb_build_object(
							'monthly_day', EXTRACT(DAY FROM first_event_date)::int,
							'monthly_day_fallback', 'last_day'
						) WHERE repeat_pattern = 'monthly' AND scheduling_config IS NULL;
						
						-- For daily and none patterns, no specific config needed
						UPDATE shows SET scheduling_config = jsonb_build_object()
						WHERE repeat_pattern IN ('daily', 'none') AND scheduling_config IS NULL;
					END IF;
				END $$;
			`,
		},
		{
			Version:     7,
			Description: "Add events and event_generation_logs tables for calendar system",
			SQL: `
				-- Add version column to shows table if not exists
				DO $$ 
				BEGIN
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
								   WHERE table_name = 'shows' AND column_name = 'version') THEN
						ALTER TABLE shows ADD COLUMN version INTEGER DEFAULT 1;
						-- Update existing shows to have version 1
						UPDATE shows SET version = 1 WHERE version IS NULL;
						ALTER TABLE shows ALTER COLUMN version SET NOT NULL;
					END IF;
				END $$;

				-- Create event status enum
				DO $$ BEGIN
					CREATE TYPE event_status AS ENUM ('scheduled', 'live', 'completed', 'cancelled', 'postponed');
				EXCEPTION
					WHEN duplicate_object THEN null;
				END $$;

				-- Create events table
				CREATE TABLE IF NOT EXISTS events (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					show_id UUID NOT NULL,
					user_id UUID NOT NULL,
					
					-- Event details (nullable = uses show defaults)
					event_title VARCHAR(500),
					event_description TEXT,
					youtube_key VARCHAR(500),
					additional_key VARCHAR(500),
					zoom_meeting_url VARCHAR(500),
					zoom_meeting_id VARCHAR(100),
					zoom_passcode VARCHAR(50),
					
					-- Timing
					start_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
					length_minutes INTEGER,
					end_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
					
					-- Event metadata
					status event_status DEFAULT 'scheduled',
					is_customized BOOLEAN DEFAULT FALSE,
					custom_fields JSONB,
					
					-- Generation tracking
					generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					last_synced_at TIMESTAMP WITH TIME ZONE,
					show_version INTEGER DEFAULT 1,
					
					-- Audit fields
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					
					-- Foreign keys
					FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE,
					
					-- Constraints
					CONSTRAINT valid_event_timing CHECK (end_datetime > start_datetime),
					CONSTRAINT valid_length CHECK (length_minutes IS NULL OR length_minutes > 0)
				);

				-- Create indexes for events table
				CREATE INDEX IF NOT EXISTS idx_events_show_id ON events(show_id);
				CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
				CREATE INDEX IF NOT EXISTS idx_events_start_datetime ON events(start_datetime);
				CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
				CREATE INDEX IF NOT EXISTS idx_events_date_range ON events(start_datetime, end_datetime);
				CREATE INDEX IF NOT EXISTS idx_events_user_status ON events(user_id, status);
				CREATE INDEX IF NOT EXISTS idx_events_customized ON events(is_customized);
				CREATE INDEX IF NOT EXISTS idx_events_show_version ON events(show_id, show_version);

				-- Create event generation logs table
				CREATE TABLE IF NOT EXISTS event_generation_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					show_id UUID NOT NULL,
					generation_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					events_generated INTEGER NOT NULL,
					generated_until TIMESTAMP WITH TIME ZONE NOT NULL,
					trigger_reason VARCHAR(100) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					
					FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE
				);

				-- Create indexes for event generation logs
				CREATE INDEX IF NOT EXISTS idx_generation_logs_show_id ON event_generation_logs(show_id);
				CREATE INDEX IF NOT EXISTS idx_generation_logs_date ON event_generation_logs(generation_date);
				CREATE INDEX IF NOT EXISTS idx_generation_logs_trigger ON event_generation_logs(trigger_reason);

				-- Create update trigger for events
				DROP TRIGGER IF EXISTS update_events_updated_at ON events;
				CREATE TRIGGER update_events_updated_at 
				BEFORE UPDATE ON events
				FOR EACH ROW 
				EXECUTE FUNCTION update_updated_at_column();
			`,
		},
		{
			Version:     8,
			Description: "Add guests table for guest management system",
			SQL: `
				-- Create contact type enum
				DO $$ BEGIN
					CREATE TYPE contact_type AS ENUM (
						'email', 'phone', 'telegram', 'discord', 'twitter', 
						'linkedin', 'instagram', 'website', 'other'
					);
				EXCEPTION
					WHEN duplicate_object THEN null;
				END $$;

				-- Install trigram extension for fuzzy search if not already installed
				CREATE EXTENSION IF NOT EXISTS pg_trgm;

				-- Create guests table
				CREATE TABLE IF NOT EXISTS guests (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					name VARCHAR(255) NOT NULL,
					surname VARCHAR(255) NOT NULL,
					short_name VARCHAR(100),
					contacts JSONB DEFAULT '[]'::jsonb,
					notes TEXT,
					avatar VARCHAR(500),
					tags JSONB DEFAULT '[]'::jsonb,
					metadata JSONB DEFAULT '{}'::jsonb,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					
					-- Constraints
					CONSTRAINT valid_name CHECK (LENGTH(TRIM(name)) > 0),
					CONSTRAINT valid_surname CHECK (LENGTH(TRIM(surname)) > 0),
					CONSTRAINT unique_user_guest UNIQUE(user_id, name, surname)
				);

				-- Create indexes for performance
				CREATE INDEX IF NOT EXISTS idx_guests_user_id ON guests(user_id);
				CREATE INDEX IF NOT EXISTS idx_guests_name ON guests(LOWER(name));
				CREATE INDEX IF NOT EXISTS idx_guests_surname ON guests(LOWER(surname));
				CREATE INDEX IF NOT EXISTS idx_guests_short_name ON guests(LOWER(short_name)) WHERE short_name IS NOT NULL;
				CREATE INDEX IF NOT EXISTS idx_guests_full_name ON guests(LOWER(name || ' ' || surname));
				CREATE INDEX IF NOT EXISTS idx_guests_contacts ON guests USING GIN (contacts);
				CREATE INDEX IF NOT EXISTS idx_guests_tags ON guests USING GIN (tags);
				CREATE INDEX IF NOT EXISTS idx_guests_search ON guests USING GIN (
					(LOWER(name) || ' ' || LOWER(surname) || ' ' || COALESCE(LOWER(short_name), '')) gin_trgm_ops
				);

				-- Create update trigger for guests
				DROP TRIGGER IF EXISTS update_guests_updated_at ON guests;
				CREATE TRIGGER update_guests_updated_at 
				BEFORE UPDATE ON guests
				FOR EACH ROW 
				EXECUTE FUNCTION update_updated_at_column();
			`,
		},
	}

	// Run each migration if not already applied
	for _, migration := range migrations {
		if err := p.runMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (p *PostgresDB) createMigrationsTable(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := p.pool.Exec(ctx, createTableSQL)
	return err
}

// runMigration executes a single migration if it hasn't been applied yet
func (p *PostgresDB) runMigration(ctx context.Context, migration Migration) error {
	// Check if migration has already been applied
	var exists bool
	checkSQL := "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)"
	err := p.pool.QueryRow(ctx, checkSQL, migration.Version).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if exists {
		// Migration already applied, skip
		utils.LogInfo(ctx, "Migration already applied", utils.Fields{
			"version":     migration.Version,
			"description": migration.Description,
		})
		return nil
	}

	// Start transaction for migration
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	utils.LogInfo(ctx, "Applying migration", utils.Fields{
		"version":     migration.Version,
		"description": migration.Description,
	})

	_, err = tx.Exec(ctx, migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	recordSQL := `
		INSERT INTO schema_migrations (version, description) 
		VALUES ($1, $2)
	`
	_, err = tx.Exec(ctx, recordSQL, migration.Version, migration.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	utils.LogInfo(ctx, "Migration applied successfully", utils.Fields{
		"version":     migration.Version,
		"description": migration.Description,
	})

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

// GetMigrationStatus returns the current migration status
func (p *PostgresDB) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
	// First check if migrations table exists
	var tableExists bool
	checkTableSQL := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'schema_migrations'
		)
	`
	err := p.pool.QueryRow(ctx, checkTableSQL).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check migrations table: %w", err)
	}

	if !tableExists {
		return []Migration{}, nil
	}

	// Get applied migrations
	query := `
		SELECT version, description, applied_at 
		FROM schema_migrations 
		ORDER BY version
	`
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		var appliedAt time.Time
		err := rows.Scan(&migration.Version, &migration.Description, &appliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		migrations = append(migrations, migration)
	}

	return migrations, nil
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

// Show operations
func (p *PostgresDB) CreateShow(ctx context.Context, show *models.Show) error {
	show.ID = uuid.New()
	show.CreatedAt = time.Now()
	show.UpdatedAt = time.Now()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(show.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert scheduling config to JSON
	var schedulingConfigJSON []byte
	if show.SchedulingConfig != nil {
		schedulingConfigJSON, err = json.Marshal(show.SchedulingConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal scheduling config: %w", err)
		}
	}

	query := `
		INSERT INTO shows (id, show_name, youtube_key, additional_key, zoom_meeting_url, 
			zoom_meeting_id, zoom_passcode, start_time, length_minutes, first_event_date, 
			repeat_pattern, scheduling_config, created_at, updated_at, status, user_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at`

	err = p.pool.QueryRow(ctx, query,
		show.ID, show.ShowName, show.YouTubeKey, show.AdditionalKey, show.ZoomMeetingURL,
		show.ZoomMeetingID, show.ZoomPasscode, show.StartTime, show.LengthMinutes, show.FirstEventDate,
		show.RepeatPattern, schedulingConfigJSON, show.CreatedAt, show.UpdatedAt, show.Status, show.UserID, metadataJSON,
	).Scan(&show.ID, &show.CreatedAt, &show.UpdatedAt)

	return err
}

func (p *PostgresDB) GetShowByID(ctx context.Context, showID uuid.UUID) (*models.Show, error) {
	show := &models.Show{}
	var metadataJSON []byte
	var schedulingConfigJSON []byte

	query := `
		SELECT id, show_name, youtube_key, additional_key, zoom_meeting_url, 
			zoom_meeting_id, zoom_passcode, start_time, length_minutes, first_event_date, 
			repeat_pattern, scheduling_config, created_at, updated_at, status, user_id, metadata
		FROM shows WHERE id = $1`

	err := p.pool.QueryRow(ctx, query, showID).Scan(
		&show.ID, &show.ShowName, &show.YouTubeKey, &show.AdditionalKey, &show.ZoomMeetingURL,
		&show.ZoomMeetingID, &show.ZoomPasscode, &show.StartTime, &show.LengthMinutes, &show.FirstEventDate,
		&show.RepeatPattern, &schedulingConfigJSON, &show.CreatedAt, &show.UpdatedAt, &show.Status, &show.UserID, &metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &show.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Unmarshal scheduling config if present
	if len(schedulingConfigJSON) > 0 {
		show.SchedulingConfig = &models.SchedulingConfig{}
		if err := json.Unmarshal(schedulingConfigJSON, show.SchedulingConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scheduling config: %w", err)
		}
	}

	return show, nil
}

func (p *PostgresDB) DeleteShow(ctx context.Context, showID uuid.UUID) error {
	// Soft delete by updating status
	query := `UPDATE shows SET status = $2 WHERE id = $1`
	
	result, err := p.pool.Exec(ctx, query, showID, models.ShowStatusCancelled)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no show found with ID: %s", showID)
	}

	return nil
}

func (p *PostgresDB) ListShows(ctx context.Context, userID uuid.UUID, filters models.ListShowsFilters, pagination models.PaginationOptions, sort models.ListShowsSortOptions) ([]models.Show, int, error) {
	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	offset := (pagination.Page - 1) * pagination.Limit

	// Build where clause
	whereConditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argCount := 1

	// Status filter
	if len(filters.Status) > 0 {
		argCount++
		statusPlaceholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			argCount++
			statusPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, status)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("status IN (%s)", strings.Join(statusPlaceholders, ",")))
	}

	// Repeat pattern filter
	if len(filters.RepeatPattern) > 0 {
		repeatPlaceholders := make([]string, len(filters.RepeatPattern))
		for i, pattern := range filters.RepeatPattern {
			argCount++
			repeatPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, pattern)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("repeat_pattern IN (%s)", strings.Join(repeatPlaceholders, ",")))
	}

	// Search filter
	if filters.Search != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("LOWER(show_name) LIKE LOWER($%d)", argCount))
		args = append(args, "%"+filters.Search+"%")
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM shows WHERE %s", whereClause)
	if err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order by clause
	orderBy := "created_at DESC" // default
	if sort.Field != "" {
		allowedFields := map[string]bool{
			"show_name":        true,
			"first_event_date": true,
			"created_at":       true,
			"updated_at":       true,
		}
		if allowedFields[sort.Field] {
			order := "ASC"
			if strings.ToUpper(sort.Order) == "DESC" {
				order = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", sort.Field, order)
		}
	}

	// Get shows
	query := fmt.Sprintf(`
		SELECT id, show_name, youtube_key, additional_key, zoom_meeting_url, 
			zoom_meeting_id, zoom_passcode, start_time, length_minutes, first_event_date, 
			repeat_pattern, scheduling_config, created_at, updated_at, status, user_id, metadata
		FROM shows 
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, argCount+1, argCount+2)

	args = append(args, pagination.Limit, offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shows []models.Show
	for rows.Next() {
		var show models.Show
		var metadataJSON []byte
		var schedulingConfigJSON []byte

		err := rows.Scan(
			&show.ID, &show.ShowName, &show.YouTubeKey, &show.AdditionalKey, &show.ZoomMeetingURL,
			&show.ZoomMeetingID, &show.ZoomPasscode, &show.StartTime, &show.LengthMinutes, &show.FirstEventDate,
			&show.RepeatPattern, &schedulingConfigJSON, &show.CreatedAt, &show.UpdatedAt, &show.Status, &show.UserID, &metadataJSON,
		)
		if err != nil {
			return nil, 0, err
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &show.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Unmarshal scheduling config if present
		if len(schedulingConfigJSON) > 0 {
			show.SchedulingConfig = &models.SchedulingConfig{}
			if err := json.Unmarshal(schedulingConfigJSON, show.SchedulingConfig); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal scheduling config: %w", err)
			}
		}

		shows = append(shows, show)
	}

	return shows, total, nil
}

func (p *PostgresDB) UpdateShow(ctx context.Context, show *models.Show) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(show.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert scheduling config to JSON
	var schedulingConfigJSON []byte
	if show.SchedulingConfig != nil {
		schedulingConfigJSON, err = json.Marshal(show.SchedulingConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal scheduling config: %w", err)
		}
	}

	query := `
		UPDATE shows SET 
			show_name = $2, youtube_key = $3, additional_key = $4, zoom_meeting_url = $5,
			zoom_meeting_id = $6, zoom_passcode = $7, start_time = $8, length_minutes = $9,
			first_event_date = $10, repeat_pattern = $11, scheduling_config = $12, status = $13, metadata = $14
		WHERE id = $1`

	_, err = p.pool.Exec(ctx, query,
		show.ID, show.ShowName, show.YouTubeKey, show.AdditionalKey, show.ZoomMeetingURL,
		show.ZoomMeetingID, show.ZoomPasscode, show.StartTime, show.LengthMinutes,
		show.FirstEventDate, show.RepeatPattern, schedulingConfigJSON, show.Status, metadataJSON,
	)
	return err
}

// Event operations

func (p *PostgresDB) CreateEvent(ctx context.Context, event *models.Event) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	event.GeneratedAt = time.Now()

	// Convert custom fields to JSON
	var customFieldsJSON []byte
	var err error
	if event.CustomFields != nil {
		customFieldsJSON, err = json.Marshal(event.CustomFields)
		if err != nil {
			return fmt.Errorf("failed to marshal custom fields: %w", err)
		}
	}

	query := `
		INSERT INTO events (id, show_id, user_id, event_title, event_description, 
			youtube_key, additional_key, zoom_meeting_url, zoom_meeting_id, zoom_passcode,
			start_datetime, length_minutes, end_datetime, status, is_customized, 
			custom_fields, generated_at, last_synced_at, show_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id, created_at, updated_at, generated_at`

	err = p.pool.QueryRow(ctx, query,
		event.ID, event.ShowID, event.UserID, event.EventTitle, event.EventDescription,
		event.YouTubeKey, event.AdditionalKey, event.ZoomMeetingURL, event.ZoomMeetingID, event.ZoomPasscode,
		event.StartDateTime, event.LengthMinutes, event.EndDateTime, event.Status, event.IsCustomized,
		customFieldsJSON, event.GeneratedAt, event.LastSyncedAt, event.ShowVersion, event.CreatedAt, event.UpdatedAt,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt, &event.GeneratedAt)

	return err
}

func (p *PostgresDB) CreateEvents(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO events (id, show_id, user_id, event_title, event_description, 
			youtube_key, additional_key, zoom_meeting_url, zoom_meeting_id, zoom_passcode,
			start_datetime, length_minutes, end_datetime, status, is_customized, 
			custom_fields, generated_at, last_synced_at, show_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	for i := range events {
		event := &events[i]
		event.ID = uuid.New()
		event.CreatedAt = time.Now()
		event.UpdatedAt = time.Now()
		if event.GeneratedAt.IsZero() {
			event.GeneratedAt = time.Now()
		}

		// Convert custom fields to JSON
		var customFieldsJSON []byte
		if event.CustomFields != nil {
			customFieldsJSON, err = json.Marshal(event.CustomFields)
			if err != nil {
				return fmt.Errorf("failed to marshal custom fields: %w", err)
			}
		}

		_, err = tx.Exec(ctx, query,
			event.ID, event.ShowID, event.UserID, event.EventTitle, event.EventDescription,
			event.YouTubeKey, event.AdditionalKey, event.ZoomMeetingURL, event.ZoomMeetingID, event.ZoomPasscode,
			event.StartDateTime, event.LengthMinutes, event.EndDateTime, event.Status, event.IsCustomized,
			customFieldsJSON, event.GeneratedAt, event.LastSyncedAt, event.ShowVersion, event.CreatedAt, event.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

func (p *PostgresDB) GetEventByID(ctx context.Context, eventID uuid.UUID) (*models.Event, error) {
	event := &models.Event{}
	var customFieldsJSON []byte

	query := `
		SELECT id, show_id, user_id, event_title, event_description, 
			youtube_key, additional_key, zoom_meeting_url, zoom_meeting_id, zoom_passcode,
			start_datetime, length_minutes, end_datetime, status, is_customized, 
			custom_fields, generated_at, last_synced_at, show_version, created_at, updated_at
		FROM events WHERE id = $1`

	err := p.pool.QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.ShowID, &event.UserID, &event.EventTitle, &event.EventDescription,
		&event.YouTubeKey, &event.AdditionalKey, &event.ZoomMeetingURL, &event.ZoomMeetingID, &event.ZoomPasscode,
		&event.StartDateTime, &event.LengthMinutes, &event.EndDateTime, &event.Status, &event.IsCustomized,
		&customFieldsJSON, &event.GeneratedAt, &event.LastSyncedAt, &event.ShowVersion, &event.CreatedAt, &event.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal custom fields if present
	if len(customFieldsJSON) > 0 {
		if err := json.Unmarshal(customFieldsJSON, &event.CustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal custom fields: %w", err)
		}
	}

	return event, nil
}

func (p *PostgresDB) UpdateEvent(ctx context.Context, event *models.Event) error {
	// Convert custom fields to JSON
	var customFieldsJSON []byte
	var err error
	if event.CustomFields != nil {
		customFieldsJSON, err = json.Marshal(event.CustomFields)
		if err != nil {
			return fmt.Errorf("failed to marshal custom fields: %w", err)
		}
	}

	query := `
		UPDATE events SET 
			event_title = $2, event_description = $3, youtube_key = $4, additional_key = $5,
			zoom_meeting_url = $6, zoom_meeting_id = $7, zoom_passcode = $8,
			start_datetime = $9, length_minutes = $10, end_datetime = $11,
			status = $12, is_customized = $13, custom_fields = $14, last_synced_at = $15
		WHERE id = $1`

	now := time.Now()
	_, err = p.pool.Exec(ctx, query,
		event.ID, event.EventTitle, event.EventDescription, event.YouTubeKey, event.AdditionalKey,
		event.ZoomMeetingURL, event.ZoomMeetingID, event.ZoomPasscode,
		event.StartDateTime, event.LengthMinutes, event.EndDateTime,
		event.Status, event.IsCustomized, customFieldsJSON, &now,
	)
	return err
}

func (p *PostgresDB) DeleteEvent(ctx context.Context, eventID uuid.UUID) error {
	// Soft delete by updating status to cancelled
	query := `UPDATE events SET status = $2 WHERE id = $1`
	
	result, err := p.pool.Exec(ctx, query, eventID, models.EventStatusCancelled)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no event found with ID: %s", eventID)
	}

	return nil
}

func (p *PostgresDB) GetEventsByShowID(ctx context.Context, showID uuid.UUID) ([]models.Event, error) {
	query := `
		SELECT id, show_id, user_id, event_title, event_description, 
			youtube_key, additional_key, zoom_meeting_url, zoom_meeting_id, zoom_passcode,
			start_datetime, length_minutes, end_datetime, status, is_customized, 
			custom_fields, generated_at, last_synced_at, show_version, created_at, updated_at
		FROM events 
		WHERE show_id = $1
		ORDER BY start_datetime ASC`

	rows, err := p.pool.Query(ctx, query, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return p.scanEvents(rows)
}

func (p *PostgresDB) GetFutureEvents(ctx context.Context, showID uuid.UUID) ([]models.Event, error) {
	query := `
		SELECT id, show_id, user_id, event_title, event_description, 
			youtube_key, additional_key, zoom_meeting_url, zoom_meeting_id, zoom_passcode,
			start_datetime, length_minutes, end_datetime, status, is_customized, 
			custom_fields, generated_at, last_synced_at, show_version, created_at, updated_at
		FROM events 
		WHERE show_id = $1 AND start_datetime > $2 AND status != $3
		ORDER BY start_datetime ASC`

	rows, err := p.pool.Query(ctx, query, showID, time.Now(), models.EventStatusCancelled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return p.scanEvents(rows)
}

func (p *PostgresDB) ListEvents(ctx context.Context, userID uuid.UUID, filters models.EventFilters, pagination models.PaginationOptions, sort models.EventSortOptions) ([]models.EventListItem, int, error) {
	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	offset := (pagination.Page - 1) * pagination.Limit

	// Build where clause
	whereConditions := []string{"e.user_id = $1"}
	args := []interface{}{userID}
	argCount := 1

	// Status filter
	if len(filters.Status) > 0 {
		statusPlaceholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			argCount++
			statusPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, status)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("e.status IN (%s)", strings.Join(statusPlaceholders, ",")))
	}

	// Show IDs filter
	if len(filters.ShowIDs) > 0 {
		showPlaceholders := make([]string, len(filters.ShowIDs))
		for i, showID := range filters.ShowIDs {
			argCount++
			showPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, showID)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("e.show_id IN (%s)", strings.Join(showPlaceholders, ",")))
	}

	// Date range filter
	if filters.DateRange != nil {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("e.start_datetime >= $%d", argCount))
		args = append(args, filters.DateRange.Start)
		
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("e.start_datetime <= $%d", argCount))
		args = append(args, filters.DateRange.End)
	}

	// Customization filter
	if filters.IsCustomized != nil {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("e.is_customized = $%d", argCount))
		args = append(args, *filters.IsCustomized)
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM events e 
		JOIN shows s ON e.show_id = s.id 
		WHERE %s`, whereClause)
	if err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order by clause
	orderBy := "e.start_datetime ASC" // default
	if sort.Field != "" {
		allowedFields := map[string]bool{
			"start_datetime": true,
			"end_datetime":   true,
			"status":         true,
			"created_at":     true,
		}
		if allowedFields[sort.Field] {
			order := "ASC"
			if strings.ToUpper(sort.Order) == "DESC" {
				order = "DESC"
			}
			orderBy = fmt.Sprintf("e.%s %s", sort.Field, order)
		}
	}

	// Get events
	query := fmt.Sprintf(`
		SELECT e.id, e.show_id, s.show_name, e.event_title, e.start_datetime, e.end_datetime,
			e.status, e.is_customized, 
			CASE WHEN s.zoom_meeting_url IS NOT NULL OR e.zoom_meeting_url IS NOT NULL THEN true ELSE false END as has_zoom_meeting
		FROM events e 
		JOIN shows s ON e.show_id = s.id 
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, argCount+1, argCount+2)

	args = append(args, pagination.Limit, offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.EventListItem
	for rows.Next() {
		var event models.EventListItem
		err := rows.Scan(
			&event.ID, &event.ShowID, &event.ShowName, &event.EventTitle,
			&event.StartDateTime, &event.EndDateTime, &event.Status,
			&event.IsCustomized, &event.HasZoomMeeting,
		)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}

	return events, total, nil
}

func (p *PostgresDB) GetWeekEvents(ctx context.Context, userID uuid.UUID, weekStart time.Time, filters models.EventFilters) ([]models.WeekDayEvent, error) {
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Build where clause
	whereConditions := []string{"e.user_id = $1", "e.start_datetime >= $2", "e.start_datetime < $3"}
	args := []interface{}{userID, weekStart, weekEnd}
	argCount := 3

	// Add additional filters similar to ListEvents
	if len(filters.Status) > 0 {
		statusPlaceholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			argCount++
			statusPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, status)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("e.status IN (%s)", strings.Join(statusPlaceholders, ",")))
	}

	if len(filters.ShowIDs) > 0 {
		showPlaceholders := make([]string, len(filters.ShowIDs))
		for i, showID := range filters.ShowIDs {
			argCount++
			showPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, showID)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("e.show_id IN (%s)", strings.Join(showPlaceholders, ",")))
	}

	whereClause := strings.Join(whereConditions, " AND ")

	query := fmt.Sprintf(`
		SELECT e.id, s.show_name, e.event_title, e.start_datetime, e.end_datetime, e.status, e.is_customized
		FROM events e 
		JOIN shows s ON e.show_id = s.id 
		WHERE %s
		ORDER BY e.start_datetime ASC`, whereClause)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.WeekDayEvent
	for rows.Next() {
		var event models.WeekDayEvent
		var startTime, endTime time.Time
		err := rows.Scan(
			&event.ID, &event.ShowName, &event.EventTitle,
			&startTime, &endTime, &event.Status, &event.IsCustomized,
		)
		if err != nil {
			return nil, err
		}

		// Format times
		event.StartTime = startTime.Format("15:04")
		event.EndTime = endTime.Format("15:04")
		events = append(events, event)
	}

	return events, nil
}

func (p *PostgresDB) GetMonthEvents(ctx context.Context, userID uuid.UUID, year int, month int, filters models.EventFilters) ([]models.MonthDayEvent, error) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Build where clause similar to GetWeekEvents
	whereConditions := []string{"e.user_id = $1", "e.start_datetime >= $2", "e.start_datetime < $3"}
	args := []interface{}{userID, monthStart, monthEnd}
	argCount := 3

	if len(filters.Status) > 0 {
		statusPlaceholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			argCount++
			statusPlaceholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, status)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("e.status IN (%s)", strings.Join(statusPlaceholders, ",")))
	}

	whereClause := strings.Join(whereConditions, " AND ")

	query := fmt.Sprintf(`
		SELECT e.id, s.show_name, e.start_datetime, e.length_minutes, e.status, e.is_customized, s.length_minutes
		FROM events e 
		JOIN shows s ON e.show_id = s.id 
		WHERE %s
		ORDER BY e.start_datetime ASC`, whereClause)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.MonthDayEvent
	for rows.Next() {
		var event models.MonthDayEvent
		var startTime time.Time
		var eventDuration *int
		var showDuration int
		err := rows.Scan(
			&event.ID, &event.ShowName, &startTime,
			&eventDuration, &event.Status, &event.IsCustomized, &showDuration,
		)
		if err != nil {
			return nil, err
		}

		// Use event duration if available, otherwise show duration
		if eventDuration != nil {
			event.DurationMinutes = *eventDuration
		} else {
			event.DurationMinutes = showDuration
		}

		event.StartTime = startTime.Format("15:04")
		events = append(events, event)
	}

	return events, nil
}

// Helper function to scan events from rows
func (p *PostgresDB) scanEvents(rows pgx.Rows) ([]models.Event, error) {
	var events []models.Event
	for rows.Next() {
		var event models.Event
		var customFieldsJSON []byte

		err := rows.Scan(
			&event.ID, &event.ShowID, &event.UserID, &event.EventTitle, &event.EventDescription,
			&event.YouTubeKey, &event.AdditionalKey, &event.ZoomMeetingURL, &event.ZoomMeetingID, &event.ZoomPasscode,
			&event.StartDateTime, &event.LengthMinutes, &event.EndDateTime, &event.Status, &event.IsCustomized,
			&customFieldsJSON, &event.GeneratedAt, &event.LastSyncedAt, &event.ShowVersion, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal custom fields if present
		if len(customFieldsJSON) > 0 {
			if err := json.Unmarshal(customFieldsJSON, &event.CustomFields); err != nil {
				return nil, fmt.Errorf("failed to unmarshal custom fields: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// Event generation log operations

func (p *PostgresDB) CreateEventGenerationLog(ctx context.Context, log *models.EventGenerationLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	if log.GenerationDate.IsZero() {
		log.GenerationDate = time.Now()
	}

	query := `
		INSERT INTO event_generation_logs (id, show_id, generation_date, events_generated, 
			generated_until, trigger_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := p.pool.QueryRow(ctx, query,
		log.ID, log.ShowID, log.GenerationDate, log.EventsGenerated,
		log.GeneratedUntil, log.TriggerReason, log.CreatedAt,
	).Scan(&log.ID, &log.CreatedAt)

	return err
}

func (p *PostgresDB) GetLastGenerationDate(ctx context.Context, showID uuid.UUID) (time.Time, error) {
	var lastDate time.Time
	query := `
		SELECT generated_until 
		FROM event_generation_logs 
		WHERE show_id = $1 
		ORDER BY generation_date DESC 
		LIMIT 1`

	err := p.pool.QueryRow(ctx, query, showID).Scan(&lastDate)
	if err == pgx.ErrNoRows {
		// Return zero time if no logs found
		return time.Time{}, nil
	}
	return lastDate, err
}

func (p *PostgresDB) GetActiveShows(ctx context.Context) ([]models.Show, error) {
	query := `
		SELECT id, show_name, youtube_key, additional_key, zoom_meeting_url, 
			zoom_meeting_id, zoom_passcode, start_time, length_minutes, first_event_date, 
			repeat_pattern, scheduling_config, version, created_at, updated_at, status, user_id, metadata
		FROM shows 
		WHERE status = $1
		ORDER BY created_at ASC`

	rows, err := p.pool.Query(ctx, query, models.ShowStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []models.Show
	for rows.Next() {
		var show models.Show
		var metadataJSON []byte
		var schedulingConfigJSON []byte

		err := rows.Scan(
			&show.ID, &show.ShowName, &show.YouTubeKey, &show.AdditionalKey, &show.ZoomMeetingURL,
			&show.ZoomMeetingID, &show.ZoomPasscode, &show.StartTime, &show.LengthMinutes, &show.FirstEventDate,
			&show.RepeatPattern, &schedulingConfigJSON, &show.Version, &show.CreatedAt, &show.UpdatedAt, &show.Status, &show.UserID, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &show.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Unmarshal scheduling config if present
		if len(schedulingConfigJSON) > 0 {
			show.SchedulingConfig = &models.SchedulingConfig{}
			if err := json.Unmarshal(schedulingConfigJSON, show.SchedulingConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal scheduling config: %w", err)
			}
		}

		shows = append(shows, show)
	}

	return shows, nil
}

// Guest operations

func (p *PostgresDB) CreateGuest(ctx context.Context, guest *models.Guest) error {
	guest.ID = uuid.New()
	guest.CreatedAt = time.Now()
	guest.UpdatedAt = time.Now()

	// Convert contacts, tags, and metadata to JSON
	contactsJSON, err := json.Marshal(guest.Contacts)
	if err != nil {
		return fmt.Errorf("failed to marshal contacts: %w", err)
	}

	tagsJSON, err := json.Marshal(guest.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(guest.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO guests (id, user_id, name, surname, short_name, contacts, 
			notes, avatar, tags, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err = p.pool.QueryRow(ctx, query,
		guest.ID, guest.UserID, guest.Name, guest.Surname, guest.ShortName,
		contactsJSON, guest.Notes, guest.Avatar, tagsJSON, metadataJSON,
		guest.CreatedAt, guest.UpdatedAt,
	).Scan(&guest.ID, &guest.CreatedAt, &guest.UpdatedAt)

	return err
}

func (p *PostgresDB) GetGuestByID(ctx context.Context, guestID uuid.UUID) (*models.Guest, error) {
	guest := &models.Guest{}
	var contactsJSON, tagsJSON, metadataJSON []byte

	query := `
		SELECT id, user_id, name, surname, short_name, contacts, notes, avatar, 
			tags, metadata, created_at, updated_at
		FROM guests WHERE id = $1`

	err := p.pool.QueryRow(ctx, query, guestID).Scan(
		&guest.ID, &guest.UserID, &guest.Name, &guest.Surname, &guest.ShortName,
		&contactsJSON, &guest.Notes, &guest.Avatar, &tagsJSON, &metadataJSON,
		&guest.CreatedAt, &guest.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(contactsJSON) > 0 {
		if err := json.Unmarshal(contactsJSON, &guest.Contacts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contacts: %w", err)
		}
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &guest.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &guest.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return guest, nil
}

func (p *PostgresDB) UpdateGuest(ctx context.Context, guest *models.Guest) error {
	// Convert contacts, tags, and metadata to JSON
	contactsJSON, err := json.Marshal(guest.Contacts)
	if err != nil {
		return fmt.Errorf("failed to marshal contacts: %w", err)
	}

	tagsJSON, err := json.Marshal(guest.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(guest.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE guests SET 
			name = $2, surname = $3, short_name = $4, contacts = $5,
			notes = $6, avatar = $7, tags = $8, metadata = $9
		WHERE id = $1`

	_, err = p.pool.Exec(ctx, query,
		guest.ID, guest.Name, guest.Surname, guest.ShortName, contactsJSON,
		guest.Notes, guest.Avatar, tagsJSON, metadataJSON,
	)
	return err
}

func (p *PostgresDB) DeleteGuest(ctx context.Context, guestID uuid.UUID) error {
	query := `DELETE FROM guests WHERE id = $1`
	
	result, err := p.pool.Exec(ctx, query, guestID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with ID: %s", guestID)
	}

	return nil
}

func (p *PostgresDB) ListGuests(ctx context.Context, userID uuid.UUID, filters models.GuestFilters, pagination models.PaginationOptions, sort models.GuestSortOptions) ([]models.GuestListItem, int, error) {
	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	offset := (pagination.Page - 1) * pagination.Limit

	// Build where clause
	whereConditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argCount := 1

	// Search filter
	if filters.Search != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf(`(
			LOWER(name) LIKE LOWER($%d) OR
			LOWER(surname) LIKE LOWER($%d) OR
			LOWER(short_name) LIKE LOWER($%d) OR
			LOWER(name || ' ' || surname) LIKE LOWER($%d)
		)`, argCount, argCount, argCount, argCount))
		searchPattern := "%" + filters.Search + "%"
		args = append(args, searchPattern)
	}

	// Tags filter
	if len(filters.Tags) > 0 {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("tags ?& $%d", argCount))
		args = append(args, filters.Tags)
	}

	// Contact type filter
	if len(filters.HasContactType) > 0 {
		contactConditions := make([]string, len(filters.HasContactType))
		for i, contactType := range filters.HasContactType {
			argCount++
			contactConditions[i] = fmt.Sprintf("contacts @> $%d", argCount)
			args = append(args, fmt.Sprintf(`[{"type": "%s"}]`, contactType))
		}
		whereConditions = append(whereConditions, "("+strings.Join(contactConditions, " OR ")+")")
	}

	// Date range filter
	if filters.CreatedDateRange != nil {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("created_at >= $%d", argCount))
		args = append(args, filters.CreatedDateRange.Start)
		
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("created_at <= $%d", argCount))
		args = append(args, filters.CreatedDateRange.End)
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM guests WHERE %s", whereClause)
	if err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order by clause
	orderBy := "name ASC, surname ASC" // default
	if sort.Field != "" {
		allowedFields := map[string]bool{
			"name":       true,
			"surname":    true,
			"short_name": true,
			"created_at": true,
			"updated_at": true,
		}
		if allowedFields[sort.Field] {
			order := "ASC"
			if strings.ToUpper(sort.Order) == "DESC" {
				order = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", sort.Field, order)
		}
	}

	// Get guests with summary data
	query := fmt.Sprintf(`
		SELECT id, name, surname, short_name, contacts, avatar, tags, 
			LEFT(notes, 100) as notes_preview, 
			jsonb_array_length(COALESCE(contacts, '[]'::jsonb)) as contact_count,
			created_at
		FROM guests 
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, argCount+1, argCount+2)

	args = append(args, pagination.Limit, offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var guests []models.GuestListItem
	for rows.Next() {
		var guest models.GuestListItem
		var contactsJSON, tagsJSON []byte
		var notesPreview sql.NullString

		err := rows.Scan(
			&guest.ID, &guest.Name, &guest.Surname, &guest.ShortName,
			&contactsJSON, &guest.Avatar, &tagsJSON, &notesPreview,
			&guest.ContactCount, &guest.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		// Process notes preview
		if notesPreview.Valid {
			guest.NotesPreview = &notesPreview.String
		}

		// Unmarshal contacts to find primary email
		var contacts []models.GuestContact
		if len(contactsJSON) > 0 {
			json.Unmarshal(contactsJSON, &contacts)
			for _, contact := range contacts {
				if contact.Type == models.ContactTypeEmail && contact.IsPrimary {
					guest.PrimaryEmail = &contact.Value
					break
				}
			}
			// If no primary email found, use first email
			if guest.PrimaryEmail == nil {
				for _, contact := range contacts {
					if contact.Type == models.ContactTypeEmail {
						guest.PrimaryEmail = &contact.Value
						break
					}
				}
			}
		}

		// Unmarshal tags
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &guest.Tags)
		}

		guests = append(guests, guest)
	}

	return guests, total, nil
}

func (p *PostgresDB) SearchGuests(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.GuestSuggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	searchQuery := `
		SELECT g.id, g.name, g.surname, g.short_name, g.avatar, g.contacts, g.tags,
			similarity(LOWER(g.name || ' ' || g.surname || ' ' || COALESCE(g.short_name, '')), LOWER($2)) as score
		FROM guests g
		WHERE g.user_id = $1 
		AND (
			LOWER(g.name) LIKE LOWER($2 || '%') OR
			LOWER(g.surname) LIKE LOWER($2 || '%') OR
			LOWER(g.short_name) LIKE LOWER($2 || '%') OR
			LOWER(g.name || ' ' || g.surname) LIKE LOWER('%' || $2 || '%')
		)
		ORDER BY score DESC, g.name ASC, g.surname ASC
		LIMIT $3`

	rows, err := p.pool.Query(ctx, searchQuery, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []models.GuestSuggestion
	for rows.Next() {
		var suggestion models.GuestSuggestion
		var contactsJSON, tagsJSON []byte
		var score float64

		err := rows.Scan(
			&suggestion.ID, &suggestion.Name, &suggestion.Surname, &suggestion.ShortName,
			&suggestion.Avatar, &contactsJSON, &tagsJSON, &score,
		)
		if err != nil {
			return nil, err
		}

		// Build display name
		displayName := suggestion.Name + " " + suggestion.Surname
		if suggestion.ShortName != nil && *suggestion.ShortName != "" {
			displayName += " (" + *suggestion.ShortName + ")"
		}
		suggestion.DisplayName = displayName

		// Parse contacts and find primary
		var contacts []models.GuestContact
		if len(contactsJSON) > 0 {
			json.Unmarshal(contactsJSON, &contacts)
			for _, contact := range contacts {
				if contact.IsPrimary {
					suggestion.PrimaryContact = &contact
					break
				}
			}
			// If no primary contact found, use first contact
			if suggestion.PrimaryContact == nil && len(contacts) > 0 {
				suggestion.PrimaryContact = &contacts[0]
			}
		}

		// Parse tags
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &suggestion.Tags)
		}

		suggestion.MatchScore = score
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}
