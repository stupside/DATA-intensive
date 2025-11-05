package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// StreamerRepository handles streamer-related MongoDB operations
type StreamerRepository struct {
	*MongoRepository
}

// NewStreamerRepository creates a new streamer repository
func NewStreamerRepository(client *mongo.Client, dbName string) *StreamerRepository {
	return &StreamerRepository{MongoRepository: NewMongoRepository(client, dbName)}
}

// Create creates a new streamer document
func (r *StreamerRepository) Create(ctx context.Context, streamer *StreamerDoc) error {
	_, err := r.Collection(StreamerCollection).InsertOne(ctx, streamer)
	return err
}

// FindByID finds a streamer by its ObjectID
func (r *StreamerRepository) FindByID(ctx context.Context, id bson.ObjectID) (*StreamerDoc, error) {
	var doc StreamerDoc
	err := r.Collection(StreamerCollection).FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// FindAll finds all streamers
func (r *StreamerRepository) FindAll(ctx context.Context) ([]StreamerDoc, error) {
	cursor, err := r.Collection(StreamerCollection).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var streamers []StreamerDoc
	if err := cursor.All(ctx, &streamers); err != nil {
		return nil, err
	}
	return streamers, nil
}
