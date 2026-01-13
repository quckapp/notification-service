package db

import (
	"context"
	"fmt"
	"time"

	"github.com/quickchat/notification-service/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB wraps the MongoDB client and database
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg config.MongoDBConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client options
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(30 * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

// Close disconnects from MongoDB
func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

// Collection returns a MongoDB collection
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}
