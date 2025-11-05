package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/stupside/DATA-intensive/assignment-4/config"
	httpserver "github.com/stupside/DATA-intensive/assignment-4/http"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/mongo"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository/postgres"
	"github.com/urfave/cli/v3"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
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

	// Configure logging based on config
	configureLogging(cfg.Server.LogLevel)

	// Initialize databases
	pgDB, mongoClient, err := initializeDatabases(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeDatabases(ctx, pgDB, mongoClient)

	// Create and start server
	return startServer(cfg, pgDB, mongoClient)
}

// loadConfig loads the application configuration
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(configFile, configEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// configureLogging sets up the global logger with the specified log level
func configureLogging(levelStr string) {
	// Default to info level if not specified
	logLevel := slog.LevelInfo

	if levelStr != "" {
		switch strings.ToLower(levelStr) {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		default:
			slog.Warn("Unknown log level, defaulting to info", "level", levelStr)
		}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))
}

// initializeDatabases initializes PostgreSQL and MongoDB connections
func initializeDatabases(ctx context.Context, cfg *config.Config) (*gorm.DB, *mongodriver.Client, error) {
	slog.InfoContext(ctx, "Initializing PostgreSQL database")
	pgDB, err := postgres.ConnectPostgres(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	slog.InfoContext(ctx, "Initializing MongoDB")
	mongoClient, err := mongo.ConnectMongo(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	return pgDB, mongoClient, nil
}

// closeDatabases gracefully closes database connections
func closeDatabases(ctx context.Context, pgDB *gorm.DB, mongoClient *mongodriver.Client) {
	slog.InfoContext(ctx, "Closing database connections")

	if err := postgres.ClosePostgres(pgDB); err != nil {
		slog.ErrorContext(ctx, "Failed to close PostgreSQL", "error", err)
	}

	if err := mongo.CloseMongo(ctx, mongoClient); err != nil {
		slog.ErrorContext(ctx, "Failed to close MongoDB", "error", err)
	}
}

// startServer creates and starts the HTTP server
func startServer(cfg *config.Config, pgDB *gorm.DB, mongoClient *mongodriver.Client) error {
	// Build composite repo using concrete implementations
	dbName := cfg.Database.MongoDB.Database
	p := postgres.NewPostgresRepositories(pgDB)
	m := mongo.NewMongoRepositories(mongoClient, dbName)

	composite := &repository.CompositeRepositories{
		Device:      p.Device,
		Share:       p.Share,
		Certificate: p.Certificate,
		Connection:  p.Connection,
		Relay:       m.Relay,
		Streamer:    m.Streamer,
		Consumer:    m.Consumer,
		Chunk:       m.Chunk,
	}

	server, err := httpserver.New(cfg, composite)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	slog.Info("Server is starting", "port", cfg.Server.Port)

	if err := server.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
