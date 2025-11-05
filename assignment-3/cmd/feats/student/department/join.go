package department

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func newJoinCommand() *cli.Command {
	return &cli.Command{
		Name:    "join",
		Aliases: []string{"j"},
		Usage:   "Add a student to a department",
		Action:  runJoin,
		Flags: []cli.Flag{
			flags.Database,
			flags.StudentID,
			flags.DepartmentID,
		},
	}
}

func runJoin(c *cli.Context) error {
	database := flags.GetDatabase(c)

	studentID := models.StudentID(flags.GetStudentID(c))
	departmentID := models.DepartmentID(flags.GetDepartmentID(c))

	validate := validator.New()
	if err := validate.Struct(&struct {
		database     string `validate:"required"`
		studentID    string `validate:"required,alphanum"`
		departmentID string `validate:"required,alphanum"`
	}{
		database:     database,
		studentID:    string(studentID),
		departmentID: string(departmentID),
	}); err != nil {
		return fmt.Errorf("invalid command line arguments: %w", err)
	}

	log.Debug().
		Str("database", database).
		Str("student_id", string(studentID)).
		Str("department_id", string(departmentID)).
		Msg("joining department")

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
		return fmt.Errorf("student with ID '%s' not found", studentID)
	}

	// Verify department exists
	departmentCount, err := db.Collection("departments").CountDocuments(c.Context, bson.M{"id": departmentID})
	if err != nil {
		return fmt.Errorf("failed to check department existence: %w", err)
	}
	if departmentCount == 0 {
		return fmt.Errorf("department with ID '%s' not found", departmentID)
	}

	// Update student's department
	if _, err := db.Collection("students").UpdateOne(c.Context, bson.M{"id": studentID}, bson.M{
		"$set": bson.M{
			"department_id": departmentID,
		},
	}); err != nil {
		return fmt.Errorf("failed to update student department: %w", err)
	}

	fmt.Printf("✓ Successfully added student %s to department %s\n", studentID, departmentID)

	return nil
}
