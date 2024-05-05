package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
)

// Generate a unique UUID for each file
func GenerateUUID() string {
	return uuid.New().String()
}

// Calculate SHA-256 hash of a file
func CalculateFileHash(file multipart.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Seek back to the beginning of the file after calculating hash
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
