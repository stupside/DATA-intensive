package models

type StudentID string

func NewStudentID(id string) StudentID {
	return StudentID("stud_" + id)
}

type Student struct {
	ID           StudentID    `bson:"id,omitempty" json:"id"`
	FirstName    string       `bson:"first_name" json:"first_name"`
	LastName     string       `bson:"last_name" json:"last_name"`
	Email        string       `bson:"email" json:"email"`
	DepartmentID DepartmentID `bson:"department_id,omitempty" json:"department_id,omitempty"`
}
