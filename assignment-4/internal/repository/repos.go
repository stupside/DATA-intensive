package repository

// CompositeRepositories is a convenience struct used at application wiring time
// to bundle all database-backed repository implementations (Postgres and Mongo)
// behind the interfaces used by the application.
type CompositeRepositories struct {
	// Postgres-backed repositories
	Device      DeviceRepository
	Share       ShareRepository
	Certificate CertificateRepository
	Connection  ConnectionRepository

	// Mongo-backed relay repositories
	Relay    RelayRepository
	Streamer StreamerRepository
	Consumer ConsumerRepository
	Chunk    ChunkRepository
}
