package student

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
		Usage:   "Display information about a student",
		Action:  runInfo,
		Flags: []cli.Flag{
			flags.Database,
			flags.StudentID,
		},
	}
}

func runInfo(c *cli.Context) error {
	database := flags.GetDatabase(c)
	studentID := models.StudentID(flags.GetStudentID(c))

	validate := validator.New()
	if err := validate.Struct(&struct {
		database  string `validate:"required"`
		studentID string `validate:"required,alphanum"`
	}{
		database:  database,
		studentID: string(studentID),
	}); err != nil {
		return fmt.Errorf("invalid command line arguments: %w", err)
	}

	log.Debug().
		Str("database", database).
		Str("student_id", string(studentID)).
		Msg("fetching student info")

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

	// Fetch student information
	var student models.Student
	if err := db.Collection("students").FindOne(c.Context, bson.M{"id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student with ID '%s' not found: %w", studentID, err)
	}

	// Fetch department information if student is in a department
	var departmentName string
	if student.DepartmentID != "" {
		var department models.Department
		if err := db.Collection("departments").FindOne(c.Context, bson.M{"id": student.DepartmentID}).Decode(&department); err != nil {
			departmentName = "N/A"
		} else {
			departmentName = department.Name
		}
	} else {
		departmentName = "N/A"
	}

	// Fetch enrollments
	cursor, err := db.Collection("enrollments").Find(c.Context, bson.M{"student_id": studentID})
	if err != nil {
		return fmt.Errorf("failed to query enrollments: %w", err)
	}
	defer cursor.Close(c.Context)

	var enrollments []models.Enrollment
	if err := cursor.All(c.Context, &enrollments); err != nil {
		return fmt.Errorf("failed to decode enrollments: %w", err)
	}

	// Create table to display student information
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleColoredBright)
	t.SetTitle("Student Information")

	t.AppendRow(table.Row{"Student ID", student.ID})
	t.AppendRow(table.Row{"First Name", student.FirstName})
	t.AppendRow(table.Row{"Last Name", student.LastName})
	t.AppendRow(table.Row{"Email", student.Email})
	t.AppendRow(table.Row{"Department", departmentName})
	t.AppendRow(table.Row{"Department ID", student.DepartmentID})
	t.AppendRow(table.Row{"Total Enrollments", len(enrollments)})

	fmt.Println()
	t.Render()
	fmt.Println()

	// Display enrollments if any exist
	if len(enrollments) > 0 {
		// Fetch course details for each enrollment
		enrollmentTable := table.NewWriter()
		enrollmentTable.SetOutputMirror(os.Stdout)
		enrollmentTable.SetStyle(table.StyleColoredBright)
		enrollmentTable.SetTitle("Enrollments")

		enrollmentTable.AppendHeader(table.Row{"#", "Enrollment ID", "Course ID", "Course Name"})

		for i, enrollment := range enrollments {
			var course models.Course
			courseName := "N/A"
			if err := db.Collection("courses").FindOne(c.Context, bson.M{"id": enrollment.CourseID}).Decode(&course); err == nil {
				courseName = course.Name
			}

			enrollmentTable.AppendRow(table.Row{
				i + 1,
				enrollment.ID,
				enrollment.CourseID,
				courseName,
			})
		}

		enrollmentTable.AppendFooter(table.Row{"Total", len(enrollments), "", ""})

		enrollmentTable.Render()
		fmt.Println()
	}

	return nil
}
