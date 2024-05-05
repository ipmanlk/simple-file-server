package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"ipmanlk/simplefileserver/sqldb"
	"ipmanlk/simplefileserver/utils"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Constants for directories and database
const (
	uploadsDir = "./uploads"
	dbFile     = "./data/data.db"
)

func HandleHelloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func HandleUploadFile(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if request has correct API key
	requestApiKey := r.Header.Get("x-api-key")
	if requestApiKey != os.Getenv("API_KEY") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // Max upload size of 10MB
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
	hash, err := utils.CalculateFileHash(file)
	if err != nil {
		http.Error(w, "Error calculating hash", http.StatusInternalServerError)
		return
	}

	// Check if file with same hash already exists in database
	_, existingUUID, err := sqldb.GetFileByHash(db, hash)

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

	// Generate unique UUID for the file
	uuid := utils.GenerateUUID()

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
	err = sqldb.SaveFile(db, uuid, handler.Filename, hash)
	if err != nil {
		http.Error(w, "Error saving file info to database", http.StatusInternalServerError)
		return
	}

	// Return UUID as response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(uuid))
}

func HandleDownloadFile(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get UUID from URL path
	ruuid := r.PathValue("uuid")

	// Check if UUID is valid
	if _, err := uuid.Parse(ruuid); err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	// Query database for filename associated with UUID
	filename, _, err := sqldb.GetFileByUUID(db, ruuid)

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
	// Set Content-Disposition header to force browser to download with proper filename
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Stream file content to response
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusInternalServerError)
		return
	}
}
