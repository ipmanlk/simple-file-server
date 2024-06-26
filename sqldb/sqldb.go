package sqldb

import "database/sql"

// Constants for directories and database
const (
	UPLOADS_DIR = "./uploads"
	dbFile     = "./data/data.db"
)

// Initialize SQLite database
func Initialize() (*sql.DB, error) {
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

	// Create new table for files with directory structure
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files_v2 (
		uuid TEXT PRIMARY KEY,
		filename TEXT,
		hash TEXT UNIQUE,
		filepath TEXT
	)`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func GetFileByUUID(db *sql.DB, uuid string) (string, error) {
	var filename string
	err := db.QueryRow("SELECT filename FROM files WHERE uuid = ?", uuid).Scan(&filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func GetFileByUUIDV2(db *sql.DB, uuid string) (string, string, error) {
	var filename, filepath string
	err := db.QueryRow("SELECT filename, filepath FROM files_v2 WHERE uuid = ?", uuid).Scan(&filename, &filepath)
	if err != nil {
		return "", "", err
	}
	return filename, filepath, nil
}

func GetFileByHash(db *sql.DB, hash string) (string, string, error) {
	var filename, uuid string
	err := db.QueryRow("SELECT filename, uuid FROM files WHERE hash = ?", hash).Scan(&filename, &uuid)
	if err != nil {
		return "", "", err
	}
	return filename, uuid, nil
}

func GetFileByHashV2(db *sql.DB, hash string) (string, string, string, error) {
	var filename, uuid, filepath string
	err := db.QueryRow("SELECT filename, uuid, filepath FROM files_v2 WHERE hash = ?", hash).Scan(&filename, &uuid, &filepath)
	if err != nil {
		return "", "", "", err
	}
	return filename, uuid, filepath, nil
}

func SaveFile(db *sql.DB, uuid, filename, hash string) error {
	_, err := db.Exec("INSERT INTO files (uuid, filename, hash) VALUES (?, ?, ?)", uuid, filename, hash)
	return err
}

func SaveFileV2(db *sql.DB, uuid, filename, hash, filepath string) error {
	_, err := db.Exec("INSERT INTO files_v2 (uuid, filename, hash, filepath) VALUES (?, ?, ?, ?)", uuid, filename, hash, filepath)
	return err
}
