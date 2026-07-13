package models

import "time"

type Lesson struct {
	ID        int       `json:"id"`
	CourseID  int       `json:"course_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}
