package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	handlers "github.com/abefas/GoTaskManager/database" // Adjust the import path as necessary

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Global variable for the database connection.
var db *sql.DB

func main() {
	// Load environment variables for database connection.
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("Missing required database environment variables. Please set DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, and DB_NAME.")
	}

	// Create the connection string.
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}
	log.Println("Successfully connected to the PostgreSQL database!")

	// Ensure the 'tasks' table exists.
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT FALSE
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create 'tasks' table: %v", err)
	}

	// Create a new router and a handlers instance with the database connection.
	router := mux.NewRouter()
	h := handlers.NewHandlers(db)

	// Define API routes and link them to the handler functions.
	router.HandleFunc("/tasks", h.GetTasks).Methods("GET")
	router.HandleFunc("/tasks", h.CreateTask).Methods("POST")
	router.HandleFunc("/tasks/{id}", h.GetTask).Methods("GET")
	router.HandleFunc("/tasks/{id}", h.UpdateTask).Methods("PUT")
	router.HandleFunc("/tasks/{id}", h.DeleteTask).Methods("DELETE")

	// Start the server.
	log.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}
