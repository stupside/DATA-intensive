package relay

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

// Consumer handles the consumer side of relay
type Consumer struct {
	cfg      *config.Config
	postgres *gorm.DB
	mongo    *mongo.Client
	sessions *SessionStore
}

// NewConsumer creates a new consumer
func NewConsumer(cfg *config.Config, postgres *gorm.DB, mongo *mongo.Client, sessions *SessionStore) *Consumer {
	return &Consumer{
		cfg:      cfg,
		postgres: postgres,
		mongo:    mongo,
		sessions: sessions,
	}
}

// Handle processes the consumer stream
func (c *Consumer) Handle(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
) error {
	// Authenticate
	user, err := authenticateUser(ctx)
	if err != nil {
		return err
	}

	// Receive and parse init message
	first, err := stream.Receive()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to receive init message", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to receive init message: %w", err))
	}

	init := first.GetInit()
	if init == nil {
		slog.ErrorContext(ctx, "Invalid init message")
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("first message must be ConsumeInit"))
	}

	// Load share from PostgreSQL
	share, err := getShareByKey(ctx, c.postgres, init.GetShareKey())
	if err != nil {
		return err
	}

	// Create consumer in MongoDB
	consumer := &consumerDoc{
		ID:     bson.NewObjectID(),
		UserID: user.ID,
	}
	consumerColl := c.mongo.Database(c.cfg.Database.MongoDB.Database).Collection(ConsumerCollection)
	if _, err := consumerColl.InsertOne(ctx, consumer); err != nil {
		slog.ErrorContext(ctx, "Failed to register consumer", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register consumer"))
	}

	// Find and update relay by share ID in MongoDB
	var relay relayDoc
	relayColl := c.mongo.Database(c.cfg.Database.MongoDB.Database).Collection(RelayCollection)
	if err := relayColl.FindOne(ctx, relayDoc{ShareID: share.ID}).Decode(&relay); err != nil {
		slog.ErrorContext(ctx, "No relay found for this share", "error", err)
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("no relay found for this share"))
	}

	if _, err := relayColl.UpdateByID(ctx, relay.ID, bson.M{"$set": &relayDoc{ConsumerID: consumer.ID}}); err != nil {
		slog.ErrorContext(ctx, "Failed to update relay with consumer", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update relay with consumer"))
	}

	// Get session
	session, ok := c.sessions.Get(relay.ID)
	if !ok {
		slog.ErrorContext(ctx, "Session not found", "relay_id", relay.ID)
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found"))
	}

	// MongoDB collections for audit
	chunkReqColl := c.mongo.Database(c.cfg.Database.MongoDB.Database).Collection(ChunkRequestCollection)
	chunkDelColl := c.mongo.Database(c.cfg.Database.MongoDB.Database).Collection(ChunkDeliveryCollection)

	// Transfer all content
	if err := c.transferContent(ctx, stream, session, share.Contents, chunkReqColl, chunkDelColl); err != nil {
		return err
	}

	// Send completion message to consumer
	if err := stream.Send(&v1.ConsumeResponse{
		Payload: &v1.ConsumeResponse_Complete{
			Complete: &v1.ConsumeComplete{},
		},
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to send consume complete message", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send consume complete message"))
	}

	// Close session to signal producer that we're done
	session.close()

	slog.InfoContext(ctx, "Consumer completed successfully", "relay_id", relay.ID)
	return nil
}

// transferContent handles the transfer of all content through the relay
func (c *Consumer) transferContent(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
	session *session,
	contents []*models.Content,
	chunkReqColl *mongo.Collection,
	chunkDelColl *mongo.Collection,
) error {
	for _, content := range contents {
		if err := c.transferSingleContent(ctx, stream, session, content, chunkReqColl, chunkDelColl); err != nil {
			return err
		}
	}
	return nil
}

// transferSingleContent handles the transfer of a single content file
func (c *Consumer) transferSingleContent(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
	session *session,
	content *models.Content,
	chunkReqColl *mongo.Collection,
	chunkDelColl *mongo.Collection,
) error {
	var offset uint64 = 0
	for offset < content.Size {
		// Calculate chunk size
		size := c.cfg.Features.Relay.ChunkSize
		if offset+size > content.Size {
			size = content.Size - offset
		}

		// Request chunk via session
		select {
		case session.requestChan <- &v1.RequestChunk{Path: content.Path, Offset: offset, Length: size}:
		case <-ctx.Done():
			return ctx.Err()
		case <-session.Done():
			return fmt.Errorf("session closed while requesting chunk")
		}

		// Record request in MongoDB
		reqID := bson.NewObjectID()
		req := &chunkRequestDoc{
			ID:        reqID,
			Offset:    offset,
			Length:    size,
			ContentID: content.ID,
		}
		if _, err := chunkReqColl.InsertOne(ctx, req); err != nil {
			slog.ErrorContext(ctx, "Failed to store chunk request", "error", err)
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to store chunk request"))
		}

		// Receive chunk from session
		chunk, err := waitForChunk(ctx, session)
		if err != nil {
			return err
		}

		// Send chunk to consumer
		if err := sendChunkToConsumer(ctx, stream, chunk); err != nil {
			return err
		}

		// Record delivery in MongoDB
		delivery := &chunkDeliveryDoc{
			ID:             bson.NewObjectID(),
			DeliveredAt:    time.Now().Unix(),
			ChunkRequestID: reqID,
		}
		if _, err := chunkDelColl.InsertOne(ctx, delivery); err != nil {
			slog.ErrorContext(ctx, "Failed to store chunk delivery", "error", err)
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to store chunk delivery"))
		}

		offset += c.cfg.Features.Relay.ChunkSize
		if offset >= content.Size {
			slog.InfoContext(ctx, "Completed transfer of content", "path", content.Path, "size", content.Size)
		}
	}
	return nil
}
