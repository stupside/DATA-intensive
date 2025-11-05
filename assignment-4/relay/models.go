package relay

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MongoDB collection names
const (
	RelayCollection         = "relays"
	StreamerCollection      = "streamers"
	ConsumerCollection      = "consumers"
	ChunkRequestCollection  = "chunk_requests"
	ChunkDeliveryCollection = "chunk_deliveries"
)

// relayDoc coordinates data transfer between one streamer (producer) and one consumer
type relayDoc struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// ShareID references the SQL share this relay belongs to
	ShareID uint `bson:"share_id,omitempty"`

	// StreamerID references the streamer using this relay
	StreamerID bson.ObjectID `bson:"streamer_id,omitempty"`
	// ConsumerID references the single consumer using this relay
	ConsumerID bson.ObjectID `bson:"consumer_id,omitempty"`
}

// streamerDoc represents a producer pushing data through a relay
type streamerDoc struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// UserID references the user who owns this streamer (SQL ID)
	UserID uint `bson:"user_id,omitempty"`
}

// consumerDoc represents a client pulling data through a relay
type consumerDoc struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// UserID references the user who is downloading (SQL ID)
	UserID uint `bson:"user_id,omitempty"`
}

// chunkRequestDoc tracks an individual chunk request
type chunkRequestDoc struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// Offset is the byte offset in the file
	Offset uint64 `bson:"offset,omitempty"`
	// Length is the number of bytes requested
	Length uint64 `bson:"length,omitempty"`

	// ContentID references the content being requested
	ContentID uint `bson:"content_id,omitempty"`
}

// chunkDeliveryDoc is an audit trail for successful chunk deliveries
type chunkDeliveryDoc struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// DeliveredAt tracks when the chunk was delivered
	DeliveredAt int64 `bson:"delivered_at,omitempty"` // Unix timestamp

	// ChunkRequestID references the chunk request that was delivered
	ChunkRequestID bson.ObjectID `bson:"chunk_request_id,omitempty"`
}
