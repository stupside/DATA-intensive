package mongo

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoRepository provides access to a MongoDB client and a helper to get collections
type MongoRepository struct {
	client *mongo.Client
	dbName string
}

// NewMongoRepository creates a new MongoRepository
func NewMongoRepository(client *mongo.Client, dbName string) *MongoRepository {
	return &MongoRepository{client: client, dbName: dbName}
}

// Collection returns the mongo collection handle
func (r *MongoRepository) Collection(name string) *mongo.Collection {
	return r.client.Database(r.dbName).Collection(name)
}
