package postgres

import (
	"context"

	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/gorm"
)

// Compile-time check to ensure PostgresShareRepository implements ShareRepository
var _ repository.ShareRepository = (*PostgresShareRepository)(nil)

// PostgresShareRepository handles share-related database operations
type PostgresShareRepository struct {
	db *gorm.DB
}

// NewPostgresShareRepository creates a new share repository
func NewPostgresShareRepository(db *gorm.DB) *PostgresShareRepository {
	return &PostgresShareRepository{db: db}
}

// Create creates a new share with its files
func (r *PostgresShareRepository) Create(ctx context.Context, share *models.Share) error {
	return gorm.G[models.Share](r.db).Create(ctx, share)
}

// FindByID finds a share by its ID without loading associated files
func (r *PostgresShareRepository) FindByID(ctx context.Context, id uint64) (*models.Share, error) {
	share, err := gorm.G[models.Share](r.db).Where(&models.Share{Model: gorm.Model{ID: id}}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// FindByIDWithFiles finds a share by its ID and preloads all associated files
func (r *PostgresShareRepository) FindByIDWithFiles(ctx context.Context, id uint64) (*models.Share, error) {
	share, err := gorm.G[models.Share](r.db).Where(&models.Share{Model: gorm.Model{ID: id}}).First(ctx)
	if err != nil {
		return nil, err
	}
	sharePtr := &share
	err = r.db.WithContext(ctx).Model(sharePtr).Association("Files").Find(&sharePtr.Files)
	if err != nil {
		return nil, err
	}
	return sharePtr, nil
}

// FindByDeviceID finds all shares created by a specific device and loads their files
func (r *PostgresShareRepository) FindByDeviceID(ctx context.Context, deviceID uint64) ([]*models.Share, error) {
	shares, err := gorm.G[models.Share](r.db).Where(&models.Share{DeviceID: deviceID}).Find(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.Share, len(shares))
	for i, share := range shares {
		sharePtr := &share
		err = r.db.WithContext(ctx).Model(sharePtr).Association("Files").Find(&sharePtr.Files)
		if err != nil {
			return nil, err
		}
		result[i] = sharePtr
	}
	return result, nil
}

// VerifySecret checks if the provided secret matches the share's secret
func (r *PostgresShareRepository) VerifySecret(share *models.Share, secret string) bool {
	return share.Secret == secret
}
