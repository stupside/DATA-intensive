package postgres

import (
	"context"

	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/gorm"
)

// Compile-time check to ensure PostgresCertificateRepository implements CertificateRepository
var _ repository.CertificateRepository = (*PostgresCertificateRepository)(nil)

// PostgresCertificateRepository handles certificate-related database operations
type PostgresCertificateRepository struct {
	db *gorm.DB
}

// NewPostgresCertificateRepository creates a new certificate repository
func NewPostgresCertificateRepository(db *gorm.DB) *PostgresCertificateRepository {
	return &PostgresCertificateRepository{db: db}
}

// Create creates a new certificate
func (r *PostgresCertificateRepository) Create(ctx context.Context, cert *models.Certificate) error {
	return gorm.G[models.Certificate](r.db).Create(ctx, cert)
}

// FindByPublicKey finds a certificate by its public key
func (r *PostgresCertificateRepository) FindByPublicKey(ctx context.Context, pubKey []byte) (*models.Certificate, error) {
	cert, err := gorm.G[models.Certificate](r.db).Where(&models.Certificate{PublicKey: pubKey}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// FindByDeviceID finds a certificate by device ID
func (r *PostgresCertificateRepository) FindByDeviceID(ctx context.Context, deviceID uint64) (*models.Certificate, error) {
	cert, err := gorm.G[models.Certificate](r.db).Where(&models.Certificate{DeviceID: deviceID}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}
