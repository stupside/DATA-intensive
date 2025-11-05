package enrollments

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

func newRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove a student's enrollment from a course",
		Action:  runRemove,
		Flags: []cli.Flag{
			flags.Database,
			flags.CourseID,
			flags.StudentID,
		},
	}
}

func runRemove(c *cli.Context) error {
	database := flags.GetDatabase(c)

	courseID := models.CourseID(flags.GetCourseID(c))
	studentID := models.StudentID(flags.GetStudentID(c))

	validate := validator.New()
	if err := validate.Struct(&struct {
		database  string `validate:"required"`
		courseID  string `validate:"required,alphanum"`
		studentID string `validate:"required,alphanum"`
	}{
		database:  database,
		courseID:  string(courseID),
		studentID: string(studentID),
	}); err != nil {
		return fmt.Errorf("invalid command line arguments: %w", err)
	}

	log.Debug().
		Str("database", database).
		Str("student_id", string(studentID)).
		Str("course_id", string(courseID)).
		Msg("removing enrollment")

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

	// Delete the enrollment
	result, err := db.Collection("enrollments").DeleteOne(c.Context, bson.M{
		"course_id":  courseID,
		"student_id": studentID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete enrollment: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no enrollment found for student %s in course %s", studentID, courseID)
	}

	fmt.Printf("✓ Successfully removed enrollment for student %s from course %s\n", studentID, courseID)

	return nil
}
