package enrollments

import (
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/stupside/DATA-intensive/assignment-3/internal/flags"

	"github.com/stupside/DATA-intensive/assignment-3/internal"
	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func newAddCommand() *cli.Command {
	return &cli.Command{
		Name:    "add",
		Aliases: []string{"a"},
		Usage:   "Add an enrollment for a student to a course",
		Action:  runAdd,
		Flags: []cli.Flag{
			flags.Database,
			flags.CourseID,
			flags.StudentID,
		},
	}
}

func runAdd(c *cli.Context) error {
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
		Msg("adding enrollment")

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

	// Verify student exists and get student data
	var student models.Student
	if err := db.Collection("students").FindOne(c.Context, bson.M{"id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student with ID '%s' not found", studentID)
	}

	// Check if student is in a department
	if student.DepartmentID == "" {
		return fmt.Errorf("student must join a department before enrolling in courses")
	}

	// Verify course exists and get course data
	courseCollection := db.Collection("courses")
	var course models.Course
	err = courseCollection.FindOne(c.Context, bson.M{"id": courseID}).Decode(&course)
	if err != nil {
		return fmt.Errorf("course with ID '%s' not found", courseID)
	}

	// Verify course belongs to student's department
	if course.DepartmentID != student.DepartmentID {
		return fmt.Errorf("student can only enroll in courses from their department '%s', but course belongs to department '%s'", student.DepartmentID, course.DepartmentID)
	}

	// Check if enrollment already exists
	existingCount, err := db.Collection("enrollments").CountDocuments(c.Context, bson.M{
		"course_id":  courseID,
		"student_id": studentID,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing enrollment: %w", err)
	}
	if existingCount > 0 {
		return fmt.Errorf("student is already enrolled in this course")
	}

	// Generate new enrollment ID
	totalEnrollments, err := db.Collection("enrollments").CountDocuments(c.Context, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count enrollments: %w", err)
	}
	newEnrollmentID := models.NewEnrollmentID(strconv.Itoa(int(totalEnrollments) + 1))

	// Create enrollment
	enrollment := models.Enrollment{
		ID:        newEnrollmentID,
		CourseID:  courseID,
		StudentID: studentID,
	}

	_, err = db.Collection("enrollments").InsertOne(c.Context, enrollment)
	if err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	fmt.Printf("✓ Student '%s' enrolled in course '%s' (Enrollment ID: %s)\n", studentID, courseID, newEnrollmentID)

	return nil
}
