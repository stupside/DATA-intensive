package enrollments

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all enrollments for a student",
		Action:  runList,
		Flags: []cli.Flag{
			flags.Database,
			flags.StudentID,
		},
	}
}

type EnrollmentWithCourse struct {
	Course     *models.Course    `json:"course,omitempty"`
	Enrollment models.Enrollment `json:"enrollment"`
}

func runList(c *cli.Context) error {
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
		Msg("listing enrollments")

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

	// Verify student exists
	var student models.Student
	if err := db.Collection("students").FindOne(c.Context, bson.M{"id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student with ID '%s' not found: %w", studentID, err)
	}

	// Find all enrollments for the student
	cursor, err := db.Collection("enrollments").Find(c.Context, bson.M{"student_id": studentID})
	if err != nil {
		return fmt.Errorf("failed to query enrollments: %w", err)
	}
	defer cursor.Close(c.Context)

	var enrollments []models.Enrollment
	if err := cursor.All(c.Context, &enrollments); err != nil {
		return fmt.Errorf("failed to decode enrollments: %w", err)
	}

	if len(enrollments) == 0 {
		fmt.Printf("Student %s (%s %s) has no enrollments\n",
			studentID, student.FirstName, student.LastName)
		return nil
	}

	// Fetch course details for each enrollment
	var data []EnrollmentWithCourse
	for _, enrollment := range enrollments {
		var course models.Course
		if err := db.Collection("courses").FindOne(c.Context, bson.M{"id": enrollment.CourseID}).Decode(&course); err != nil {
			return fmt.Errorf("failed to fetch course details for course ID '%s': %w", enrollment.CourseID, err)
		}

		data = append(data, EnrollmentWithCourse{
			Course:     &course,
			Enrollment: enrollment,
		})
	}

	// Create table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleColoredBright)
	t.SetTitle(fmt.Sprintf("Enrollments for %s %s (ID: %s)",
		student.FirstName, student.LastName, studentID))

	t.AppendHeader(table.Row{"#", "Enrollment ID", "Course ID", "Course Name", "Department ID"})

	for i, e := range data {
		t.AppendRow(table.Row{
			i + 1,
			e.Enrollment.ID,
			e.Enrollment.CourseID,
			e.Course.Name,
			string(e.Course.DepartmentID),
		})
	}

	t.AppendFooter(table.Row{"Total", len(enrollments), "", "", ""})

	fmt.Println()
	t.Render()
	fmt.Println()

	return nil
}
