package relay

import (
	"context"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1/v1connect"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Handler implements the RelayServiceHandler interface
type Handler struct {
	cfg      *config.Config
	repos    *Repositories
	sessions *SessionStore
	producer *Producer
	consumer *Consumer
}

// Repositories bundles all repository interfaces used by relay handlers
type Repositories struct {
	Share    repository.ShareRepository
	Relay    repository.RelayRepository
	Streamer repository.StreamerRepository
	Consumer repository.ConsumerRepository
	Chunk    repository.ChunkRepository
}

var _ v1connect.RelayServiceHandler = (*Handler)(nil)

// NewHandler creates a new relay handler with repository dependencies
func NewHandler(
	cfg *config.Config,
	repos *Repositories,
	sessions *SessionStore,
) *Handler {
	return &Handler{
		cfg:      cfg,
		repos:    repos,
		sessions: sessions,
		producer: NewProducer(cfg, repos, sessions),
		consumer: NewConsumer(cfg, repos, sessions),
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
	// Validate request
	shareID := req.Msg.GetShareId()
	shareSecret := req.Msg.GetShareSecret()

	// Load share by ID and verify secret using repository
	share, err := h.repos.Share.FindByIDWithFiles(ctx, shareID)
	if err != nil {
		return nil, errors.NotFound(ctx, "share", shareID)
	}

	if !h.repos.Share.VerifySecret(share, shareSecret) {
		return nil, errors.PermissionDenied(ctx, "Invalid share secret", "share_id", share.ID)
	}

	// Find the relay in MongoDB using repository
	_, err = h.repos.Relay.FindByShareID(ctx, share.ID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No relay started yet, return all files with 0 progress
			files := make([]*v1.SummaryResponse_FileSummary, len(share.Files))
			for i, file := range share.Files {
				files[i] = &v1.SummaryResponse_FileSummary{
					Path:           file.Path,
					BytesRelayed:   0,
					BytesRemaining: file.Size,
				}
			}
			return connect.NewResponse(&v1.SummaryResponse{
				Files: files,
			}), nil
		}
		return nil, errors.Internal(ctx, "Failed to find relay", err)
	}

	// Build file summaries using chunk repository
	files := make([]*v1.SummaryResponse_FileSummary, len(share.Files))
	for i, file := range share.Files {
		// Count delivered bytes for this file using repository
		bytesRelayed, err := h.repos.Chunk.CountDeliveredBytesForFile(ctx, file.ID)
		if err != nil {
			return nil, errors.Internal(ctx, "Failed to count delivered bytes", err)
		}

		bytesRemaining := uint64(0)
		if file.Size > bytesRelayed {
			bytesRemaining = file.Size - bytesRelayed
		}

		files[i] = &v1.SummaryResponse_FileSummary{
			Path:           file.Path,
			BytesRelayed:   bytesRelayed,
			BytesRemaining: bytesRemaining,
		}
	}

	return connect.NewResponse(&v1.SummaryResponse{
		Files: files,
	}), nil
}
