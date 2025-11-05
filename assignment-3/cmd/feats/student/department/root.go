package department

import "github.com/urfave/cli/v2"

func NewDepartmentCommand() *cli.Command {
	return &cli.Command{
		Name:    "department",
		Aliases: []string{"dept"},
		Usage:   "Manage student department memberships",
		Subcommands: []*cli.Command{
			newJoinCommand(),
			newLeaveCommand(),
		},
	}
}
