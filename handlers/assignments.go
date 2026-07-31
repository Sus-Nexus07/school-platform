package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"school_platform/db"
	"school_platform/models"
	"school_platform/utils"
)

func CreateAssignment(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Only teachers can create assignments", http.StatusForbidden)
		return
	}

	courseIDStr := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	courseIDStr = strings.TrimSuffix(courseIDStr, "/assignments")
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
		Title       string `json:"title"`
		Description string `json:"description"`
		DueDate     string `json:"due_date"`
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

	dueDate, err := time.Parse("2006-01-02", input.DueDate)
	if err != nil {
		http.Error(w, "Invalid due date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	var assignment models.Assignment
	query := `
		INSERT INTO assignments (course_id, title, description, due_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, course_id, title, description, due_date, created_at
	`
	err = db.DB.QueryRow(query, courseID, input.Title, input.Description, dueDate).Scan(
		&assignment.ID,
		&assignment.CourseID,
		&assignment.Title,
		&assignment.Description,
		&assignment.DueDate,
		&assignment.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Error creating assignment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(assignment)
}

func SubmitAssignment(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Only students can submit assignments", http.StatusForbidden)
		return
	}

	assignmentIDStr := strings.TrimPrefix(r.URL.Path, "/api/assignments/")
	assignmentIDStr = strings.TrimSuffix(assignmentIDStr, "/submit")
	assignmentID, err := strconv.Atoi(assignmentIDStr)
	if err != nil {
		http.Error(w, "Invalid assignment ID", http.StatusBadRequest)
		return
	}

	var courseID int
	err = db.DB.QueryRow(
		`SELECT course_id FROM assignments WHERE id = $1`,
		assignmentID,
	).Scan(&courseID)
	if err != nil {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	var enrolled bool
	err = db.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id = $1 AND course_id = $2)`,
		int(claims.UserID), courseID,
	).Scan(&enrolled)
	if err != nil || !enrolled {
		http.Error(w, "You are not enrolled in this course", http.StatusForbidden)
		return
	}

	var input struct {
		Content string `json:"content"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	var submission models.Submission
	query := `
		INSERT INTO submissions (assignment_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, assignment_id, user_id, content, grade, submitted_at
	`
	err = db.DB.QueryRow(query, assignmentID, int(claims.UserID), input.Content).Scan(
		&submission.ID,
		&submission.AssignmentID,
		&submission.UserID,
		&submission.Content,
		&submission.Grade,
		&submission.SubmittedAt,
	)
	if err != nil {
		http.Error(w, "Already submitted or database error", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(submission)
}

func GradeSubmission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "teacher" {
		http.Error(w, "Only teachers can grade submissions", http.StatusForbidden)
		return
	}

	submissionIDStr := strings.TrimPrefix(r.URL.Path, "/api/submissions/")
	submissionIDStr = strings.TrimSuffix(submissionIDStr, "/grade")
	submissionID, err := strconv.Atoi(submissionIDStr)
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Grade int `json:"grade"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Grade < 0 || input.Grade > 100 {
		http.Error(w, "Grade must be between 0 and 100", http.StatusBadRequest)
		return
	}

	var submission models.Submission
	query := `
		UPDATE submissions
		SET grade = $1
		WHERE id = $2
		RETURNING id, assignment_id, user_id, content, grade, submitted_at
	`
	err = db.DB.QueryRow(query, input.Grade, submissionID).Scan(
		&submission.ID,
		&submission.AssignmentID,
		&submission.UserID,
		&submission.Content,
		&submission.Grade,
		&submission.SubmittedAt,
	)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

func GetAssignments(w http.ResponseWriter, r *http.Request) {
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
	courseIDStr = strings.TrimSuffix(courseIDStr, "/assignments")
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
		`SELECT id, course_id, title, description, due_date, created_at
		FROM assignments WHERE course_id = $1 ORDER BY due_date ASC`,
		courseID,
	)
	if err != nil {
		http.Error(w, "Error fetching assignments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var assignments []models.Assignment
	for rows.Next() {
		var a models.Assignment
		err := rows.Scan(&a.ID, &a.CourseID, &a.Title, &a.Description, &a.DueDate, &a.CreatedAt)
		if err != nil {
			http.Error(w, "Error reading assignments", http.StatusInternalServerError)
			return
		}
		assignments = append(assignments, a)
	}

	if assignments == nil {
		assignments = []models.Assignment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}

func AssignmentRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/submit") && r.Method == http.MethodPost:
		SubmitAssignment(w, r)
	default:
		http.Error(w, "Route not found", http.StatusNotFound)
	}
}

func SubmissionRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/grade") && r.Method == http.MethodPatch:
		GradeSubmission(w, r)
	default:
		http.Error(w, "Route not found", http.StatusNotFound)
	}
}

func GetAllAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var rows *sql.Rows
	var err error

	if claims.Role == "teacher" {
		rows, err = db.DB.Query(`
			SELECT assignments.id, assignments.course_id, assignments.title, 
			       assignments.description, assignments.due_date, assignments.created_at,
			       courses.title
			FROM assignments
			JOIN courses ON assignments.course_id = courses.id
			WHERE courses.teacher_id = $1
			ORDER BY assignments.due_date ASC
		`, int(claims.UserID))
	} else {
		rows, err = db.DB.Query(`
			SELECT assignments.id, assignments.course_id, assignments.title, 
			       assignments.description, assignments.due_date, assignments.created_at,
			       courses.title
			FROM assignments
			JOIN enrollments ON assignments.course_id = enrollments.course_id
			JOIN courses ON assignments.course_id = courses.id
			WHERE enrollments.user_id = $1
			ORDER BY assignments.due_date ASC
		`, int(claims.UserID))
	}

	if err != nil {
		http.Error(w, "Error fetching assignments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AssignmentWithCourse struct {
		models.Assignment
		CourseTitle string  `json:"course_title"`
		SubmittedAt *string `json:"submitted_at"`
		Grade       *int    `json:"grade"`
	}

	var assignments []AssignmentWithCourse
	for rows.Next() {
		var a AssignmentWithCourse
		err := rows.Scan(&a.ID, &a.CourseID, &a.Title, &a.Description, &a.DueDate, &a.CreatedAt, &a.CourseTitle)
		if err != nil {
			http.Error(w, "Error reading assignments", http.StatusInternalServerError)
			return
		}

		if claims.Role == "student" {
			db.DB.QueryRow(
				`SELECT grade FROM submissions WHERE assignment_id = $1 AND user_id = $2`,
				a.ID, int(claims.UserID),
			).Scan(&a.Grade)
		}

		assignments = append(assignments, a)
	}

	if assignments == nil {
		assignments = []AssignmentWithCourse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}
