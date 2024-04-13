package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// Constants for directories and database
const (
	uploadsDir = "./uploads"
	dbFile     = "./data/data.db"
)

var apiKey string

// Load environmental variables from .env file
func init() {
	godotenv.Load()
	apiKey = os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Println("[WARNING] API_KEY not found in .env file, using default value")
		apiKey = "development"
	}
}

// Function to initialize SQLite database
func initializeDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	// Create file table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
					uuid TEXT PRIMARY KEY,
					filename TEXT,
					hash TEXT UNIQUE
				  )`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Function to generate a unique UUID for each file
func generateUUID() string {
	return uuid.New().String()
}

// Function to calculate SHA-256 hash of a file
func calculateHash(file io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Function to handle file uploads
func uploadFileHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if request has correct API key
	requestApiKey := r.Header.Get("x-api-key")
	if requestApiKey != apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // Max upload size of 10MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file from the request
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if file size is zero
	if handler.Size == 0 {
		http.Error(w, "Uploaded file is empty", http.StatusBadRequest)
		return
	}

	// Calculate hash of the file
	hash, err := calculateHash(file)
	if err != nil {
		http.Error(w, "Error calculating hash", http.StatusInternalServerError)
		return
	}

	// Check if file with same hash already exists in database
	var existingUUID string
	err = db.QueryRow("SELECT uuid FROM files WHERE hash=?", hash).Scan(&existingUUID)
	if err == nil {
		// File with same hash already exists, return existing UUID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(existingUUID))
		return
	} else if err != sql.ErrNoRows {
		// Error occurred while querying database
		http.Error(w, "Error checking existing file", http.StatusInternalServerError)
		return
	}

	// Seek back to the beginning of the file after calculating hash
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		http.Error(w, "Error seeking file", http.StatusInternalServerError)
		return
	}

	// Generate unique UUID for the file
	uuid := generateUUID()

	// Create file with UUID as name
	f, err := os.OpenFile(filepath.Join(uploadsDir, uuid+"_"+handler.Filename), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Copy file data to destination
	_, err = io.Copy(f, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Store UUID, file name, and hash in database
	_, err = db.Exec("INSERT INTO files (uuid, filename, hash) VALUES (?, ?, ?)", uuid, handler.Filename, hash)
	if err != nil {
		http.Error(w, "Error saving file info to database", http.StatusInternalServerError)
		return
	}

	// Return UUID as response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(uuid))
}

// Function to handle file downloads
func downloadFileHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get UUID from URL path
	ruuid := r.PathValue("uuid")

	// Check if UUID is valid
	if _, err := uuid.Parse(ruuid); err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	// Query database for filename associated with UUID
	var filename string
	err := db.QueryRow("SELECT filename FROM files WHERE uuid=?", ruuid).Scan(&filename)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving file info from database", http.StatusInternalServerError)
		return
	}

	// Open file for reading
	f, err := os.Open(filepath.Join(uploadsDir, ruuid+"_"+filename))
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Set Content-Type header
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream file content to response
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusInternalServerError)
		return
	}
}

func main() {
	// Initialize database
	db, err := initializeDB()
	if err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}
	defer db.Close()

	// Create a new empty servemux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		uploadFileHandler(w, r, db)
	})

	mux.HandleFunc("GET /files/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		downloadFileHandler(w, r, db)
	})

	// Start server
	fmt.Println("Server listening on port 8080...")
	http.ListenAndServe(":8080", mux)
}
