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

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	rows, err := db.DB.Query(
		`SELECT id, name, email, role, class_id, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ClassID, &u.CreatedAt)
		if err != nil {
			http.Error(w, "Error reading users", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	if users == nil {
		users = []models.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	var totalUsers, totalStudents, totalTeachers, totalCourses, totalEnrollments int

	db.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'student'`).Scan(&totalStudents)
	db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'teacher'`).Scan(&totalTeachers)
	db.DB.QueryRow(`SELECT COUNT(*) FROM courses`).Scan(&totalCourses)
	db.DB.QueryRow(`SELECT COUNT(*) FROM enrollments`).Scan(&totalEnrollments)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"total_users":       totalUsers,
		"total_students":    totalStudents,
		"total_teachers":    totalTeachers,
		"total_courses":     totalCourses,
		"total_enrollments": totalEnrollments,
	})
}

func UpdateUserRole(w http.ResponseWriter, r *http.Request) {
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
	userIDStr = strings.TrimSuffix(userIDStr, "/role")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Role string `json:"role"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validRoles := map[string]bool{"student": true, "teacher": true, "admin": true}
	if !validRoles[input.Role] {
		http.Error(w, "Invalid role. Must be student, teacher, or admin", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.DB.QueryRow(
		`UPDATE users SET role = $1 WHERE id = $2 RETURNING id, name, email, role, class_id, created_at`,
		input.Role, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.ClassID, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func AdminUserRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/role") && r.Method == http.MethodPatch:
		UpdateUserRole(w, r)
	case strings.HasSuffix(path, "/class") && r.Method == http.MethodPatch:
		AssignStudentClass(w, r)
	default:
		http.Error(w, "Route not found", http.StatusNotFound)
	}
}
