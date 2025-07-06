# Database Schema Documentation

## Overview
This document describes the PostgreSQL database schema for stPlaner, a media downloading service that processes content from Telegram and YouTube.

## Schema Diagram

The database schema is defined in [database-schema.dbml](database-schema.dbml) using DBML (Database Markup Language) format, which can be visualized at [dbdiagram.io](https://dbdiagram.io).

### Quick Visualization
1. Copy the contents of `database-schema.dbml`
2. Go to [dbdiagram.io](https://dbdiagram.io)
3. Paste the DBML code to see the interactive diagram

## Database Structure

### Core Tables

#### 1. posts
Central table storing information about content to be downloaded.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Auto-generated UUID |
| content_id | varchar(255) | UNIQUE, NOT NULL | Unique content identifier |
| telegram_link | varchar(500) | UNIQUE, NOT NULL | Source Telegram URL |
| channel_name | varchar(255) | NOT NULL | Channel username (e.g., @channel) |
| original_channel_name | varchar(255) | NOT NULL | Channel display name |
| message_id | bigint | NOT NULL | Telegram message ID |
| created_at | timestamptz | DEFAULT CURRENT_TIMESTAMP | Creation time |
| updated_at | timestamptz | DEFAULT CURRENT_TIMESTAMP | Last update time |
| status | post_status | DEFAULT 'pending' | Processing status |
| media_count | integer | DEFAULT 0 | Number of media files |
| total_size | bigint | DEFAULT 0 | Total size in bytes |
| error_message | text | NULL | Error details if failed |

#### 2. media
Stores individual media files associated with posts.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Auto-generated UUID |
| media_id | varchar(255) | UNIQUE, NOT NULL | Unique media identifier |
| content_id | varchar(255) | FOREIGN KEY | Links to posts.content_id |
| telegram_file_id | varchar(500) | NOT NULL | Telegram file reference |
| file_name | varchar(500) | NOT NULL | System file name |
| original_file_name | varchar(500) | NOT NULL | Original file name |
| file_type | varchar(100) | NOT NULL | MIME type |
| file_size | bigint | NOT NULL | Size in bytes |
| s3_bucket | varchar(255) | NOT NULL | AWS S3 bucket |
| s3_key | varchar(500) | NOT NULL | S3 object path |
| file_hash | varchar(255) | NOT NULL | SHA-256 hash |
| downloaded_at | timestamptz | DEFAULT CURRENT_TIMESTAMP | Download time |
| metadata | jsonb | NULL | Additional metadata |

#### 3. schema_migrations
Tracks database schema versions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| version | integer | PRIMARY KEY | Migration version |
| description | varchar(255) | NOT NULL | Migration description |
| applied_at | timestamptz | DEFAULT CURRENT_TIMESTAMP | Application time |

### Custom Types

```sql
CREATE TYPE post_status AS ENUM ('pending', 'processing', 'completed', 'failed');
```

### Relationships

- **posts → media**: One-to-Many relationship
  - Foreign Key: `media.content_id` references `posts.content_id`
  - On Delete: CASCADE (deleting a post removes all associated media)

### Indexes

#### posts table indexes:
- `idx_posts_content_id` - Fast content lookup
- `idx_posts_channel_name` - Filter by channel
- `idx_posts_original_channel_name` - Filter by original name
- `idx_posts_created_at` - Sort by date (DESC)
- `idx_posts_status` - Filter by status

#### media table indexes:
- `idx_media_media_id` - Fast media lookup
- `idx_media_content_id` - Find media by post
- `idx_media_file_hash` - Deduplication
- `idx_media_telegram_file_id` - Telegram lookups
- `idx_media_original_file_name` - Name searches

## Required Extensions

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- Cryptographic functions
```

## Triggers

### update_posts_updated_at
Automatically updates the `updated_at` timestamp when a post is modified.

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_posts_updated_at 
BEFORE UPDATE ON posts
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
```

## Sample Queries

### Get all media for a post
```sql
SELECT m.*, p.channel_name, p.status
FROM media m
JOIN posts p ON m.content_id = p.content_id
WHERE p.content_id = 'example-content-id';
```

### Find duplicate files
```sql
SELECT file_hash, COUNT(*) as count, 
       STRING_AGG(original_file_name, ', ') as files
FROM media
GROUP BY file_hash
HAVING COUNT(*) > 1;
```

### Get processing statistics
```sql
SELECT status, COUNT(*) as count,
       SUM(media_count) as total_media,
       SUM(total_size) as total_bytes
FROM posts
GROUP BY status;
```

## Performance Considerations

1. **UUID Primary Keys**: Distributed system friendly, no sequence bottlenecks
2. **JSONB Metadata**: Flexible schema without migrations
3. **Content Hashing**: Enables efficient deduplication
4. **Indexed Foreign Keys**: Fast join operations
5. **Partial Indexes**: Consider adding partial indexes for common queries

## Backup and Maintenance

1. **Regular Backups**: Use pg_dump for consistent backups
2. **VACUUM**: Regular VACUUM ANALYZE for statistics
3. **Index Maintenance**: Monitor index bloat
4. **Archiving**: Consider partitioning for old posts