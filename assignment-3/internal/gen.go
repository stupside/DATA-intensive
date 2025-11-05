package internal

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"

	"github.com/stupside/DATA-intensive/assignment-3/internal/config"
	"github.com/stupside/DATA-intensive/assignment-3/internal/models"
)

var (
	courses     = []string{"Data Structures", "Algorithms", "Database Systems", "Web Development", "Machine Learning", "Networking", "Security", "Artificial Intelligence", "Mobile Development", "Cloud Computing"}
	firstNames  = []string{"John", "Jane", "Michael", "Sarah", "David", "Emma", "Robert", "Lisa", "James", "Maria"}
	lastNames   = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Martinez", "Lopez"}
	departments = []string{"Computer Science", "Engineering", "Mathematics", "Physics", "Business", "Medicine", "Law", "Arts", "Biology", "Chemistry"}
)

type GeneratedData struct {
	Departments []models.Department
	Courses     []models.Course
	Students    []models.Student
	Professors  []models.Professor
	Enrollments []models.Enrollment
}

// GenerateAndWriteData generates data based on config and writes to JSON files
func GenerateAndWriteData(cfg *config.Config) error {
	// Ensure output directory exists
	if err := os.MkdirAll(cfg.Generation.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	numReplicated := cfg.NumReplicatedEntities()
	numFragmented := cfg.NumFragmentedEntities()

	// Generate replicated data once (shared across all databases)
	replicatedData := generateReplicatedData(numReplicated)

	// Generate data for each database
	for i := 1; i <= cfg.NumDatabases(); i++ {
		if err := generateDatabaseFiles(cfg, i, replicatedData, numFragmented); err != nil {
			return fmt.Errorf("failed to generate files for database %d: %w", i, err)
		}
	}

	return nil
}

// generateDatabaseFiles generates and writes all collection files for a single database
func generateDatabaseFiles(cfg *config.Config, dbIndex int, replicatedData GeneratedData, numFragmented int) error {
	// Generate fragmented data for this database
	fragmentData := generateFragmentedData(numFragmented, dbIndex, replicatedData)

	// Combine replicated and fragmented data
	collections := map[string]any{
		"departments": append(replicatedData.Departments, fragmentData.Departments...),
		"professors":  append(replicatedData.Professors, fragmentData.Professors...),
		"courses":     append(replicatedData.Courses, fragmentData.Courses...),
		"students":    append(replicatedData.Students, fragmentData.Students...),
		"enrollments": append(replicatedData.Enrollments, fragmentData.Enrollments...),
	}

	// Write each collection to a file
	for collectionName, data := range collections {
		path := cfg.GetDatabaseOutputPath(dbIndex, collectionName)
		if err := writeToFile(path, data); err != nil {
			return fmt.Errorf("failed to write %s: %w", collectionName, err)
		}
	}

	return nil
}

// writeToFile writes data to a JSON file with proper formatting
func writeToFile(filename string, data any) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// generateFragmentedData creates fragment-specific data unique to each database
func generateFragmentedData(num int, dbIndex int, replicatedData GeneratedData) GeneratedData {
	offset := dbIndex * 1000

	data := GeneratedData{
		Departments: make([]models.Department, num),
		Professors:  make([]models.Professor, num),
		Courses:     make([]models.Course, num),
		Students:    make([]models.Student, num),
		Enrollments: make([]models.Enrollment, num),
	}

	// Collect all departments (replicated + fragmented being created) for student assignment
	allDepartments := make([]models.Department, 0, len(replicatedData.Departments)+num)
	allDepartments = append(allDepartments, replicatedData.Departments...)

	// Generate fragmented departments, professors, students first
	for i := range num {
		id := offset + i + 1

		data.Departments[i] = models.Department{
			ID:   models.NewDepartmentID(strconv.Itoa(id)),
			Name: fmt.Sprintf("DB%d %s", dbIndex, randomDepartment()),
		}
		allDepartments = append(allDepartments, data.Departments[i])

		data.Professors[i] = models.Professor{
			ID:        models.NewProfessorID(strconv.Itoa(id)),
			FirstName: randomFirstName(),
			LastName:  randomLastName(),
			Email:     randomEmail(fmt.Sprintf("db%d.prof", dbIndex), id),
		}

		// Assign student to any department (replicated or fragmented)
		data.Students[i] = models.Student{
			ID:           models.NewStudentID(strconv.Itoa(id)),
			FirstName:    randomFirstName(),
			LastName:     randomLastName(),
			Email:        randomEmail(fmt.Sprintf("db%d.student", dbIndex), id),
			DepartmentID: allDepartments[rand.IntN(len(allDepartments))].ID,
		}
	}

	// Collect all professors and students for course/enrollment generation
	allProfessors := append(replicatedData.Professors, data.Professors...)
	allStudents := append(replicatedData.Students, data.Students...)

	// Group all students by department
	studentsByDept := make(map[models.DepartmentID][]models.Student)
	for _, student := range allStudents {
		studentsByDept[student.DepartmentID] = append(studentsByDept[student.DepartmentID], student)
	}

	// Get list of departments that have students
	deptsWithStudents := make([]models.DepartmentID, 0, len(studentsByDept))
	for deptID := range studentsByDept {
		deptsWithStudents = append(deptsWithStudents, deptID)
	}

	// Generate fragmented courses and enrollments
	for i := range num {
		id := offset + i + 1

		// Pick a department that has students
		deptID := deptsWithStudents[rand.IntN(len(deptsWithStudents))]

		data.Courses[i] = models.Course{
			ID:           models.NewCourseID(strconv.Itoa(id)),
			Name:         fmt.Sprintf("DB%d %s", dbIndex, randomCourse()),
			ProfessorID:  allProfessors[rand.IntN(len(allProfessors))].ID,
			DepartmentID: deptID,
		}

		// Enroll a student from the same department
		studentsInDept := studentsByDept[deptID]
		student := studentsInDept[rand.IntN(len(studentsInDept))]

		data.Enrollments[i] = models.Enrollment{
			ID:        models.NewEnrollmentID(strconv.Itoa(id)),
			CourseID:  data.Courses[i].ID,
			StudentID: student.ID,
		}
	}

	return data
}

// generateReplicatedData creates data that will be replicated across all databases
func generateReplicatedData(num int) GeneratedData {
	if num == 0 {
		return GeneratedData{
			Departments: []models.Department{},
			Professors:  []models.Professor{},
			Courses:     []models.Course{},
			Students:    []models.Student{},
			Enrollments: []models.Enrollment{},
		}
	}

	data := GeneratedData{
		Departments: make([]models.Department, num),
		Professors:  make([]models.Professor, num),
		Courses:     make([]models.Course, num),
		Students:    make([]models.Student, num),
		Enrollments: make([]models.Enrollment, num),
	}

	// First pass: create departments, professors, and students
	for i := range num {
		id := i + 1

		data.Departments[i] = models.Department{
			ID:   models.NewDepartmentID(strconv.Itoa(id)),
			Name: randomDepartment(),
		}

		data.Professors[i] = models.Professor{
			ID:        models.NewProfessorID(strconv.Itoa(id)),
			FirstName: randomFirstName(),
			LastName:  randomLastName(),
			Email:     randomEmail("repl.prof", id),
		}

		// Assign student to a department
		data.Students[i] = models.Student{
			ID:           models.NewStudentID(strconv.Itoa(id)),
			FirstName:    randomFirstName(),
			LastName:     randomLastName(),
			Email:        randomEmail("repl.student", id),
			DepartmentID: data.Departments[rand.IntN(i+1)].ID,
		}
	}

	// Group students by department for enrollment matching
	studentsByDept := make(map[models.DepartmentID][]models.Student)
	for _, student := range data.Students {
		studentsByDept[student.DepartmentID] = append(studentsByDept[student.DepartmentID], student)
	}

	// Get list of departments that have students
	deptsWithStudents := make([]models.DepartmentID, 0, len(studentsByDept))
	for deptID := range studentsByDept {
		deptsWithStudents = append(deptsWithStudents, deptID)
	}

	// Second pass: create courses and enrollments
	for i := range num {
		id := i + 1

		// Pick a department that has students
		deptID := deptsWithStudents[rand.IntN(len(deptsWithStudents))]

		data.Courses[i] = models.Course{
			ID:           models.NewCourseID(strconv.Itoa(id)),
			Name:         randomCourse(),
			ProfessorID:  data.Professors[rand.IntN(len(data.Professors))].ID,
			DepartmentID: deptID,
		}

		// Enroll a student from the same department
		studentsInDept := studentsByDept[deptID]
		student := studentsInDept[rand.IntN(len(studentsInDept))]

		data.Enrollments[i] = models.Enrollment{
			ID:        models.NewEnrollmentID(strconv.Itoa(id)),
			CourseID:  data.Courses[i].ID,
			StudentID: student.ID,
		}
	}

	return data
}

func randomFirstName() string {
	return firstNames[rand.IntN(len(firstNames))]
}

func randomLastName() string {
	return lastNames[rand.IntN(len(lastNames))]
}

func randomCourse() string {
	return courses[rand.IntN(len(courses))]
}

func randomDepartment() string {
	return departments[rand.IntN(len(departments))]
}

func randomEmail(prefix string, id int) string {
	return fmt.Sprintf("%s%d@example.com", prefix, id)
}
