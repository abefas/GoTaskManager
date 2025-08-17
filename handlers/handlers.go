package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/abefas/GoTaskManager/middleware" // Import the middleware package
	"github.com/abefas/GoTaskManager/models"     // Adjust the import path as necessary
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// A hardcoded secret key for signing JWTs.
// IMPORTANT: In a production app, this should be a strong, randomly generated
// value loaded from an environment variable.
var jwtKey = []byte("your_super_secret_jwt_key")

// Handlers struct holds the database connection, allowing methods to share it.
type Handlers struct {
	DB *sql.DB
}

// NewHandlers is a constructor for the Handlers struct.
func NewHandlers(db *sql.DB) *Handlers {
	return &Handlers{DB: db}
}

// respondWithJSON is a helper function to format and send JSON responses.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// hashPassword generates a bcrypt hash of the plain-text password.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPasswordHash compares a bcrypt password hash with a plain-text password.
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// RegisterUser handles a new user registration.
func (h *Handlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var user models.LoginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		log.Printf("JSON decode error in RegisterUser: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Hash the user's password.
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	var existingUser int
	// Check if the username already exists.
	err = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", user.Username).Scan(&existingUser)
	if err != nil {
		log.Printf("Database error checking for existing user: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if existingUser > 0 {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	var newUser models.User
	// Insert the new user into the database.
	err = h.DB.QueryRow("INSERT INTO users(username, password_hash) VALUES($1, $2) RETURNING id, created_at", user.Username, hashedPassword).Scan(&newUser.ID, &newUser.CreatedAt)
	if err != nil {
		log.Printf("Database error inserting new user: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	newUser.Username = user.Username
	respondWithJSON(w, http.StatusCreated, newUser)
}

// LoginUser handles user authentication and returns a JWT.
func (h *Handlers) LoginUser(w http.ResponseWriter, r *http.Request) {
	var loginRequest models.LoginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&loginRequest); err != nil {
		log.Printf("JSON decode error in LoginUser: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var storedPasswordHash string
	var storedUser models.User
	err := h.DB.QueryRow("SELECT id, username, password_hash FROM users WHERE username = $1", loginRequest.Username).Scan(&storedUser.ID, &storedUser.Username, &storedPasswordHash)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error retrieving user for login: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if !checkPasswordHash(loginRequest.Password, storedPasswordHash) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create JWT token with user ID in the claims
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		UserID: storedUser.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("Error signing token: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Login successful!", "token": tokenString})
}

// GetTasks retrieves all tasks from the database.
func (h *Handlers) GetTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, title, completed FROM tasks ORDER BY id ASC")
	if err != nil {
		http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed); err != nil {
			http.Error(w, "Failed to scan task row", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, "Error during row iteration", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, tasks)
}

// GetTask retrieves a single task by its ID.
func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var t models.Task
	err = h.DB.QueryRow("SELECT id, title, completed FROM tasks WHERE id = $1", id).Scan(&t.ID, &t.Title, &t.Completed)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, t)
}

// CreateTask creates a new task in the database.
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	var t models.Task
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&t); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Retrieve the user ID from the request context.
	// This is the new logic we need.
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		log.Println("User ID not found in context")
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}
	t.UserID = userID

	err := h.DB.QueryRow("INSERT INTO tasks(title, completed, user_id) VALUES($1, $2, $3) RETURNING id", t.Title, t.Completed, t.UserID).Scan(&t.ID)
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusCreated, t)
}

// UpdateTask updates an existing task.
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var t models.Task
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&t); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	res, err := h.DB.Exec("UPDATE tasks SET title=$1, completed=$2 WHERE id=$3", t.Title, t.Completed, id)
	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	t.ID = id
	respondWithJSON(w, http.StatusOK, t)
}

// DeleteTask deletes a task by its ID.
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	res, err := h.DB.Exec("DELETE FROM tasks WHERE id=$1", id)
	if err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
