package models

import "fmt"

type CourseID string

func NewCourseID(id string) CourseID {
	return CourseID(fmt.Sprintf("course_%s", id))
}

type Course struct {
	ID           CourseID     `bson:"id,omitempty" json:"id"`
	Name         string       `bson:"name" json:"name"`
	ProfessorID  ProfessorID  `bson:"professor_id,omitempty" json:"professor_id,omitempty"`
	DepartmentID DepartmentID `bson:"department_id,omitempty" json:"department_id,omitempty"`
}
