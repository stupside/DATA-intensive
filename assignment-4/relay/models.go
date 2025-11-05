package relay

import (
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/mongo"
)

// Aliases to database package types and constants
type RelayDoc = mongo.RelayDoc
type StreamerDoc = mongo.StreamerDoc
type ConsumerDoc = mongo.ConsumerDoc
type ChunkRequestDoc = mongo.ChunkRequestDoc
type ChunkDeliveryDoc = mongo.ChunkDeliveryDoc

const (
	RelayCollection         = mongo.RelayCollection
	StreamerCollection      = mongo.StreamerCollection
	ConsumerCollection      = mongo.ConsumerCollection
	ChunkRequestCollection  = mongo.ChunkRequestCollection
	ChunkDeliveryCollection = mongo.ChunkDeliveryCollection
)
