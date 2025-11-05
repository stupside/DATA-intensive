package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/department"
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/list"
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/mock"
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/student"
	"github.com/urfave/cli/v2"
)

const (
	appName    = "study"
	appUsage   = "A CLI tool to manage student and course data"
	appVersion = "1.0.0"
)

func main() {
	// Configure zerolog to write to stderr (keeps stdout clean for CLI output)
	zerolog.SetGlobalLevel(zerolog.WarnLevel) // Only show warnings and errors by default
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false})

	app := &cli.App{
		Name:  appName,
		Usage: appUsage,
		Flags: []cli.Flag{
			flags.Config,
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging",
			},
		},
		Version: appVersion,
		Before: func(c *cli.Context) error {
			if c.Bool("verbose") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			return nil
		},
		Commands: []*cli.Command{
			mock.NewMockCommand(),
			list.NewListCommand(),
			student.NewStudentCommand(),
			department.NewDepartmentCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("application error")
		os.Exit(1)
	}
}
