package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config represents the application configuration
type Config struct {
	Databases  []DatabaseConfig `koanf:"databases" validate:"dive,required"`
	Generation GenerationConfig `koanf:"generation" validate:"required"`
}

func (cfg *Config) FindDatabaseByName(name string) (*DatabaseConfig, error) {
	for _, dbConfig := range cfg.Databases {
		if dbConfig.Name == name {
			return &dbConfig, nil
		}
	}
	return nil, fmt.Errorf("database configuration with name %q not found", name)
}

// GenerationConfig holds data generation parameters
type GenerationConfig struct {
	OutputDir              string  `koanf:"output_dir" validate:"required,dir"`
	ReplicatedDataRatio    float64 `koanf:"replicated_data_ratio" validate:"gte=0,lte=1"`
	NumEntitiesPerDatabase int     `koanf:"num_entities_per_database" validate:"gte=1"`
}

// DatabaseConfig holds database connection information
type DatabaseConfig struct {
	Name             string `koanf:"name" validate:"required"`
	ConnectionString string `koanf:"connection_string" validate:"required,url"`
}

// LoadConfig loads configuration from a JSON file with environment variable overrides
func LoadConfig(path string) (*Config, error) {
	k := koanf.New(".")

	// Load JSON config file
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load config file '%s': %w", path, err)
	}

	// Load environment variables with prefix (e.g., CONFIG_GENERATION_OUTPUT_DIR)
	if err := k.Load(env.Provider("CONFIG_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Basic validation use go playground/validator if needed
	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		return nil, fmt.Errorf("configuration validation error: %w", err)
	}

	return &config, nil
}

// NumDatabases returns the number of configured databases
func (c *Config) NumDatabases() int {
	return len(c.Databases)
}

// NumReplicatedEntities returns the number of replicated entities
func (c *Config) NumReplicatedEntities() int {
	return int(float64(c.Generation.NumEntitiesPerDatabase) * c.Generation.ReplicatedDataRatio)
}

// NumFragmentedEntities returns the number of fragmented entities
func (c *Config) NumFragmentedEntities() int {
	return c.Generation.NumEntitiesPerDatabase - c.NumReplicatedEntities()
}

// GetDatabaseOutputPath returns the output path for a database's collection file
func (c *Config) GetDatabaseOutputPath(dbIndex int, collection string) string {
	return fmt.Sprintf("%s/db%d_%s.json", c.Generation.OutputDir, dbIndex, collection)
}
