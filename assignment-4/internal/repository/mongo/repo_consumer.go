package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ConsumerRepository handles consumer-related MongoDB operations
type ConsumerRepository struct {
	*MongoRepository
}

// NewConsumerRepository creates a new consumer repository
func NewConsumerRepository(client *mongo.Client, dbName string) *ConsumerRepository {
	return &ConsumerRepository{MongoRepository: NewMongoRepository(client, dbName)}
}

// Create creates a new consumer document
func (r *ConsumerRepository) Create(ctx context.Context, consumer *ConsumerDoc) error {
	_, err := r.Collection(ConsumerCollection).InsertOne(ctx, consumer)
	return err
}

// FindByID finds a consumer by its ObjectID
func (r *ConsumerRepository) FindByID(ctx context.Context, id bson.ObjectID) (*ConsumerDoc, error) {
	var doc ConsumerDoc
	err := r.Collection(ConsumerCollection).FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// FindAll finds all consumers
func (r *ConsumerRepository) FindAll(ctx context.Context) ([]ConsumerDoc, error) {
	cursor, err := r.Collection(ConsumerCollection).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var consumers []ConsumerDoc
	if err := cursor.All(ctx, &consumers); err != nil {
		return nil, err
	}
	return consumers, nil
}
