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

func CreateDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.Name == "" {
		http.Error(w, "Department name is required", http.StatusBadRequest)
		return
	}

	var dept models.Department
	err = db.DB.QueryRow(
		`INSERT INTO departments (name) VALUES ($1) RETURNING id, name, created_at`,
		input.Name,
	).Scan(&dept.ID, &dept.Name, &dept.CreatedAt)

	if err != nil {
		http.Error(w, "Department already exists or database error", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dept)
}

func GetDepartments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.DB.Query(`SELECT id, name, created_at FROM departments ORDER BY name ASC`)
	if err != nil {
		http.Error(w, "Error fetching departments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt); err != nil {
			http.Error(w, "Error reading departments", http.StatusInternalServerError)
			return
		}
		departments = append(departments, d)
	}

	if departments == nil {
		departments = []models.Department{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(departments)
}

func CreateClass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	var input struct {
		Name         string `json:"name"`
		DepartmentID int    `json:"department_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.Name == "" || input.DepartmentID == 0 {
		http.Error(w, "Class name and department_id are required", http.StatusBadRequest)
		return
	}

	var class models.Class
	err = db.DB.QueryRow(
		`INSERT INTO classes (name, department_id) VALUES ($1, $2) RETURNING id, name, department_id, created_at`,
		input.Name, input.DepartmentID,
	).Scan(&class.ID, &class.Name, &class.DepartmentID, &class.CreatedAt)

	if err != nil {
		http.Error(w, "Invalid department or database error", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(class)
}

func GetClasses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.DB.Query(`SELECT id, name, department_id, created_at FROM classes ORDER BY name ASC`)
	if err != nil {
		http.Error(w, "Error fetching classes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var classes []models.Class
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(&c.ID, &c.Name, &c.DepartmentID, &c.CreatedAt); err != nil {
			http.Error(w, "Error reading classes", http.StatusInternalServerError)
			return
		}
		classes = append(classes, c)
	}

	if classes == nil {
		classes = []models.Class{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(classes)
}

func AssignStudentClass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	userIDStr := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	userIDStr = strings.TrimSuffix(userIDStr, "/class")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var input struct {
		ClassID int `json:"class_id"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.DB.QueryRow(
		`UPDATE users SET class_id = $1 WHERE id = $2 RETURNING id, name, email, role, class_id, created_at`,
		input.ClassID, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.ClassID, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found or invalid class", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func GetStudentGPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentIDStr := strings.TrimPrefix(r.URL.Path, "/api/students/")
	studentIDStr = strings.TrimSuffix(studentIDStr, "/gpa")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	if claims.Role == "student" && int(claims.UserID) != studentID {
		http.Error(w, "You can only view your own GPA", http.StatusForbidden)
		return
	}

	var avgGrade *float64
	err = db.DB.QueryRow(
		`SELECT AVG(grade) FROM submissions WHERE user_id = $1 AND grade IS NOT NULL`,
		studentID,
	).Scan(&avgGrade)

	if err != nil {
		http.Error(w, "Error calculating GPA", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"student_id":    studentID,
		"average_grade": nil,
		"gpa":           nil,
		"graded_count":  0,
	}

	if avgGrade != nil {
		gpa := (*avgGrade / 100) * 4.0
		response["average_grade"] = *avgGrade
		response["gpa"] = gpa
	}

	var gradedCount int
	db.DB.QueryRow(
		`SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND grade IS NOT NULL`,
		studentID,
	).Scan(&gradedCount)
	response["graded_count"] = gradedCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
