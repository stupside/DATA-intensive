package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RelayRepository handles relay-related MongoDB operations
type RelayRepository struct {
	*MongoRepository
}

// NewRelayRepository creates a new relay repository
func NewRelayRepository(client *mongo.Client, dbName string) *RelayRepository {
	return &RelayRepository{MongoRepository: NewMongoRepository(client, dbName)}
}

// Create creates a new relay document
func (r *RelayRepository) Create(ctx context.Context, relayDoc *RelayDoc) error {
	_, err := r.Collection(RelayCollection).InsertOne(ctx, relayDoc)
	return err
}

// FindByID finds a relay by its ObjectID
func (r *RelayRepository) FindByID(ctx context.Context, id bson.ObjectID) (*RelayDoc, error) {
	var doc RelayDoc
	err := r.Collection(RelayCollection).FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// FindByShareID finds a relay by its share ID
func (r *RelayRepository) FindByShareID(ctx context.Context, shareID uint64) (*RelayDoc, error) {
	var doc RelayDoc
	err := r.Collection(RelayCollection).FindOne(ctx, bson.M{"share_id": shareID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpdateConsumerID updates the consumer ID for a relay
func (r *RelayRepository) UpdateConsumerID(ctx context.Context, relayID bson.ObjectID, consumerID bson.ObjectID) error {
	update := bson.M{"$set": &bson.M{"consumer_id": consumerID}}
	_, err := r.Collection(RelayCollection).UpdateByID(ctx, relayID, update)
	return err
}
