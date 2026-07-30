package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"school_platform/db"
	"school_platform/models"
	"school_platform/utils"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" || input.Email == "" || input.Password == "" {
		http.Error(w, "Name, email and password are required", http.StatusBadRequest)
		return
	}

	if input.Role == "" {
		input.Role = "student"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	var user models.User
	query := `
        INSERT INTO users (name, email, password, role)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, email, role, class_id, created_at
    `
	err = db.DB.QueryRow(query, input.Name, input.Email, string(hashedPassword), input.Role).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.ClassID,
		&user.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Email already exists or database error", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Email == "" || input.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	var user models.User
	var hashedPassword string
	query := `SELECT id, name, email, password, role, class_id FROM users WHERE email = $1`
	err = db.DB.QueryRow(query, input.Email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&hashedPassword,
		&user.Role,
		&user.ClassID,
	)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(input.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"role":   user.Role,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(utils.UserContextKey).(utils.UserClaims)
	if !ok {
		http.Error(w, "Could not read user from context", http.StatusInternalServerError)
		return
	}

	var user models.User
	err := db.DB.QueryRow(
		`SELECT id, name, email, role, class_id, created_at FROM users WHERE id = $1`,
		int(claims.UserID),
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.ClassID, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"class_id":   user.ClassID,
		"created_at": user.CreatedAt,
	}

	if user.ClassID != nil {
		var className, deptName string
		var deptID int
		err = db.DB.QueryRow(
			`SELECT classes.name, departments.id, departments.name
			 FROM classes
			 JOIN departments ON classes.department_id = departments.id
			 WHERE classes.id = $1`,
			*user.ClassID,
		).Scan(&className, &deptID, &deptName)

		if err == nil {
			response["class_name"] = className
			response["department_id"] = deptID
			response["department_name"] = deptName
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
