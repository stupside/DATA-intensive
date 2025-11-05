package models

type EnrollmentID string

func NewEnrollmentID(id string) EnrollmentID {
	return EnrollmentID("enroll_" + id)
}

type Enrollment struct {
	ID        EnrollmentID `bson:"id,omitempty" json:"id"`
	CourseID  CourseID     `bson:"course_id" json:"course_id"`
	StudentID StudentID    `bson:"student_id" json:"student_id"`
}
