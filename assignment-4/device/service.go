package device

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1/v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the DeviceServiceHandler interface
type Service struct {
	connectionRepo repository.ConnectionRepository
}

var _ v1connect.DeviceServiceHandler = (*Service)(nil)

// NewService creates a new device service
func NewService(connectionRepo repository.ConnectionRepository) *Service {
	return &Service{
		connectionRepo: connectionRepo,
	}
}

// Connections returns connection history for the authenticated device
func (s *Service) Connections(ctx context.Context, req *connect.Request[v1.ConnectionsRequest]) (*connect.Response[v1.ConnectionsResponse], error) {
	// Authenticate device
	device, err := auth.DeviceFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated(ctx, "Authentication required")
	}

	// Get all connections for this device
	connections, err := s.connectionRepo.FindByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, errors.Internal(ctx, "Failed to fetch connections", err)
	}

	// Convert to proto response
	protoConnections := make([]*v1.ConnectionInfo, len(connections))
	for i, conn := range connections {
		protoConnections[i] = &v1.ConnectionInfo{
			Id:        conn.ID,
			Success:   conn.Success,
			CreatedAt: timestamppb.New(conn.CreatedAt),
			IpAddress: conn.IPAddress,
		}
	}

	slog.InfoContext(ctx, "Connections retrieved", "device_id", device.ID, "count", len(connections))

	return connect.NewResponse(&v1.ConnectionsResponse{
		Connections: protoConnections,
	}), nil
}
