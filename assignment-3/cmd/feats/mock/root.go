package mock

import (
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/mock/db"
	"github.com/urfave/cli/v2"
)

func NewMockCommand() *cli.Command {
	return &cli.Command{
		Name:    "mock",
		Aliases: []string{"m"},
		Usage:   "Mock data and seeding commands",
		Subcommands: []*cli.Command{
			newGenCommand(),
			db.NewDBCommand(),
		},
	}
}
