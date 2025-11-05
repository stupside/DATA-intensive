package models

type DepartmentID string

func NewDepartmentID(id string) DepartmentID {
	return DepartmentID("dept_" + id)
}

type Department struct {
	ID   DepartmentID `bson:"id,omitempty" json:"id"`
	Name string       `bson:"name" json:"name"`
}
