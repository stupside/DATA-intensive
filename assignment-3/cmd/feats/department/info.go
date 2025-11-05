package department

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func newInfoCommand() *cli.Command {
	return &cli.Command{
		Name:    "info",
		Aliases: []string{"i"},
		Usage:   "Display information about a department",
		Action:  runInfo,
		Flags: []cli.Flag{
			flags.Database,
			flags.DepartmentID,
		},
	}
}

func runInfo(c *cli.Context) error {
	database := flags.GetDatabase(c)
	departmentID := models.DepartmentID(flags.GetDepartmentID(c))

	validate := validator.New()
	if err := validate.Struct(&struct {
		database     string `validate:"required"`
		departmentID string `validate:"required,alphanum"`
	}{
		database:     database,
		departmentID: string(departmentID),
	}); err != nil {
		return fmt.Errorf("invalid command line arguments: %w", err)
	}

	log.Debug().
		Str("database", database).
		Str("department_id", string(departmentID)).
		Msg("fetching department info")

	// Load configuration
	cfg, err := config.LoadConfig(flags.GetConfig(c))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	dbConfig, err := cfg.FindDatabaseByName(database)
	if err != nil {
		return err
	}

	// Connect to database
	client, err := internal.Client(c.Context, dbConfig.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create database client: %w", err)
	}
	defer client.Disconnect(c.Context)

	db := client.Database(dbConfig.Name)

	// Fetch department information
	var department models.Department
	if err := db.Collection("departments").FindOne(c.Context, bson.M{"id": departmentID}).Decode(&department); err != nil {
		return fmt.Errorf("department with ID '%s' not found: %w", departmentID, err)
	}

	// Fetch professors in this department
	professorCursor, err := db.Collection("professors").Find(c.Context, bson.M{"department_id": departmentID})
	if err != nil {
		return fmt.Errorf("failed to query professors: %w", err)
	}
	defer professorCursor.Close(c.Context)

	var professors []models.Professor
	if err := professorCursor.All(c.Context, &professors); err != nil {
		return fmt.Errorf("failed to decode professors: %w", err)
	}

	// Fetch students in this department
	studentCursor, err := db.Collection("students").Find(c.Context, bson.M{"department_id": departmentID})
	if err != nil {
		return fmt.Errorf("failed to query students: %w", err)
	}
	defer studentCursor.Close(c.Context)

	var students []models.Student
	if err := studentCursor.All(c.Context, &students); err != nil {
		return fmt.Errorf("failed to decode students: %w", err)
	}

	// Fetch courses in this department
	courseCursor, err := db.Collection("courses").Find(c.Context, bson.M{"department_id": departmentID})
	if err != nil {
		return fmt.Errorf("failed to query courses: %w", err)
	}
	defer courseCursor.Close(c.Context)

	var courses []models.Course
	if err := courseCursor.All(c.Context, &courses); err != nil {
		return fmt.Errorf("failed to decode courses: %w", err)
	}

	// Create table to display department information
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleColoredBright)
	t.SetTitle("Department Information")

	t.AppendRow(table.Row{"Department ID", department.ID})
	t.AppendRow(table.Row{"Name", department.Name})
	t.AppendRow(table.Row{"Total Professors", len(professors)})
	t.AppendRow(table.Row{"Total Students", len(students)})
	t.AppendRow(table.Row{"Total Courses", len(courses)})

	fmt.Println()
	t.Render()
	fmt.Println()

	// Display professors if any exist
	if len(professors) > 0 {
		professorTable := table.NewWriter()
		professorTable.SetOutputMirror(os.Stdout)
		professorTable.SetStyle(table.StyleColoredBright)
		professorTable.SetTitle("Professors")

		professorTable.AppendHeader(table.Row{"#", "Professor ID", "First Name", "Last Name", "Email"})

		for i, professor := range professors {
			professorTable.AppendRow(table.Row{
				i + 1,
				professor.ID,
				professor.FirstName,
				professor.LastName,
				professor.Email,
			})
		}

		professorTable.AppendFooter(table.Row{"Total", len(professors), "", "", ""})

		professorTable.Render()
		fmt.Println()
	}

	// Display students if any exist
	if len(students) > 0 {
		studentTable := table.NewWriter()
		studentTable.SetOutputMirror(os.Stdout)
		studentTable.SetStyle(table.StyleColoredBright)
		studentTable.SetTitle("Students")

		studentTable.AppendHeader(table.Row{"#", "Student ID", "First Name", "Last Name", "Email"})

		for i, student := range students {
			studentTable.AppendRow(table.Row{
				i + 1,
				student.ID,
				student.FirstName,
				student.LastName,
				student.Email,
			})
		}

		studentTable.AppendFooter(table.Row{"Total", len(students), "", "", ""})

		studentTable.Render()
		fmt.Println()
	}

	// Display courses if any exist
	if len(courses) > 0 {
		courseTable := table.NewWriter()
		courseTable.SetOutputMirror(os.Stdout)
		courseTable.SetStyle(table.StyleColoredBright)
		courseTable.SetTitle("Courses")

		courseTable.AppendHeader(table.Row{"#", "Course ID", "Course Name", "Professor ID"})

		for i, course := range courses {
			courseTable.AppendRow(table.Row{
				i + 1,
				course.ID,
				course.Name,
				course.ProfessorID,
			})
		}

		courseTable.AppendFooter(table.Row{"Total", len(courses), "", ""})

		courseTable.Render()
		fmt.Println()
	}

	return nil
}
