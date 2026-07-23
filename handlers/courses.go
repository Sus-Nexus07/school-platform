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

func CreateCourse(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Only teachers can create courses", http.StatusForbidden)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	var course models.Course
	query := `
		INSERT INTO courses (title, description, teacher_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, description, teacher_id, created_at
	`
	err = db.DB.QueryRow(query, input.Title, input.Description, int(claims.UserID)).Scan(
		&course.ID,
		&course.Title,
		&course.Description,
		&course.TeacherID,
		&course.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Error creating course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

func GetCourses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var query string
	var args []interface{}

	if claims.Role == "teacher" {
		query = `SELECT id, title, description, teacher_id, created_at FROM courses WHERE teacher_id = $1 ORDER BY created_at DESC`
		args = append(args, int(claims.UserID))
	} else {
		query = `SELECT id, title, description, teacher_id, created_at FROM courses ORDER BY created_at DESC`
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Error fetching courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		err := rows.Scan(
			&course.ID,
			&course.Title,
			&course.Description,
			&course.TeacherID,
			&course.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Error reading courses", http.StatusInternalServerError)
			return
		}
		courses = append(courses, course)
	}

	if courses == nil {
		courses = []models.Course{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

func EnrollCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "student" {
		http.Error(w, "Only students can enroll in courses", http.StatusForbidden)
		return
	}

	courseIDStr := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	courseIDStr = strings.TrimSuffix(courseIDStr, "/enroll")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	var exists bool
	err = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, courseID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Course not found", http.StatusNotFound)
		return
	}

	_, err = db.DB.Exec(
		`INSERT INTO enrollments (user_id, course_id) VALUES ($1, $2)`,
		int(claims.UserID), courseID,
	)
	if err != nil {
		http.Error(w, "Already enrolled or database error", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Enrolled successfully",
	})
}

func CourseRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/enroll") && r.Method == http.MethodPost:
		EnrollCourse(w, r)
	case strings.HasSuffix(path, "/lessons") && r.Method == http.MethodPost:
		CreateLesson(w, r)
	case strings.HasSuffix(path, "/lessons") && r.Method == http.MethodGet:
		GetLessons(w, r)
	case strings.HasSuffix(path, "/assignments") && r.Method == http.MethodPost:
		CreateAssignment(w, r)
	case strings.HasSuffix(path, "/assignments") && r.Method == http.MethodGet:
		GetAssignments(w, r)
	case r.Method == http.MethodDelete:
		DeleteCourse(w, r)
	default:
		http.Error(w, "Route not found", http.StatusNotFound)
	}
}

func DeleteCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	courseIDStr := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	var teacherID int
	err = db.DB.QueryRow(`SELECT teacher_id FROM courses WHERE id = $1`, courseID).Scan(&teacherID)
	if err != nil {
		http.Error(w, "Course not found", http.StatusNotFound)
		return
	}

	if claims.Role != "admin" && (claims.Role != "teacher" || int(claims.UserID) != teacherID) {
		http.Error(w, "You don't have permission to delete this course", http.StatusForbidden)
		return
	}

	_, err = db.DB.Exec(`DELETE FROM courses WHERE id = $1`, courseID)
	if err != nil {
		http.Error(w, "Error deleting course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Course deleted successfully",
	})
}
