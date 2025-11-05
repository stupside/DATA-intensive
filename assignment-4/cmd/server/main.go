package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/database"
	httpserver "github.com/stupside/DATA-intensive/assignment-4/http"
	"github.com/urfave/cli/v3"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

const (
	configEnv  = "CONFIG"
	configFile = "config.yaml"
)

func main() {
	app := &cli.Command{
		Name:   "run",
		Usage:  "Run the server",
		Action: runServer,
	}

	// Set slog LogLevel to Debug
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	ctx := context.Background()

	if err := app.Run(ctx, nil); err != nil {
		slog.ErrorContext(ctx, "failed to run app", "error", err)
	}
}

func runServer(ctx context.Context, _ *cli.Command) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Initialize databases
	postgres, mongoClient, err := initializeDatabases(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeDatabases(ctx, postgres, mongoClient)

	// Create and start server
	return startServer(cfg, postgres, mongoClient)
}

// loadConfig loads the application configuration
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(configFile, configEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// initializeDatabases initializes PostgreSQL and MongoDB connections
func initializeDatabases(ctx context.Context, cfg *config.Config) (*gorm.DB, *mongo.Client, error) {
	slog.InfoContext(ctx, "Initializing PostgreSQL database")
	postgres, err := database.ConnectPostgres(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	slog.InfoContext(ctx, "Initializing MongoDB")
	mongoClient, err := database.ConnectMongo(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	return postgres, mongoClient, nil
}

// closeDatabases gracefully closes database connections
func closeDatabases(ctx context.Context, postgres *gorm.DB, mongoClient *mongo.Client) {
	slog.InfoContext(ctx, "Closing database connections")

	if err := database.ClosePostgres(postgres); err != nil {
		slog.ErrorContext(ctx, "Failed to close PostgreSQL", "error", err)
	}

	if err := database.CloseMongo(ctx, mongoClient); err != nil {
		slog.ErrorContext(ctx, "Failed to close MongoDB", "error", err)
	}
}

// startServer creates and starts the HTTP server
func startServer(cfg *config.Config, postgres *gorm.DB, mongoClient *mongo.Client) error {
	server, err := httpserver.New(cfg, postgres, mongoClient)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	slog.Info("Server is starting", "port", cfg.Server.Port)

	if err := server.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
