package enrollments

import "github.com/urfave/cli/v2"

func NewEnrollmentCommand() *cli.Command {
	return &cli.Command{
		Name:    "enrollment",
		Aliases: []string{"enroll", "en"},
		Usage:   "Manage student enrollments",
		Subcommands: []*cli.Command{
			newAddCommand(),
			newListCommand(),
			newRemoveCommand(),
		},
	}
}
