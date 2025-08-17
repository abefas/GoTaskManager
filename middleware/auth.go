// --- middleware/auth.go ---
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/abefas/GoTaskManager/models" // Adjust the import path as necessary
	"github.com/golang-jwt/jwt/v5"
)

// The same JWT key is used for signing and validation.
// In a production app, this would be a secure, non-exposed environment variable.
var jwtKey = []byte("your_super_secret_jwt_key")

// ContextKey is a custom type to avoid context key collisions.
type ContextKey string

// UserIDKey is the key we'll use to store the user's ID in the request context.
const UserIDKey ContextKey = "userId"

// AuthMiddleware checks for a valid JWT in the request header and
// adds the user ID to the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the token from the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("Authorization header missing")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// The token is in the format "Bearer <token>". We need to split this string.
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Invalid Authorization header format")
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		// Parse and validate the token.
		claims := &models.Claims{} // Use the Claims struct from the models package.
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				log.Println("Invalid token signature")
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			log.Printf("Token parsing error: %v", err)
			http.Error(w, "Invalid token", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			log.Println("Token is not valid")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add the user ID to the request context before passing it to the next handler.
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
