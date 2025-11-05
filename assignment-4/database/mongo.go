package database

import (
	"context"
	"fmt"

	"github.com/stupside/DATA-intensive/assignment-4/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectMongo creates and returns a configured MongoDB client
func ConnectMongo(ctx context.Context, cfg *config.Config) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.Database.MongoDB.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

// CloseMongo closes the MongoDB connection
func CloseMongo(ctx context.Context, client *mongo.Client) error {
	if client == nil {
		return nil
	}

	if err := client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect MongoDB: %w", err)
	}

	return nil
}
