package relay

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

// Producer handles the producer (streamer) side of relay
type Producer struct {
	cfg      *config.Config
	postgres *gorm.DB
	mongo    *mongo.Client
	sessions *SessionStore
}

// NewProducer creates a new producer
func NewProducer(cfg *config.Config, postgres *gorm.DB, mongo *mongo.Client, sessions *SessionStore) *Producer {
	return &Producer{
		cfg:      cfg,
		postgres: postgres,
		mongo:    mongo,
		sessions: sessions,
	}
}

// Handle processes the producer stream
func (p *Producer) Handle(
	ctx context.Context,
	stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse],
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
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("first message must be StreamInit"))
	}

	// Load share from PostgreSQL
	share, err := getShareByKey(ctx, p.postgres, init.GetShareKey())
	if err != nil {
		return err
	}

	// Verify ownership
	if share.UserID != user.ID {
		slog.ErrorContext(ctx, "User not authorized for this share", "user_id", user.ID, "share_id", share.ID)
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("user not authorized for this share"))
	}

	// Create streamer in MongoDB
	streamer := &streamerDoc{
		ID:     bson.NewObjectID(),
		UserID: user.ID,
	}
	streamerColl := p.mongo.Database(p.cfg.Database.MongoDB.Database).Collection(StreamerCollection)
	if _, err := streamerColl.InsertOne(ctx, streamer); err != nil {
		slog.ErrorContext(ctx, "Failed to register streamer", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register streamer"))
	}

	// Create relay in MongoDB
	relay := &relayDoc{
		ID:         bson.NewObjectID(),
		ShareID:    share.ID,
		StreamerID: streamer.ID,
	}
	relayColl := p.mongo.Database(p.cfg.Database.MongoDB.Database).Collection(RelayCollection)
	if _, err := relayColl.InsertOne(ctx, relay); err != nil {
		slog.ErrorContext(ctx, "Failed to create relay", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create relay"))
	}

	// Create session
	session, err := p.sessions.Create(ctx, relay.ID, 1)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create session", "error", err)
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	defer p.sessions.Remove(relay.ID)

	slog.InfoContext(ctx, "Waiting for consumer to connect", "relay_id", relay.ID)

	// Coordinate streaming
	return p.coordinateStreaming(ctx, stream, session)
}

// handleChunkRequest processes a single chunk request
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
		slog.ErrorContext(ctx, "Failed to send chunk request to producer", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send chunk request to producer"))
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
		return fmt.Errorf("invalid chunk message")
	}

	// Forward to consumer via session
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
			// Session closed by consumer side, producer should exit gracefully
			return nil
		}
	}
}
