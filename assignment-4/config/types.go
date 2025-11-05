package config

import "time"

// Server contains network and protocol configuration
type Server struct {
	Port int  `yaml:"port" validate:"required,min=1,max=65535"`
	HTTP HTTP `yaml:"http" validate:"required"`
}

// HTTP contains HTTP protocol configuration
type HTTP struct {
	EnableHTTP1      bool `yaml:"enable_http1"`
	UnencryptedHTTP2 bool `yaml:"unencrypted_http2"`
}

// TLS contains certificate configuration for server and CA
type TLS struct {
	Server Certificate `yaml:"server" validate:"required"`
	CA     Certificate `yaml:"ca" validate:"required"`
}

// Certificate contains paths to certificate and key files
type Certificate struct {
	CertPath string `yaml:"cert_path" validate:"required"`
	KeyPath  string `yaml:"key_path" validate:"required"`
}

// Security contains access control and security policy configuration
type Security struct {
	CORS CORS `yaml:"cors" validate:"required"`
}

// CORS contains Cross-Origin Resource Sharing configuration
type CORS struct {
	AllowedOrigins []string `yaml:"allowed_origins" validate:"dive,required"`
}

// Database contains data persistence configuration
type Database struct {
	PostgreSQL PostgreSQL `yaml:"postgresql" validate:"required"`
	MongoDB    MongoDB    `yaml:"mongodb" validate:"required"`
}

// PostgreSQL contains PostgreSQL database configuration
type PostgreSQL struct {
	DSN         string `yaml:"dsn" validate:"required"`
	AutoMigrate bool   `yaml:"auto_migrate"`
}

// MongoDB contains MongoDB database configuration
type MongoDB struct {
	URI      string `yaml:"uri" validate:"required"`
	Database string `yaml:"database" validate:"required"`
}

// Features contains application feature configuration
type Features struct {
	Relay      Relay      `yaml:"relay" validate:"required"`
	Onboarding Onboarding `yaml:"onboarding" validate:"required"`
	GRPC       GRPC       `yaml:"grpc" validate:"required"`
}

// Relay contains relay streaming configuration
type Relay struct {
	ChunkSize uint64 `yaml:"chunk_size" validate:"required,min=1024"`
}

// Onboarding contains device onboarding configuration
type Onboarding struct {
	CertificateValidity time.Duration `yaml:"certificate_validity" validate:"required"`
}

// GRPC contains gRPC protocol configuration
type GRPC struct {
	EnableReflection bool `yaml:"enable_reflection"`
}

// Config is the root configuration structure
type Config struct {
	Server   Server   `yaml:"server" validate:"required"`
	TLS      TLS      `yaml:"tls" validate:"required"`
	Security Security `yaml:"security" validate:"required"`
	Database Database `yaml:"database" validate:"required"`
	Features Features `yaml:"features" validate:"required"`
}
