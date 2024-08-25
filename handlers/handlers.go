package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"ipmanlk/simplefileserver/sqldb"
	"ipmanlk/simplefileserver/utils"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const UPLOADS_DIR = "./uploads"
const UPLOADS_DIRV2 = "./uploadsv2"

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
		log.Printf("Error retrieving file: %v", err)
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
		log.Printf("Error calculating hash: %v", err)
		http.Error(w, "Error calculating hash", http.StatusInternalServerError)
		return
	}

	// Seek to the beginning of the file
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		log.Printf("Error seeking file: %v", err)
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	// Check if file with same hash already exists in database
	_, existingUUID, _, err := sqldb.GetFileByHashV2(db, hash)

	if err == nil {
		// File with same hash already exists, return existing UUID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(existingUUID))
		return
	} else if err != sql.ErrNoRows {
		// Error occurred while querying database
		log.Printf("Error checking existing file: %v", err)
		http.Error(w, "Error checking existing file", http.StatusInternalServerError)
		return
	}

	// Generate unique UUID for the file
	uuid := utils.GenerateUUID()

	// Create Year/Month/Day directory structure
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	dir := filepath.Join(UPLOADS_DIRV2, year, month, day)

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	// Create file with UUID as name
	filePath := filepath.Join(dir, uuid+"_"+handler.Filename)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Copy file data to destination
	_, err = io.Copy(f, file)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Store UUID, file name, hash, and directory in new table
	err = sqldb.SaveFileV2(db, uuid, handler.Filename, hash, filePath)
	if err != nil {
		log.Printf("Error saving file info to database: %v", err)
		http.Error(w, "Error saving file info to database", http.StatusInternalServerError)
		return
	}

	// Return UUID as response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(uuid))
}

func HandleUploadFileFromURL(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if request has correct API key
	requestApiKey := r.Header.Get("x-api-key")
	if requestApiKey != os.Getenv("API_KEY") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	err := r.ParseMultipartForm(1 << 20) // Max upload size of 1MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get URL from form data
	fileUrl, ok := r.Form["url"]

	if !ok {
		http.Error(w, "URL parameter missing", http.StatusBadRequest)
		return
	}

	// Fetch the file from the URL
	resp, err := http.Get(fileUrl[0])
	if err != nil {
		log.Printf("Error fetching file: %v", err)
		http.Error(w, "Error fetching file", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Check if response status code is 200
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error fetching file: %v", resp.Status)
		http.Error(w, "Error fetching file", http.StatusInternalServerError)
		return
	}

	// Get filename from request form data
	filenames, ok := r.Form["filename"]
	var filename string
	if ok {
		filename = filenames[0]
	}

	// If filename is still empty, use a default name
	if filename == "" {
		filename = "unknown"
	}

	// Change response body to reusable reader
	// This is required because we need to read the response body twice
	resp.Body = io.NopCloser(utils.ReusableReader(resp.Body))

	// Calculate hash of the file
	hash, err := utils.CalculateFileHash(resp.Body)
	if err != nil {
		log.Printf("Error calculating hash: %v", err)
		http.Error(w, "Error calculating hash", http.StatusInternalServerError)
		return
	}

	// Check if file with same hash already exists in database
	_, existingUUID, _, err := sqldb.GetFileByHashV2(db, hash)

	if err == nil {
		// File with same hash already exists, return existing UUID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(existingUUID))
		return
	} else if err != sql.ErrNoRows {
		// Error occurred while querying database
		log.Printf("Error checking existing file: %v", err)
		http.Error(w, "Error checking existing file", http.StatusInternalServerError)
		return
	}

	// Generate unique UUID for the file
	uuid := utils.GenerateUUID()

	// Create Year/Month/Day directory structure
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	dir := filepath.Join(UPLOADS_DIRV2, year, month, day)

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	// Create file with UUID as name
	filePath := filepath.Join(dir, uuid+"_"+filename)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Copy file data to destination
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Store UUID, file name, hash, and directory in new table
	err = sqldb.SaveFileV2(db, uuid, filename, hash, filePath)
	if err != nil {
		log.Printf("Error saving file info to database: %v", err)
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
	filename, filePath, err := sqldb.GetFileByUUIDV2(db, ruuid)

	if err != nil {
		// If file is not found in the new table, check the old table
		if err == sql.ErrNoRows {
			filename, err = sqldb.GetFileByUUID(db, ruuid)
			filePath = filepath.Join(UPLOADS_DIR, ruuid+"_"+filename)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("Error retrieving file info from database: %v", err)
			http.Error(w, "Error retrieving file info from database", http.StatusInternalServerError)
			return
		}
	}

	// Open file for reading
	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		http.Error(w, "Error opeing file", http.StatusInternalServerError)
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
		log.Printf("Error streaming file: %v", err)
		http.Error(w, "Error streaming file", http.StatusInternalServerError)
		return
	}
}
