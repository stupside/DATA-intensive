package list

import (
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/urfave/cli/v2"
)

func NewListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List entities from database",
		Subcommands: []*cli.Command{
			{
				Name:    "courses",
				Aliases: []string{"c"},
				Usage:   "List all courses",
				Action:  listCourses,
				Flags:   []cli.Flag{flags.Database},
			},
			{
				Name:    "students",
				Aliases: []string{"s"},
				Usage:   "List all students",
				Action:  listStudents,
				Flags:   []cli.Flag{flags.Database},
			},
			{
				Name:    "professors",
				Aliases: []string{"p"},
				Usage:   "List all professors",
				Action:  listProfessors,
				Flags:   []cli.Flag{flags.Database},
			},
			{
				Name:    "departments",
				Aliases: []string{"d"},
				Usage:   "List all departments",
				Action:  listDepartments,
				Flags:   []cli.Flag{flags.Database},
			},
			{
				Name:    "enrollments",
				Aliases: []string{"e"},
				Usage:   "List all enrollments",
				Action:  listEnrollments,
				Flags:   []cli.Flag{flags.Database},
			},
		},
	}
}
