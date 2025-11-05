package db

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"
	"github.com/urfave/cli/v2"
)

func newSeedCommand() *cli.Command {
	return &cli.Command{
		Name:    "seed",
		Aliases: []string{"s"},
		Usage:   "Seed MongoDB databases from generated JSON files",
		Flags: []cli.Flag{
			flags.Timeout,
		},
		Action: runSeed,
	}
}

func runSeed(c *cli.Context) error {
	timeout := flags.GetTimeout(c)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Debug().Str("path", flags.GetConfig(c)).Msg("loading configuration")
	cfg, err := config.LoadConfig(flags.GetConfig(c))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Connect to MongoDB using the first database connection string
	if cfg.NumDatabases() == 0 {
		return fmt.Errorf("no databases configured")
	}

	log.Debug().Str("connection", maskConnectionString(cfg.Databases[0].ConnectionString)).Msg("connecting to MongoDB")

	client, err := internal.Client(ctx, cfg.Databases[0].ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Error().Err(err).Msg("failed to disconnect from MongoDB")
		}
	}()

	log.Info().Int("databases", cfg.NumDatabases()).Msg("seeding databases...")

	if err := internal.SeedFromConfig(ctx, client, cfg); err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	fmt.Printf("✓ Database seeding completed\n")
	fmt.Printf("  Databases seeded: %d\n", cfg.NumDatabases())

	return nil
}

// maskConnectionString masks sensitive parts of the connection string for logging
func maskConnectionString(connStr string) string {
	if len(connStr) > 20 {
		return connStr[:20] + "..."
	}
	return "***"
}
