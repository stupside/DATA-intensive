package mongo

import (
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoRepositories bundles Mongo-specific repository implementations
type MongoRepositories struct {
	Relay    *RelayRepository
	Streamer *StreamerRepository
	Consumer *ConsumerRepository
	Chunk    *ChunkRepository
}

// NewMongoRepositories creates and initializes Mongo repository implementations
func NewMongoRepositories(client *mongodriver.Client, dbName string) *MongoRepositories {
	return &MongoRepositories{
		Relay:    NewRelayRepository(client, dbName),
		Streamer: NewStreamerRepository(client, dbName),
		Consumer: NewConsumerRepository(client, dbName),
		Chunk:    NewChunkRepository(client, dbName),
	}
}
