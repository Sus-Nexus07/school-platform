package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
        RETURNING id, name, email, role, class_id, verified, created_at
    `
	err = db.DB.QueryRow(query, input.Name, input.Email, string(hashedPassword), input.Role).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.ClassID,
		&user.Verified,
		&user.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Email already exists or database error", http.StatusConflict)
		return
	}

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err == nil {
		token := hex.EncodeToString(tokenBytes)
		expiresAt := time.Now().Add(24 * time.Hour)

		_, dbErr := db.DB.Exec(
			`INSERT INTO email_verifications (user_id, token, expires_at) VALUES ($1, $2, $3)`,
			user.ID, token, expiresAt,
		)

		if dbErr == nil {
			verifyLink := "http://localhost:8080/verify-email.html?token=" + token
			emailBody := fmt.Sprintf(`
				<h2>Welcome to EduFlow, %s!</h2>
				<p>Please verify your email address to activate your account.</p>
				<p><a href="%s">Click here to verify your email</a></p>
				<p>This link expires in 24 hours.</p>
			`, user.Name, verifyLink)

			go func() {
    			sendErr := utils.SendEmail(user.Email, "Verify your EduFlow account", emailBody)
    			if sendErr != nil {
        			log.Println("EMAIL SEND ERROR:", sendErr)
    			} else {
        			log.Println("Email sent successfully to", user.Email)
    			}
			}()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Account created! Please check your email to verify your account.",
		"user":    user,
	})
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

func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Token string `json:"token"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var userID int
	var expiresAt time.Time

	err = db.DB.QueryRow(
		`SELECT user_id, expires_at FROM email_verifications WHERE token = $1`,
		input.Token,
	).Scan(&userID, &expiresAt)

	if err != nil {
		http.Error(w, "Invalid or expired verification link", http.StatusBadRequest)
		return
	}

	if time.Now().After(expiresAt) {
		http.Error(w, "This verification link has expired", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`UPDATE users SET verified = true WHERE id = $1`, userID)
	if err != nil {
		http.Error(w, "Error verifying account", http.StatusInternalServerError)
		return
	}

	db.DB.Exec(`DELETE FROM email_verifications WHERE token = $1`, input.Token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully! You can now log in.",
	})
}

func ResendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Email string `json:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var userID int
	var name string
	var verified bool
	err = db.DB.QueryRow(
		`SELECT id, name, verified FROM users WHERE email = $1`,
		input.Email,
	).Scan(&userID, &name, &verified)

	if err != nil || verified {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If that email exists and isn't verified, a new link has been sent",
		})
		return
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(24 * time.Hour)

	db.DB.Exec(
		`INSERT INTO email_verifications (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)

	verifyLink := "http://localhost:8080/verify-email.html?token=" + token
	emailBody := fmt.Sprintf(`
		<h2>Verify your EduFlow account, %s</h2>
		<p><a href="%s">Click here to verify your email</a></p>
		<p>This link expires in 24 hours.</p>
	`, name, verifyLink)

	go func() {
    	sendErr := utils.SendEmail(input.Email, "Verify your EduFlow account", emailBody)
    	if sendErr != nil {
        	log.Println("EMAIL SEND ERROR:", sendErr)
    	} else {
        	log.Println("Email sent successfully to", input.Email)
    	}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "If that email exists and isn't verified, a new link has been sent",
	})
}
