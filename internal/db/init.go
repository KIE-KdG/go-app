package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

type DB struct {
	*sql.DB
}

// OpenDB opens an existing database connection or creates a new one if it doesn't exist
func OpenDB(dsn string) (*sql.DB, error) {
	return InitializeDB(dsn)
}

// InitializeDB creates a new SQLite database file if it doesn't exist and sets up the schema
func InitializeDB(dsn string) (*sql.DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Check if the database already exists
	_, err := os.Stat(dsn)
	dbExists := !os.IsNotExist(err)

	// Open or create the database
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// If the database is new, initialize the schema
	if !dbExists {
		if err := createSchema(db); err != nil {
			db.Close()
			return nil, err
		}
		log.Printf("New database initialized at %s", dsn)
	} else {
		log.Printf("Using existing database at %s", dsn)
	}

	return db, nil
}

// createSchema sets up all the required tables and indices
func createSchema(db *sql.DB) error {
	// Create tables within a transaction for atomicity
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Enable foreign keys
	if _, err = tx.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return err
	}

	// Create users table
	if _, err = tx.Exec(`
	CREATE TABLE users (
			id BLOB PRIMARY KEY DEFAULT (uuid_blob(uuid())),
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			hashed_password TEXT NOT NULL,
			created DATETIME NOT NULL
	);`); err != nil {
		return err
	}

	// Create sessions table
	if _, err = tx.Exec(`
	CREATE TABLE sessions (
		token TEXT PRIMARY KEY,
		data BLOB NOT NULL,
		expiry REAL NOT NULL
	);`); err != nil {
		return err
	}

	// Create sessions index
	if _, err = tx.Exec(`
	CREATE INDEX sessions_expiry_idx ON sessions(expiry);`); err != nil {
		return err
	}

	// Create chats table with UUID extension enabled
	if _, err = tx.Exec(`
	CREATE TABLE chats (
			id BLOB PRIMARY KEY DEFAULT (uuid_blob(uuid())),
			user_id BLOB NOT NULL,
			created TIMESTAMP NOT NULL,
			last_activity TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
	);`); err != nil {
		return err
	}

	// Create chats indices
	if _, err = tx.Exec(`
	CREATE INDEX idx_chats_user_id ON chats(user_id);`); err != nil {
		return err
	}

	if _, err = tx.Exec(`
	CREATE INDEX idx_chats_last_activity ON chats(last_activity);`); err != nil {
		return err
	}

	// Create messages table
	if _, err = tx.Exec(`
	CREATE TABLE messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id BLOB NOT NULL,
		sender_type TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
	);`); err != nil {
		return err
	}

	// Create projects table
	if _, err = tx.Exec(`
	CREATE TABLE projects (
			id BLOB PRIMARY KEY DEFAULT (uuid_blob(uuid())),
			name TEXT NOT NULL,
			description TEXT,
			user_id BLOB NOT NULL,
			external_id TEXT,  -- Add this line
			created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`); err != nil {
		return err
	}

	// Create project indices
	if _, err = tx.Exec(`
CREATE INDEX idx_projects_user_id ON projects(user_id);`); err != nil {
		return err
	}
	if _, err = tx.Exec(`
CREATE TABLE files (
    id BLOB PRIMARY KEY DEFAULT (uuid_blob(uuid())),
    name TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    role TEXT NOT NULL,
    storage_location TEXT NOT NULL,
    project_id BLOB NOT NULL,
    user_id BLOB NOT NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'uploaded',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}

	// Create file indices
	if _, err = tx.Exec(`
CREATE INDEX idx_files_project_id ON files(project_id);
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_status ON files(status);
`); err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}
