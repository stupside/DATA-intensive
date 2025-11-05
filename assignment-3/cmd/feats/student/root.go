package student

import (
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/student/department"
	"github.com/stupside/DATA-intensive/assignment-3/cmd/feats/student/enrollments"
	"github.com/urfave/cli/v2"
)

func NewStudentCommand() *cli.Command {
	return &cli.Command{
		Name:    "student",
		Aliases: []string{"s"},
		Usage:   "Manage students",
		Subcommands: []*cli.Command{
			newInfoCommand(),
			department.NewDepartmentCommand(),
			enrollments.NewEnrollmentCommand(),
		},
	}
}
