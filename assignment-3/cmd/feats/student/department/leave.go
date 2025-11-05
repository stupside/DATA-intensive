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

func newLeaveCommand() *cli.Command {
	return &cli.Command{
		Name:    "leave",
		Aliases: []string{"l"},
		Usage:   "Remove a student from their department",
		Action:  runLeave,
		Flags: []cli.Flag{
			flags.Database,
			flags.StudentID,
		},
	}
}

func runLeave(c *cli.Context) error {
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
		Msg("leaving department")

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

	// Verify student exists and has a department
	var student models.Student
	if err := db.Collection("students").FindOne(c.Context, bson.M{"id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student with ID '%s' not found", studentID)
	}

	// Remove student's department
	_, err = db.Collection("students").UpdateOne(c.Context, bson.M{"id": studentID}, bson.M{
		"$unset": bson.M{
			"department_id": "",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to remove student from department: %w", err)
	}

	// Remove all enrollments for the student
	result, err := db.Collection("enrollments").DeleteMany(c.Context, bson.M{"student_id": studentID})
	if err != nil {
		return fmt.Errorf("failed to remove student enrollments: %w", err)
	}

	if result.DeletedCount > 0 {
		fmt.Printf("✓ Student '%s' has left their department and %d enrollment(s) removed\n", studentID, result.DeletedCount)
	} else {
		fmt.Printf("✓ Student '%s' has left their department\n", studentID)
	}

	return nil
}
