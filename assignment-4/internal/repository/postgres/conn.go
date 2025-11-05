package postgres

import (
	"fmt"

	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres creates and returns a configured PostgreSQL connection
func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.PostgreSQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQL database: %w", err)
	}

	if cfg.Database.PostgreSQL.AutoMigrate {
		if err := MigratePostgres(db); err != nil {
			return nil, fmt.Errorf("failed to migrate SQL database: %w", err)
		}
	}

	return db, nil
}

// MigratePostgres runs auto-migration for all models
func MigratePostgres(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Device{},
		&models.Share{},
		&models.File{},
		&models.Certificate{},
		&models.Connection{},
	)
}

// ClosePostgres closes the PostgreSQL connection
func ClosePostgres(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close SQL database: %w", err)
	}

	return nil
}
