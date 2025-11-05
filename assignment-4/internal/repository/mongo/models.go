package mongo

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MongoDB collection names used by relay repositories
const (
	RelayCollection         = "relays"
	StreamerCollection      = "streamers"
	ConsumerCollection      = "consumers"
	ChunkRequestCollection  = "chunk_requests"
	ChunkDeliveryCollection = "chunk_deliveries"
)

// RelayDoc coordinates data transfer between one streamer (producer) and one consumer
type RelayDoc struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	ShareID    uint64        `bson:"share_id,omitempty"`
	StreamerID bson.ObjectID `bson:"streamer_id,omitempty"`
	ConsumerID bson.ObjectID `bson:"consumer_id,omitempty"`
}

// StreamerDoc represents a producer pushing data through a relay
type StreamerDoc struct {
	ID       bson.ObjectID `bson:"_id,omitempty"`
	DeviceID uint64        `bson:"device_id,omitempty"`
}

// ConsumerDoc represents a client pulling data through a relay
type ConsumerDoc struct {
	ID       bson.ObjectID `bson:"_id,omitempty"`
	DeviceID uint64        `bson:"device_id,omitempty"`
}

// ChunkRequestDoc tracks an individual chunk request
type ChunkRequestDoc struct {
	ID     bson.ObjectID `bson:"_id,omitempty"`
	Offset uint64        `bson:"offset,omitempty"`
	Length uint64        `bson:"length,omitempty"`
	FileID uint64        `bson:"file_id,omitempty"`
}

// ChunkDeliveryDoc is an audit trail for successful chunk deliveries
type ChunkDeliveryDoc struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	DeliveredAt    bson.DateTime `bson:"delivered_at,omitempty"`
	ChunkRequestID bson.ObjectID `bson:"chunk_request_id,omitempty"`
}
