package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ChunkRepository handles chunk-related MongoDB operations
type ChunkRepository struct {
	*MongoRepository
}

// NewChunkRepository creates a new chunk repository
func NewChunkRepository(client *mongo.Client, dbName string) *ChunkRepository {
	return &ChunkRepository{MongoRepository: NewMongoRepository(client, dbName)}
}

// CreateRequest creates a new chunk request document
func (r *ChunkRepository) CreateRequest(ctx context.Context, req *ChunkRequestDoc) error {
	_, err := r.Collection(ChunkRequestCollection).InsertOne(ctx, req)
	return err
}

// CreateDelivery creates a new chunk delivery document (audit trail)
func (r *ChunkRepository) CreateDelivery(ctx context.Context, delivery *ChunkDeliveryDoc) error {
	_, err := r.Collection(ChunkDeliveryCollection).InsertOne(ctx, delivery)
	return err
}

// CountDeliveredBytesForFile counts the total bytes delivered for a specific file.
func (r *ChunkRepository) CountDeliveredBytesForFile(ctx context.Context, fileID uint64) (uint64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: ChunkRequestCollection},
			{Key: "localField", Value: "chunk_request_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "request"},
		}}},
		{{Key: "$unwind", Value: "$request"}},
		{{Key: "$match", Value: bson.D{{Key: "request.file_id", Value: fileID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$request.length"}}},
		}}},
	}

	cursor, err := r.Collection(ChunkDeliveryCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total uint64 `bson:"total"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.Total, nil
	}
	return 0, nil
}
