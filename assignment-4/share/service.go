package share

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1/v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service handles share management business logic and implements the gRPC handler interface
type Service struct {
	shareRepo repository.ShareRepository
}

var _ v1connect.ShareServiceHandler = (*Service)(nil)

// NewService creates a new share service that accepts a repository interface
func NewService(shareRepo repository.ShareRepository) *Service {
	return &Service{shareRepo: shareRepo}
}

// List returns all shares created by the authenticated device
func (s *Service) List(ctx context.Context, req *connect.Request[v1.ListRequest]) (*connect.Response[v1.ListResponse], error) {
	// Extract authenticated device
	device, err := auth.DeviceFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated(ctx, "Authentication required")
	}

	shares, err := s.shareRepo.FindByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, errors.Internal(ctx, "Failed to query shares", err)
	}

	// Convert to proto response
	respShares := make([]*v1.ListResponse_ShareInfo, len(shares))
	for i, share := range shares {
		respShares[i] = &v1.ListResponse_ShareInfo{
			Id:          share.ID,
			FileCount:   uint32(len(share.Files)),
			CreatedAt:   timestamppb.New(share.CreatedAt),
			ShareSecret: share.Secret,
		}
	}

	slog.InfoContext(ctx, "Shares listed for device", "device_id", device.ID, "count", len(shares))

	return connect.NewResponse(&v1.ListResponse{
		Shares: respShares,
	}), nil
}

// Create handles creation of new shares
func (s *Service) Create(ctx context.Context, req *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	// Extract authenticated device
	device, err := auth.DeviceFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated(ctx, "Authentication required")
	}

	// Create share
	share := &models.Share{
		Files:    prepareFiles(req.Msg.GetFiles()),
		Secret:   uuid.NewString(),
		DeviceID: device.ID,
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, errors.Internal(ctx, "Failed to create share", err)
	}

	slog.InfoContext(ctx, "Share created successfully", "share_id", share.ID, "device_id", device.ID, "file_count", len(share.Files))

	return connect.NewResponse(&v1.CreateResponse{
		Id:          share.ID,
		ShareSecret: share.Secret,
	}), nil
}

// Detail retrieves share details
func (s *Service) Detail(ctx context.Context, req *connect.Request[v1.DetailRequest]) (*connect.Response[v1.DetailResponse], error) {
	share, err := s.shareRepo.FindByIDWithFiles(ctx, req.Msg.GetId())
	if err != nil {
		return nil, errors.NotFound(ctx, "share", req.Msg.GetId())
	}

	// Build response
	respFiles := make([]*v1.DetailResponse_File, len(share.Files))
	for i, file := range share.Files {
		respFiles[i] = &v1.DetailResponse_File{
			Size: file.Size,
			Path: file.Path,
		}
	}

	slog.InfoContext(ctx, "Share details retrieved", "share_id", share.ID, "file_count", len(share.Files))

	return connect.NewResponse(&v1.DetailResponse{
		Files: respFiles,
	}), nil
}

// prepareFiles converts protobuf files to domain models
func prepareFiles(files []*v1.CreateRequest_File) []*models.File {
	result := make([]*models.File, len(files))
	for i, file := range files {
		result[i] = &models.File{
			Size: file.GetSize(),
			Path: file.GetPath(),
		}
	}
	return result
}
