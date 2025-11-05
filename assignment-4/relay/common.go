package relay

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
)

// authenticateDevice extracts and validates the device from context
func authenticateDevice(ctx context.Context) (*models.Device, error) {
	device, err := auth.DeviceFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated(ctx, "Authentication failed")
	}
	return device, nil
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
		return errors.Internal(ctx, "Failed to send chunk to consumer", err)
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
		return nil, errors.Internal(ctx, "Session closed while waiting for chunk", fmt.Errorf("session closed"))
	}
}
