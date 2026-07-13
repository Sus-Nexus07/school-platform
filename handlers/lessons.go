package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"school_platform/db"
	"school_platform/models"
	"school_platform/utils"
)

func CreateLesson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "teacher" {
		http.Error(w, "Only teachers can create lessons", http.StatusForbidden)
		return
	}

	courseIDStr := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	courseIDStr = strings.TrimSuffix(courseIDStr, "/lessons")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	var owns bool
	err = db.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND teacher_id = $2)`,
		courseID, int(claims.UserID),
	).Scan(&owns)
	if err != nil || !owns {
		http.Error(w, "Course not found or you don't own it", http.StatusForbidden)
		return
	}

	var input struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Position int    `json:"position"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if input.Position == 0 {
		input.Position = 1
	}

	var lesson models.Lesson
	query := `
		INSERT INTO lessons (course_id, title, content, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, course_id, title, content, position, created_at
	`
	err = db.DB.QueryRow(query, courseID, input.Title, input.Content, input.Position).Scan(
		&lesson.ID,
		&lesson.CourseID,
		&lesson.Title,
		&lesson.Content,
		&lesson.Position,
		&lesson.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Error creating lesson", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lesson)
}

func GetLessons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	courseIDStr := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	courseIDStr = strings.TrimSuffix(courseIDStr, "/lessons")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	if claims.Role == "student" {
		var enrolled bool
		err = db.DB.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id = $1 AND course_id = $2)`,
			int(claims.UserID), courseID,
		).Scan(&enrolled)
		if err != nil || !enrolled {
			http.Error(w, "You are not enrolled in this course", http.StatusForbidden)
			return
		}
	}

	rows, err := db.DB.Query(
		`SELECT id, course_id, title, content, position, created_at 
		FROM lessons WHERE course_id = $1 ORDER BY position ASC`,
		courseID,
	)
	if err != nil {
		http.Error(w, "Error fetching lessons", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var lesson models.Lesson
		err := rows.Scan(
			&lesson.ID,
			&lesson.CourseID,
			&lesson.Title,
			&lesson.Content,
			&lesson.Position,
			&lesson.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Error reading lessons", http.StatusInternalServerError)
			return
		}
		lessons = append(lessons, lesson)
	}

	if lessons == nil {
		lessons = []models.Lesson{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lessons)
}

func LessonRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		CreateLesson(w, r)
	case http.MethodGet:
		GetLessons(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
