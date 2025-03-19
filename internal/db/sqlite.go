// internal/db/sqlite.go
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// OpenSQLiteDB opens or creates a SQLite database
func OpenSQLiteDB(dsn string) (*sql.DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open the database
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create the sessions table if it doesn't exist
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sessions (
            token TEXT PRIMARY KEY,
            data BLOB NOT NULL,
            expiry REAL NOT NULL
        )
    `)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create index on expiry
	_, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry)
    `)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create sessions index: %w", err)
	}

	return db, nil
}