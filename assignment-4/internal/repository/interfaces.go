package repository

import (
	"context"

	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/mongo"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Postgres repository interfaces
type DeviceRepository interface {
	Create(ctx context.Context, device *models.Device) error
	FindByID(ctx context.Context, id uint64) (*models.Device, error)
}

type CertificateRepository interface {
	Create(ctx context.Context, cert *models.Certificate) error
	FindByPublicKey(ctx context.Context, pubKey []byte) (*models.Certificate, error)
	FindByDeviceID(ctx context.Context, deviceID uint64) (*models.Certificate, error)
}

type ConnectionRepository interface {
	Create(ctx context.Context, conn *models.Connection) error
	FindByDeviceID(ctx context.Context, deviceID uint64) ([]*models.Connection, error)
}

type ShareRepository interface {
	Create(ctx context.Context, share *models.Share) error
	FindByID(ctx context.Context, id uint64) (*models.Share, error)
	FindByIDWithFiles(ctx context.Context, id uint64) (*models.Share, error)
	FindByDeviceID(ctx context.Context, deviceID uint64) ([]*models.Share, error)
	VerifySecret(share *models.Share, secret string) bool
}

// MongoDB relay repository interfaces
type RelayRepository interface {
	Create(ctx context.Context, relay *mongo.RelayDoc) error
	FindByShareID(ctx context.Context, shareID uint64) (*mongo.RelayDoc, error)
	UpdateConsumerID(ctx context.Context, relayID bson.ObjectID, consumerID bson.ObjectID) error
}

type StreamerRepository interface {
	Create(ctx context.Context, streamer *mongo.StreamerDoc) error
}

type ConsumerRepository interface {
	Create(ctx context.Context, consumer *mongo.ConsumerDoc) error
}

type ChunkRepository interface {
	CreateRequest(ctx context.Context, req *mongo.ChunkRequestDoc) error
	CreateDelivery(ctx context.Context, delivery *mongo.ChunkDeliveryDoc) error
	CountDeliveredBytesForFile(ctx context.Context, fileID uint64) (uint64, error)
}
