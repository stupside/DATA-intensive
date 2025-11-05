package models

type ProfessorID string

func NewProfessorID(id string) ProfessorID {
	return ProfessorID("prof_" + id)
}

type Professor struct {
	ID        ProfessorID `bson:"id,omitempty" json:"id"`
	Email     string      `bson:"email" json:"email"`
	LastName  string      `bson:"last_name" json:"last_name"`
	FirstName string      `bson:"first_name" json:"first_name"`
}
