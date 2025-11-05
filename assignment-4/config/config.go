package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load reads configuration from YAML file and overrides with environment variables using koanf
func Load(path string, envPrefix string) (*Config, error) {
	// Create a new koanf instance
	k := koanf.New(".")

	// Load YAML config file
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("load config file: %w", err)
	}

	envProvider := env.Provider(strings.ToUpper(envPrefix)+"_", ".", func(s string) string {
		// Convert environment variable names to lowercase for matching
		// e.g., CONFIG_STORAGE_SQL_DSN -> storage.sql.dsn
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, strings.ToUpper(envPrefix)+"_")), "_", ".")
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("load environment variables: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "yaml",
	}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}
