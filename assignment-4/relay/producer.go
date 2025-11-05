package relay

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/mongo"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Producer handles the producer (streamer) side of relay
type Producer struct {
	cfg      *config.Config
	repos    *Repositories
	sessions *SessionStore
}

// NewProducer creates a new producer with repository dependencies
func NewProducer(
	cfg *config.Config,
	repos *Repositories,
	sessions *SessionStore,
) *Producer {
	return &Producer{
		cfg:      cfg,
		repos:    repos,
		sessions: sessions,
	}
}

// Handle processes the producer stream
func (p *Producer) Handle(
	ctx context.Context,
	stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse],
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
		return errors.InvalidArgumentMsg(ctx, "First message must be StreamInit")
	}

	// Load share by ID and verify secret using repository
	share, err := p.repos.Share.FindByIDWithFiles(ctx, init.GetShareId())
	if err != nil {
		return errors.NotFound(ctx, "share", init.GetShareId())
	}

	if !p.repos.Share.VerifySecret(share, init.GetShareSecret()) {
		return errors.PermissionDenied(ctx, "Invalid share secret", "share_id", share.ID)
	}

	// Verify ownership
	if share.DeviceID != device.ID {
		return errors.PermissionDenied(ctx, "Not authorized for this share", "device_id", device.ID, "share_id", share.ID)
	}

	// Create streamer in MongoDB using repository
	streamer := &mongo.StreamerDoc{
		ID:       bson.NewObjectID(),
		DeviceID: device.ID,
	}
	if err := p.repos.Streamer.Create(ctx, streamer); err != nil {
		return errors.Internal(ctx, "Failed to register streamer", err)
	}

	// Create relay in MongoDB using repository
	relay := &mongo.RelayDoc{
		ID:         bson.NewObjectID(),
		ShareID:    share.ID,
		StreamerID: streamer.ID,
	}
	if err := p.repos.Relay.Create(ctx, relay); err != nil {
		return errors.Internal(ctx, "Failed to create relay", err)
	}

	// Create session (now persists to MongoDB)
	session, err := p.sessions.Create(ctx, relay.ID, 1)
	if err != nil {
		return errors.Internal(ctx, "Failed to create session", err)
	}
	defer p.sessions.Remove(ctx, relay.ID)

	slog.InfoContext(ctx, "Producer ready, waiting for consumer", "relay_id", relay.ID.Hex(), "share_id", share.ID, "device_id", device.ID)

	// Coordinate streaming
	return p.coordinateStreaming(ctx, stream, session)
}

// handleChunkRequest processes a single chunk request from the consumer
func (p *Producer) handleChunkRequest(
	ctx context.Context,
	stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse],
	session *session,
	req *v1.RequestChunk,
) error {
	// Send request to producer
	if err := stream.Send(&v1.StreamResponse{
		Payload: &v1.StreamResponse_Request{
			Request: &v1.RequestChunk{
				Path:   req.Path,
				Offset: req.Offset,
				Length: req.Length,
			},
		},
	}); err != nil {
		return errors.Internal(ctx, "Failed to send chunk request to producer", err)
	}

	// Receive chunk from producer
	msg, err := stream.Receive()
	if err != nil {
		// Stream ended by client is expected
		slog.InfoContext(ctx, "Producer stream ended", "error", err)
		session.close()
		return nil
	}

	chunk := msg.GetChunk()
	if chunk == nil {
		slog.WarnContext(ctx, "Received non-chunk message from producer")
		return errors.InvalidArgumentMsg(ctx, "invalid chunk message")
	}

	// Forward chunk to consumer via session
	select {
	case session.consumeChan <- &v1.ConsumeChunk{
		Data:   chunk.GetData(),
		Path:   chunk.GetPath(),
		Offset: chunk.GetOffset(),
		Length: chunk.GetLength(),
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-session.Done():
		return nil
	}
}

// coordinateStreaming manages the main producer event loop
func (p *Producer) coordinateStreaming(
	ctx context.Context,
	stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse],
	session *session,
) error {
	for {
		select {
		case req := <-session.requestChan:
			if err := p.handleChunkRequest(ctx, stream, session, req); err != nil {
				return err
			}

		case <-ctx.Done():
			slog.InfoContext(ctx, "Stream context done")
			return ctx.Err()

		case <-session.Done():
			slog.InfoContext(ctx, "Stream session completed")
			// Session closed by consumer side, producer exits gracefully
			return nil
		}
	}
}
