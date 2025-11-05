package mock

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/urfave/cli/v2"
)

// newGenCommand creates the generate subcommand
func newGenCommand() *cli.Command {
	return &cli.Command{
		Name:    "generate",
		Aliases: []string{"gen", "g"},
		Usage:   "Generate test data for distributed databases",
		Action:  runGen,
	}
}

func runGen(c *cli.Context) error {
	log.Debug().Str("path", flags.GetConfig(c)).Msg("loading configuration")

	cfg, err := config.LoadConfig(flags.GetConfig(c))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	log.Info().Msg("generating data...")

	if err := internal.GenerateAndWriteData(cfg); err != nil {
		return fmt.Errorf("data generation failed: %w", err)
	}

	fmt.Printf("✓ Data generation completed\n")
	fmt.Printf("  Output directory: %s\n", cfg.Generation.OutputDir)
	fmt.Printf("  Databases generated: %d\n", cfg.NumDatabases())

	return nil
}
