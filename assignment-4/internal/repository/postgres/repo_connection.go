package postgres

import (
	"context"

	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/gorm"
)

// Compile-time check to ensure PostgresConnectionRepository implements ConnectionRepository
var _ repository.ConnectionRepository = (*PostgresConnectionRepository)(nil)

// PostgresConnectionRepository handles connection-related database operations
type PostgresConnectionRepository struct {
	db *gorm.DB
}

// NewPostgresConnectionRepository creates a new connection repository
func NewPostgresConnectionRepository(db *gorm.DB) *PostgresConnectionRepository {
	return &PostgresConnectionRepository{db: db}
}

// Create records a new connection attempt
func (r *PostgresConnectionRepository) Create(ctx context.Context, conn *models.Connection) error {
	return gorm.G[models.Connection](r.db).Create(ctx, conn)
}

// FindByDeviceID retrieves all connections for a specific device ordered by creation time (newest first)
func (r *PostgresConnectionRepository) FindByDeviceID(ctx context.Context, deviceID uint64) ([]*models.Connection, error) {
	connections, err := gorm.G[models.Connection](r.db).
		Where(&models.Connection{DeviceID: deviceID}).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*models.Connection, len(connections))
	for i := range connections {
		result[i] = &connections[i]
	}
	return result, nil
}
