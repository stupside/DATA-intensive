package db

import "github.com/urfave/cli/v2"

func NewDBCommand() *cli.Command {
	return &cli.Command{
		Name:    "db",
		Aliases: []string{"d"},
		Usage:   "Database management commands",
		Subcommands: []*cli.Command{
			newSeedCommand(),
			newResetCommand(),
		},
	}
}
