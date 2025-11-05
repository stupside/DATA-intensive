package postgres

import (
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"gorm.io/gorm"
)

// PostgresRepositories bundles Postgres-backed repository implementations
type PostgresRepositories struct {
	Device      repository.DeviceRepository
	Share       repository.ShareRepository
	Certificate repository.CertificateRepository
	Connection  repository.ConnectionRepository
}

// NewPostgresRepositories creates Postgres repository implementations
func NewPostgresRepositories(db *gorm.DB) *PostgresRepositories {
	return &PostgresRepositories{
		Device:      NewPostgresDeviceRepository(db),
		Share:       NewPostgresShareRepository(db),
		Certificate: NewPostgresCertificateRepository(db),
		Connection:  NewPostgresConnectionRepository(db),
	}
}
