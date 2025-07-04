// MongoDB initialization script for Docker
// This script creates the database and collections with proper indexes

// Switch to telegram_media database
db = db.getSiblingDB('telegram_media');

// Create collections
db.createCollection('posts');
db.createCollection('media');

// Create indexes for posts collection
db.posts.createIndex({ "post_id": 1 }, { unique: true });
db.posts.createIndex({ "telegram_link": 1 }, { unique: true });
db.posts.createIndex({ "channel_name": 1 });
db.posts.createIndex({ "created_at": -1 });
db.posts.createIndex({ "status": 1 });

// Create indexes for media collection
db.media.createIndex({ "media_id": 1 }, { unique: true });
db.media.createIndex({ "post_id": 1 });
db.media.createIndex({ "file_hash": 1 });
db.media.createIndex({ "telegram_file_id": 1 });

print('Database and collections initialized successfully');