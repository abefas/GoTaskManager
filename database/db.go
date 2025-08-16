package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abefas/GoTaskManager/models" // Adjust the import path as necessary

	"github.com/gorilla/mux"
)

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

	err := h.DB.QueryRow("INSERT INTO tasks(title, completed) VALUES($1, $2) RETURNING id", t.Title, t.Completed).Scan(&t.ID)
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
