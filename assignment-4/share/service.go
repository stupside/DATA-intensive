package share

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1/v1connect"
	"gorm.io/gorm"
)

// Service handles share management business logic and implements the gRPC handler interface
type Service struct {
	db *gorm.DB
}

var _ v1connect.ShareServiceHandler = (*Service)(nil)

// NewService creates a new share service
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Create handles creation of new shares
func (s *Service) Create(ctx context.Context, req *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	// Extract authenticated user
	user, err := auth.UserFromContext(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Could not extract user from context", "error", err)
		return nil, err
	}

	id, key, err := createShare(ctx, s.db, user.ID, req.Msg.GetContents())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.CreateResponse{
		Id:  id,
		Key: key,
	}), nil
}

// Detail retrieves share details with access control
func (s *Service) Detail(ctx context.Context, req *connect.Request[v1.DetailRequest]) (*connect.Response[v1.DetailResponse], error) {

	contents, err := getShareContents(ctx, s.db, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	// Build response
	responseContents := make([]*v1.DetailResponse_Content, len(contents))
	for i, content := range contents {
		responseContents[i] = &v1.DetailResponse_Content{
			Size: content.Size,
			Path: content.Path,
		}
	}

	return connect.NewResponse(&v1.DetailResponse{
		Contents: responseContents,
	}), nil
}

// createShare creates a new file share
func createShare(ctx context.Context, db *gorm.DB, userID uint, contents []*v1.CreateRequest_Content) (string, string, error) {
	// Create share
	share := &models.Share{
		Key:      uuid.NewString(),
		UserID:   userID,
		Contents: prepareContents(contents),
	}
	if err := gorm.G[models.Share](db).Create(ctx, share); err != nil {
		slog.ErrorContext(ctx, "Failed to create share in database", "error", err)
		return "", "", connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create share: %w", err))
	}

	return fmt.Sprint(share.ID), share.Key, nil
}

// getShareContents retrieves share details with access control
func getShareContents(ctx context.Context, db *gorm.DB, shareID uint64) ([]*models.Content, error) {

	share, err := gorm.G[models.Share](db).Where(&models.Share{Model: gorm.Model{ID: uint(shareID)}}).First(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get share from database", "id", shareID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get share: %w", err))
	}

	return share.Contents, nil
}

// prepareContents converts protobuf contents to domain models
func prepareContents(contents []*v1.CreateRequest_Content) []*models.Content {
	result := make([]*models.Content, len(contents))
	for i, content := range contents {
		result[i] = &models.Content{
			Size: content.GetSize(),
			Path: content.GetPath(),
		}
	}
	return result
}
