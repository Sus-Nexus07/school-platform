package models

import "time"

type Course struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TeacherID   int       `json:"teacher_id"`
	CreatedAt   time.Time `json:"created_at"`
}
