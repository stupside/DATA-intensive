package relay

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/mongo"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Consumer handles the consumer side of relay
type Consumer struct {
	cfg      *config.Config
	repos    *Repositories
	sessions *SessionStore
}

// NewConsumer creates a new consumer with repository dependencies
func NewConsumer(
	cfg *config.Config,
	repos *Repositories,
	sessions *SessionStore,
) *Consumer {
	return &Consumer{
		cfg:      cfg,
		repos:    repos,
		sessions: sessions,
	}
}

// Handle processes the consumer stream
func (c *Consumer) Handle(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
) error {
	// Authenticate
	device, err := authenticateDevice(ctx)
	if err != nil {
		return err
	}

	// Receive and parse init message
	first, err := stream.Receive()
	if err != nil {
		return errors.Internal(ctx, "Failed to receive init message", err)
	}

	init := first.GetInit()
	if init == nil {
		return errors.InvalidArgumentMsg(ctx, "First message must be ConsumeInit")
	}

	// Load share by ID and verify secret using repository
	share, err := c.repos.Share.FindByIDWithFiles(ctx, init.GetShareId())
	if err != nil {
		return errors.NotFound(ctx, "share", init.GetShareId())
	}

	if !c.repos.Share.VerifySecret(share, init.GetShareSecret()) {
		return errors.PermissionDenied(ctx, "Invalid share secret", "share_id", share.ID)
	}

	// Create consumer in MongoDB using repository
	consumer := &mongo.ConsumerDoc{
		ID:       bson.NewObjectID(),
		DeviceID: device.ID,
	}
	if err := c.repos.Consumer.Create(ctx, consumer); err != nil {
		return errors.Internal(ctx, "Failed to register consumer", err)
	}

	// Find relay by share ID using repository
	relay, err := c.repos.Relay.FindByShareID(ctx, share.ID)
	if err != nil {
		return errors.NotFound(ctx, "relay for share", share.ID)
	}

	// Update relay with consumer ID using repository
	if err := c.repos.Relay.UpdateConsumerID(ctx, relay.ID, consumer.ID); err != nil {
		return errors.Internal(ctx, "Failed to update relay with consumer", err)
	}

	// Get session
	session, ok := c.sessions.Get(relay.ID)
	if !ok {
		return errors.NotFound(ctx, "session for relay", relay.ID.Hex())
	}

	// Transfer all files
	if err := c.transferFile(ctx, stream, session, share.Files); err != nil {
		return err
	}

	// Send completion message to consumer
	if err := stream.Send(&v1.ConsumeResponse{
		Payload: &v1.ConsumeResponse_Complete{
			Complete: &v1.ConsumeComplete{},
		},
	}); err != nil {
		return errors.Internal(ctx, "Failed to send consume complete message", err)
	}

	// Close session to signal producer that we're done
	session.close()

	slog.InfoContext(ctx, "Consumer completed successfully", "relay_id", relay.ID.Hex(), "device_id", device.ID, "share_id", share.ID)
	return nil
}

// transferFile handles the transfer of all files through the relay
func (c *Consumer) transferFile(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
	session *session,
	files []*models.File,
) error {
	for _, file := range files {
		if err := c.transferSingleFile(ctx, stream, session, file); err != nil {
			return err
		}
	}
	return nil
}

// transferSingleFile handles the transfer of a single file
func (c *Consumer) transferSingleFile(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
	session *session,
	file *models.File,
) error {
	var offset uint64 = 0
	for offset < file.Size {
		// Calculate chunk size
		size := c.cfg.Features.Relay.ChunkSize
		if offset+size > file.Size {
			size = file.Size - offset
		}

		// Request chunk via session
		select {
		case session.requestChan <- &v1.RequestChunk{Path: file.Path, Offset: offset, Length: size}:
		case <-ctx.Done():
			return ctx.Err()
		case <-session.Done():
			return errors.Internal(ctx, "Session closed while requesting chunk", nil)
		}

		// Record request in MongoDB using repository
		reqID := bson.NewObjectID()
		req := &mongo.ChunkRequestDoc{
			ID:     reqID,
			Offset: offset,
			Length: size,
			FileID: file.ID,
		}
		if err := c.repos.Chunk.CreateRequest(ctx, req); err != nil {
			return errors.Internal(ctx, "Failed to store chunk request", err)
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

		// Record delivery in MongoDB using repository
		delivery := &mongo.ChunkDeliveryDoc{
			ID:             bson.NewObjectID(),
			DeliveredAt:    bson.DateTime(time.Now().UnixMilli()),
			ChunkRequestID: reqID,
		}
		if err := c.repos.Chunk.CreateDelivery(ctx, delivery); err != nil {
			return errors.Internal(ctx, "Failed to store chunk delivery", err)
		}

		offset += c.cfg.Features.Relay.ChunkSize
		if offset >= file.Size {
			slog.InfoContext(ctx, "Completed transfer of file", "path", file.Path, "size", file.Size)
		}
	}
	return nil
}
