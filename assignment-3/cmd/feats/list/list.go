package list

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Generic list function that handles all entity types
func listEntities[T any](c *cli.Context, collectionName string, headers []string, rowMapper func(T) []interface{}) error {
	database := flags.GetDatabase(c)

	cfg, err := config.LoadConfig(flags.GetConfig(c))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	dbConfig, err := cfg.FindDatabaseByName(database)
	if err != nil {
		return err
	}

	client, err := internal.Client(c.Context, dbConfig.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create database client: %w", err)
	}
	defer client.Disconnect(c.Context)

	collection := client.Database(dbConfig.Name).Collection(collectionName)

	cursor, err := collection.Find(c.Context, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to query collection: %w", err)
	}
	defer cursor.Close(c.Context)

	var entities []T
	if err := cursor.All(c.Context, &entities); err != nil {
		return fmt.Errorf("failed to decode results: %w", err)
	}

	if len(entities) == 0 {
		fmt.Printf("No %s found in '%s'\n", collectionName, database)
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleColoredBright)
	t.SetTitle(fmt.Sprintf("%s in '%s'", collectionName, database))

	// Set header
	headerRow := make(table.Row, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	t.AppendHeader(headerRow)

	// Add rows
	for _, entity := range entities {
		t.AppendRow(rowMapper(entity))
	}

	t.AppendFooter(table.Row{"Total", len(entities)})

	fmt.Println()
	t.Render()
	fmt.Println()

	return nil
}

func listCourses(c *cli.Context) error {
	return listEntities(c, "courses",
		[]string{"ID", "Name", "Professor ID", "Department ID"},
		func(course models.Course) []interface{} {
			return []interface{}{course.ID, course.Name, course.ProfessorID, course.DepartmentID}
		})
}

func listStudents(c *cli.Context) error {
	return listEntities(c, "students",
		[]string{"ID", "First Name", "Last Name", "Email"},
		func(student models.Student) []interface{} {
			return []interface{}{student.ID, student.FirstName, student.LastName, student.Email}
		})
}

func listProfessors(c *cli.Context) error {
	return listEntities(c, "professors",
		[]string{"ID", "First Name", "Last Name", "Email"},
		func(professor models.Professor) []interface{} {
			return []interface{}{professor.ID, professor.FirstName, professor.LastName, professor.Email}
		})
}

func listDepartments(c *cli.Context) error {
	return listEntities(c, "departments",
		[]string{"ID", "Name"},
		func(dept models.Department) []interface{} {
			return []interface{}{dept.ID, dept.Name}
		})
}

func listEnrollments(c *cli.Context) error {
	return listEntities(c, "enrollments",
		[]string{"ID", "Course ID", "Student ID"},
		func(enrollment models.Enrollment) []interface{} {
			return []interface{}{enrollment.ID, enrollment.CourseID, enrollment.StudentID}
		})
}
