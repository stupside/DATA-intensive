package internal

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/mongodb/mongo-tools/mongoimport"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
)

type databaseSource struct {
	path       string
	database   string
	collection string
}

// SeedFromConfig seeds databases based on the configuration
func SeedFromConfig(ctx context.Context, client *mongo.Client, cfg *config.Config) error {
	collections := []string{"courses", "students", "professors", "departments", "enrollments"}

	for i, dbConfig := range cfg.Databases {
		dbIndex := i + 1
		slog.Info("seeding database",
			slog.String("database", dbConfig.Name),
			slog.Int("index", dbIndex),
		)

		for _, collection := range collections {
			sourcePath := cfg.GetDatabaseOutputPath(dbIndex, collection)

			source := databaseSource{
				path:       sourcePath,
				database:   dbConfig.Name,
				collection: collection,
			}

			if err := loadData(dbConfig.ConnectionString, source); err != nil {
				return fmt.Errorf("failed to load data for %s.%s: %w", dbConfig.Name, collection, err)
			}

			slog.Info("loaded collection",
				slog.String("database", dbConfig.Name),
				slog.String("collection", collection),
				slog.String("file", sourcePath),
			)
		}
	}
	return nil
}

func loadData(connStr string, source databaseSource) error {
	args := []string{
		fmt.Sprintf("--db=%s", source.database),
		fmt.Sprintf("--uri=%s", connStr),
		fmt.Sprintf("--file=%s", source.path),
		fmt.Sprintf("--collection=%s", source.collection),
		"--type=json",
		"--jsonArray",
		"--drop",
		"--mode=insert",
	}

	opts, err := mongoimport.ParseOptions(args, "", "")
	if err != nil {
		return fmt.Errorf("failed to parse options: %w", err)
	}

	// Create and run importer
	imp, err := mongoimport.New(opts)
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}

	if _, _, err := imp.ImportDocuments(); err != nil {
		return fmt.Errorf("failed to import documents: %w", err)
	}

	return nil
}
