package relay

import (
	"context"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1/v1connect"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

// Handler implements the RelayServiceHandler interface
type Handler struct {
	cfg      *config.Config
	postgres *gorm.DB
	mongo    *mongo.Client
	sessions *SessionStore
	producer *Producer
	consumer *Consumer
}

var _ v1connect.RelayServiceHandler = (*Handler)(nil)

// NewHandler creates a new relay handler
func NewHandler(cfg *config.Config, postgres *gorm.DB, mongo *mongo.Client, sessions *SessionStore) *Handler {
	return &Handler{
		cfg:      cfg,
		postgres: postgres,
		mongo:    mongo,
		sessions: sessions,
		producer: NewProducer(cfg, postgres, mongo, sessions),
		consumer: NewConsumer(cfg, postgres, mongo, sessions),
	}
}

// Stream handles the producer (streaming) side
func (h *Handler) Stream(
	ctx context.Context,
	stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse],
) error {
	return h.producer.Handle(ctx, stream)
}

// Consume handles the consumer (downloading) side
func (h *Handler) Consume(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
) error {
	return h.consumer.Handle(ctx, stream)
}

// Summary returns relay statistics for a share
func (h *Handler) Summary(ctx context.Context, req *connect.Request[v1.SummaryRequest]) (*connect.Response[v1.SummaryResponse], error) {
	shareKey := req.Msg.GetShareKey()
	if shareKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	// Load share from PostgreSQL using existing helper
	share, err := getShareByKey(ctx, h.postgres, shareKey)
	if err != nil {
		return nil, err
	}

	// Find the relay in MongoDB
	relayColl := h.mongo.Database(h.cfg.Database.MongoDB.Database).Collection(RelayCollection)

	var relay relayDoc
	if err := relayColl.FindOne(ctx, relayDoc{ShareID: share.ID}).Decode(&relay); err != nil {
		if err == mongo.ErrNoDocuments {
			// No relay started yet, return all files with 0 progress
			files := make([]*v1.SummaryResponse_FileSummary, len(share.Contents))
			for i, content := range share.Contents {
				files[i] = &v1.SummaryResponse_FileSummary{
					Path:           content.Path,
					BytesRelayed:   0,
					BytesRemaining: content.Size,
				}
			}
			return connect.NewResponse(&v1.SummaryResponse{
				Files: files,
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Query chunk deliveries to calculate bytes relayed per file
	chunkDeliveryColl := h.mongo.Database(h.cfg.Database.MongoDB.Database).Collection(ChunkDeliveryCollection)

	// Build file summaries
	files := make([]*v1.SummaryResponse_FileSummary, len(share.Contents))
	for i, content := range share.Contents {
		// Count delivered bytes for this content using aggregation pipeline
		pipeline := []interface{}{
			map[string]interface{}{
				"$lookup": map[string]interface{}{
					"from":         ChunkRequestCollection,
					"localField":   "chunk_request_id",
					"foreignField": "_id",
					"as":           "request",
				},
			},
			map[string]interface{}{
				"$unwind": "$request",
			},
			map[string]interface{}{
				"$match": map[string]interface{}{
					"request.content_id": content.ID,
				},
			},
			map[string]interface{}{
				"$group": map[string]interface{}{
					"_id":   nil,
					"total": map[string]interface{}{"$sum": "$request.length"},
				},
			},
		}

		cursor, err := chunkDeliveryColl.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var result []struct {
			Total uint64 `bson:"total"`
		}
		if err := cursor.All(ctx, &result); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var bytesRelayed uint64
		if len(result) > 0 {
			bytesRelayed = result[0].Total
		}

		bytesRemaining := uint64(0)
		if content.Size > bytesRelayed {
			bytesRemaining = content.Size - bytesRelayed
		}

		files[i] = &v1.SummaryResponse_FileSummary{
			Path:           content.Path,
			BytesRelayed:   bytesRelayed,
			BytesRemaining: bytesRemaining,
		}
	}

	return connect.NewResponse(&v1.SummaryResponse{
		Files: files,
	}), nil
}
