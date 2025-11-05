package postgres

import (
	"context"

	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/gorm"
)

// Compile-time check to ensure PostgresDeviceRepository implements DeviceRepository
var _ repository.DeviceRepository = (*PostgresDeviceRepository)(nil)

// PostgresDeviceRepository handles device-related database operations
type PostgresDeviceRepository struct {
	db *gorm.DB
}

// NewPostgresDeviceRepository creates a new device repository
func NewPostgresDeviceRepository(db *gorm.DB) *PostgresDeviceRepository {
	return &PostgresDeviceRepository{db: db}
}

// Create creates a new device with its certificate
func (r *PostgresDeviceRepository) Create(ctx context.Context, device *models.Device) error {
	return gorm.G[models.Device](r.db).Create(ctx, device)
}

// FindByID finds a device by its ID
func (r *PostgresDeviceRepository) FindByID(ctx context.Context, id uint64) (*models.Device, error) {
	device, err := gorm.G[models.Device](r.db).Where(&models.Device{Model: gorm.Model{ID: id}}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &device, nil
}
