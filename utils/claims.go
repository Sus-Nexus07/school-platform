package utils

type contextKey string

const UserContextKey contextKey = "user"

type UserClaims struct {
	UserID  float64
	Role  string 
}