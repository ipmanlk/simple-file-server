package sqldb

import "database/sql"

// Constants for directories and database
const (
	uploadsDir = "./uploads"
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

	return db, nil
}

func GetFileByUUID(db *sql.DB, uuid string) (string, string, error) {
	var filename, hash string
	err := db.QueryRow("SELECT filename, hash FROM files WHERE uuid = ?", uuid).Scan(&filename, &hash)
	if err != nil {
		return "", "", err
	}
	return filename, hash, nil
}

func GetFileByHash(db *sql.DB, hash string) (string, string, error) {
	var filename, uuid string
	err := db.QueryRow("SELECT filename, uuid FROM files WHERE hash = ?", hash).Scan(&filename, &uuid)
	if err != nil {
		return "", "", err
	}
	return filename, uuid, nil
}

func SaveFile(db *sql.DB, uuid, filename, hash string) error {
	_, err := db.Exec("INSERT INTO files (uuid, filename, hash) VALUES (?, ?, ?)", uuid, filename, hash)
	return err
}
