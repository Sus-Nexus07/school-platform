package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"school_platform/utils"
	"school_platform/db"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		secret := os.Getenv("JWT_SECRET")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userClaims := utils.UserClaims{
			UserID: claims["userId"].(float64),
			Role:   claims["role"].(string),
		}

		var verified bool
		err = db.DB.QueryRow(`SELECT verified FROM users WHERE id = $1`, int(userClaims.UserID)).Scan(&verified)
		if err != nil {
    		http.Error(w, "User not found", http.StatusUnauthorized)
    		return
		}

		if !verified {
    		http.Error(w, "Please verify your email before accessing this", http.StatusForbidden)
    		return
		}

		ctx := context.WithValue(r.Context(), utils.UserContextKey, userClaims)
		next.ServeHTTP(w, r.WithContext(ctx))

	}
}