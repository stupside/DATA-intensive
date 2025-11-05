package department

import (
	"github.com/urfave/cli/v2"
)

func NewDepartmentCommand() *cli.Command {
	return &cli.Command{
		Name:    "department",
		Aliases: []string{"d"},
		Usage:   "Manage departments",
		Subcommands: []*cli.Command{
			newInfoCommand(),
		},
	}
}
