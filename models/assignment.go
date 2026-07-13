package models

import "time"

type Assignment struct {
	ID          int       `json:"id"`
	CourseID    int       `json:"course_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
}

type Submission struct {
	ID           int       `json:"id"`
	AssignmentID int       `json:"assignment_id"`
	UserID       int       `json:"user_id"`
	Content      string    `json:"content"`
	Grade        *int      `json:"grade"`
	SubmittedAt  time.Time `json:"submitted_at"`
}
