package main

import (
	"log"
	"net/http"
	"os"

	"github.com/abefas/GoTaskManager/database"
	"github.com/abefas/GoTaskManager/handlers"   // Adjust the import path as necessary
	"github.com/abefas/GoTaskManager/middleware" // Import the new middleware package
	"github.com/joho/godotenv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize the database connection
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Initialize the handlers
	h := handlers.NewHandlers(db)

	// Set up the router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/register", h.RegisterUser).Methods("POST")
	router.HandleFunc("/login", h.LoginUser).Methods("POST")

	// Protected task routes. They are wrapped in our new middleware.
	tasksRouter := router.PathPrefix("/tasks").Subrouter()
	tasksRouter.Use(middleware.AuthMiddleware)
	tasksRouter.HandleFunc("", h.CreateTask).Methods("POST")
	tasksRouter.HandleFunc("", h.GetTasks).Methods("GET")
	tasksRouter.HandleFunc("/{id}", h.GetTask).Methods("GET")
	tasksRouter.HandleFunc("/{id}", h.UpdateTask).Methods("PUT")
	tasksRouter.HandleFunc("/{id}", h.DeleteTask).Methods("DELETE")

	// Start the server
	log.Printf("Starting server on :%s...", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
