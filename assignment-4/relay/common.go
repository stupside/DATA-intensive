package relay

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"gorm.io/gorm"
)

// authenticateUser extracts and validates the user from context
func authenticateUser(ctx context.Context) (*models.User, error) {
	user, err := auth.UserFromContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Authentication failed", "error", err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication failed: %w", err))
	}
	return user, nil
}

// getShareByKey loads a share from the database by its key
func getShareByKey(ctx context.Context, db *gorm.DB, key string) (*models.Share, error) {
	var share models.Share
	if err := db.Where(&models.Share{Key: key}).First(&share).Error; err != nil {
		slog.ErrorContext(ctx, "Share not found", "key", key, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("share not found"))
	}

	if err := db.Model(&share).Association("Contents").Find(&share.Contents); err != nil {
		slog.ErrorContext(ctx, "Failed to load share contents", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load share contents"))
	}

	return &share, nil
}

// sendChunkToConsumer sends a chunk through the bidirectional stream
func sendChunkToConsumer(
	ctx context.Context,
	stream *connect.BidiStream[v1.ConsumeRequest, v1.ConsumeResponse],
	chunk *v1.ConsumeChunk,
) error {
	if err := stream.Send(&v1.ConsumeResponse{
		Payload: &v1.ConsumeResponse_Chunk{Chunk: chunk},
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to send chunk to consumer", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send chunk to consumer"))
	}
	return nil
}

// waitForChunk waits for a chunk from the session or context cancellation
func waitForChunk(
	ctx context.Context,
	session *session,
) (*v1.ConsumeChunk, error) {
	select {
	case chunk := <-session.consumeChan:
		return chunk, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-session.Done():
		return nil, fmt.Errorf("session closed while waiting for chunk")
	}
}
