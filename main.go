package main

import (
	"fmt"
	"ipmanlk/simplefileserver/handlers"
	"ipmanlk/simplefileserver/sqldb"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load environment variables
	godotenv.Load()
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Println("[WARNING] API_KEY not found in .env file, using default value")
		os.Setenv("API KEY", "development")
	}

	// Initialize database
	db, err := sqldb.Initialize()
	if err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}
	defer db.Close()

	// Create a new empty servemux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("GET /", handlers.HandleHelloWorld)

	mux.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUploadFile(w, r, db)
	})

	mux.HandleFunc("POST /upload-url", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUploadFileFromURL(w, r, db)
	})

	mux.HandleFunc("GET /files/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleDownloadFile(w, r, db)
	})

	// Start server
	fmt.Println("Server listening on port 8080...")
	http.ListenAndServe(":8080", mux)
}
