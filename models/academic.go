package models

import "time"

type Department struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Class struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	DepartmentID int       `json:"department_id"`
	CreatedAt    time.Time `json:"created_at"`
}
