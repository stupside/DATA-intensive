package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

// Flag name constants
const (
	ConfigFlagName       = "config"
	DatabaseFlagName     = "database"
	IDFlagName           = "id"
	CollectionFlagName   = "collection"
	StudentIDFlagName    = "student-id"
	CourseIDFlagName     = "course-id"
	EnrollmentIDFlagName = "enrollment-id"
	DepartmentIDFlagName = "department-id"
	TimeoutFlagName      = "timeout"
)

// Global flags - used across multiple commands
var (
	// Config specifies the path to the configuration file
	Config = &cli.StringFlag{
		Name:     ConfigFlagName,
		Aliases:  []string{"cfg", "c"},
		Value:    "config.json",
		Usage:    "path to configuration file",
		EnvVars:  []string{"CONFIG"},
		Required: false,
	}

	// Database specifies which database to operate on
	Database = &cli.StringFlag{
		Name:     DatabaseFlagName,
		Aliases:  []string{"db", "d"},
		Usage:    "name of the database",
		EnvVars:  []string{"DATABASE"},
		Required: true,
	}

	// Timeout specifies operation timeout duration
	Timeout = &cli.DurationFlag{
		Name:    TimeoutFlagName,
		Aliases: []string{"t"},
		Value:   5 * time.Minute,
		Usage:   "operation timeout",
		EnvVars: []string{"TIMEOUT"},
	}
)

// Entity-specific flags
var (
	// ID specifies a generic entity ID
	ID = &cli.StringFlag{
		Name:     IDFlagName,
		Aliases:  []string{"i"},
		Usage:    "entity ID",
		EnvVars:  []string{"ENTITY_ID"},
		Required: true,
	}

	// Collection specifies the collection/entity name
	Collection = &cli.StringFlag{
		Name:     CollectionFlagName,
		Aliases:  []string{"col", "c"},
		Usage:    "collection/entity name",
		EnvVars:  []string{"COLLECTION"},
		Required: true,
	}

	// StudentID specifies a student ID
	StudentID = &cli.StringFlag{
		Name:     StudentIDFlagName,
		Aliases:  []string{"sid"},
		Usage:    "student ID",
		EnvVars:  []string{"STUDENT_ID"},
		Required: true,
	}

	// CourseID specifies a course ID
	CourseID = &cli.StringFlag{
		Name:     CourseIDFlagName,
		Aliases:  []string{"cid"},
		Usage:    "course ID",
		EnvVars:  []string{"COURSE_ID"},
		Required: true,
	}

	// EnrollmentID specifies an enrollment ID
	EnrollmentID = &cli.StringFlag{
		Name:     EnrollmentIDFlagName,
		Aliases:  []string{"eid"},
		Usage:    "enrollment ID",
		EnvVars:  []string{"ENROLLMENT_ID"},
		Required: true,
	}

	// DepartmentID specifies a department ID
	DepartmentID = &cli.StringFlag{
		Name:     DepartmentIDFlagName,
		Aliases:  []string{"did"},
		Usage:    "department ID",
		EnvVars:  []string{"DEPARTMENT_ID"},
		Required: true,
	}
)

// Helper functions to retrieve flag values
func GetConfig(c *cli.Context) string {
	return c.String(ConfigFlagName)
}

func GetDatabase(c *cli.Context) string {
	return c.String(DatabaseFlagName)
}

func GetID(c *cli.Context) string {
	return c.String(IDFlagName)
}

func GetCollection(c *cli.Context) string {
	return c.String(CollectionFlagName)
}

func GetStudentID(c *cli.Context) string {
	return c.String(StudentIDFlagName)
}

func GetCourseID(c *cli.Context) string {
	return c.String(CourseIDFlagName)
}

func GetEnrollmentID(c *cli.Context) string {
	return c.String(EnrollmentIDFlagName)
}

func GetTimeout(c *cli.Context) time.Duration {
	return c.Duration(TimeoutFlagName)
}

func GetDepartmentID(c *cli.Context) string {
	return c.String(DepartmentIDFlagName)
}
