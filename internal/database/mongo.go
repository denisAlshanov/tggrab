package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/tggrab/tggrab/internal/config"
)

type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	posts    *mongo.Collection
	media    *mongo.Collection
}

func NewMongoDB(cfg *config.MongoDBConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	mongodb := &MongoDB{
		client:   client,
		database: db,
		posts:    db.Collection("posts"),
		media:    db.Collection("media"),
	}

	// Create indexes
	if err := mongodb.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return mongodb, nil
}

func (m *MongoDB) createIndexes(ctx context.Context) error {
	// Create indexes for posts collection
	postsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "post_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "telegram_link", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "channel_name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	}

	if _, err := m.posts.Indexes().CreateMany(ctx, postsIndexes); err != nil {
		return fmt.Errorf("failed to create posts indexes: %w", err)
	}

	// Create indexes for media collection
	mediaIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "media_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "post_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "file_hash", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "telegram_file_id", Value: 1}},
		},
	}

	if _, err := m.media.Indexes().CreateMany(ctx, mediaIndexes); err != nil {
		return fmt.Errorf("failed to create media indexes: %w", err)
	}

	return nil
}

func (m *MongoDB) Posts() *mongo.Collection {
	return m.posts
}

func (m *MongoDB) Media() *mongo.Collection {
	return m.media
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *MongoDB) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sessCtx mongo.SessionContext) error {
		_, err := session.WithTransaction(sessCtx, func(sessCtx mongo.SessionContext) (interface{}, error) {
			return nil, fn(sessCtx)
		})
		return err
	})
}

func (m *MongoDB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.client.Ping(ctx, readpref.Primary())
}
