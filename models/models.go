// --- models/models.go ---
package models

import "github.com/golang-jwt/jwt/v5"

// Task represents a task in the to-do list.
type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	UserID    int    `json:"userId"`
}

// User represents a user in the system.
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Omit from JSON output for security
	CreatedAt    string `json:"createdAt"`
}

// LoginRequest defines the structure for user login and registration requests.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Claims defines the information stored in the JWT.
type Claims struct {
	UserID int `json:"userId"`
	jwt.RegisteredClaims
}
