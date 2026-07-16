package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"school_platform/db"

	"golang.org/x/crypto/bcrypt"
)

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
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
	err = db.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, input.Email).Scan(&userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If that email exists, a reset link has been generated",
		})
		return
	}

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(15 * time.Minute)

	_, err = db.DB.Exec(
		`INSERT INTO password_resets (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)
	if err != nil {
		http.Error(w, "Error creating reset token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":    "Reset link generated",
		"reset_link": "http://localhost:8080/reset-password.html?token=" + token,
	})
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(input.NewPassword) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	var userID int
	var expiresAt time.Time
	var used bool

	err = db.DB.QueryRow(
		`SELECT user_id, expires_at, used FROM password_resets WHERE token = $1`,
		input.Token,
	).Scan(&userID, &expiresAt, &used)

	if err != nil {
		http.Error(w, "Invalid or expired reset link", http.StatusBadRequest)
		return
	}

	if used {
		http.Error(w, "This reset link has already been used", http.StatusBadRequest)
		return
	}

	if time.Now().After(expiresAt) {
		http.Error(w, "This reset link has expired", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec(`UPDATE users SET password = $1 WHERE id = $2`, string(hashedPassword), userID)
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec(`UPDATE password_resets SET used = true WHERE token = $1`, input.Token)
	if err != nil {
		http.Error(w, "Error finalizing reset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password reset successfully! You can now log in.",
	})
}
