package db

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func newResetCommand() *cli.Command {
	return &cli.Command{
		Name:    "reset",
		Aliases: []string{"r"},
		Usage:   "Reset all collections in all databases (drops all data)",
		Action:  runReset,
	}
}

func runReset(c *cli.Context) error {
	log.Debug().Str("path", flags.GetConfig(c)).Msg("loading configuration")

	cfg, err := config.LoadConfig(flags.GetConfig(c))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	collections := []string{"courses", "students", "professors", "departments", "enrollments"}

	log.Info().Msg("resetting databases...")

	for _, dbConfig := range cfg.Databases {
		log.Info().
			Str("database", dbConfig.Name).
			Msg("resetting database")

		client, err := internal.Client(c.Context, dbConfig.ConnectionString)
		if err != nil {
			return fmt.Errorf("failed to create database client for %s: %w", dbConfig.Name, err)
		}

		db := client.Database(dbConfig.Name)

		for _, collection := range collections {
			_, err := db.Collection(collection).DeleteMany(c.Context, bson.M{})
			if err != nil {
				client.Disconnect(context.Background())
				return fmt.Errorf("failed to reset collection %s in %s: %w", collection, dbConfig.Name, err)
			}

			log.Info().
				Str("database", dbConfig.Name).
				Str("collection", collection).
				Msg("collection reset")
		}

		client.Disconnect(context.Background())
	}

	fmt.Printf("✓ Database reset completed\n")
	fmt.Printf("  Databases reset: %d\n", cfg.NumDatabases())
	fmt.Printf("  Collections wiped per database: %d\n", len(collections))

	return nil
}
