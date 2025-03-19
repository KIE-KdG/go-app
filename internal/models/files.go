// models/files.go
package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// File represents a document in the system
type File struct {
	ID              uuid.UUID
	Name            string
	Description     string
	FilePath        string
	MimeType        string
	Size            int64
	Role            string
	StorageLocation string
	ProjectID       uuid.UUID
	UserID          uuid.UUID
	UploadedAt      time.Time
	ProcessedAt     sql.NullTime
	Status          string
}

type FileModel struct {
	DB *sql.DB
}

func NewFileModel(db *sql.DB) *FileModel {
	return &FileModel{DB: db}
}

func (m *FileModel) Insert(file *File) error {
	// Generate a new UUID if not provided
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}

	// Insert into files table
	stmt := `
		INSERT INTO files (
			id, name, description, file_path, mime_type, size, 
			role, storage_location, uploaded_at, status, owner_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, $10)
		RETURNING id
	`

	var id uuid.UUID
	err := m.DB.QueryRow(
		stmt,
		file.ID,
		file.Name,
		file.Description,
		file.FilePath,
		file.MimeType,
		file.Size,
		file.Role,
		file.StorageLocation,
		file.Status,
		file.UserID,
	).Scan(&id)

	if err != nil {
		return err
	}

	// Link to project if ProjectID is provided
	if file.ProjectID != uuid.Nil {
		linkStmt := `
			INSERT INTO files_projects (file_id, project_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`
		_, err = m.DB.Exec(linkStmt, id, file.ProjectID)
		if err != nil {
			// Log but don't fail on linking error
			return err
		}
	}

	return nil
}

func (m *FileModel) GetByID(id uuid.UUID) (*File, error) {
	stmt := `
		SELECT f.id, f.name, f.description, f.file_path, f.mime_type, f.size,
			   f.role, f.storage_location, f.uploaded_at, f.processed_at, f.status, f.owner_id
		FROM files f
		WHERE f.id = $1
	`

	var file File
	var description, filePath sql.NullString
	
	err := m.DB.QueryRow(stmt, id).Scan(
		&file.ID,
		&file.Name,
		&description,
		&filePath,
		&file.MimeType,
		&file.Size,
		&file.Role,
		&file.StorageLocation,
		&file.UploadedAt,
		&file.ProcessedAt,
		&file.Status,
		&file.UserID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	if description.Valid {
		file.Description = description.String
	}
	
	if filePath.Valid {
		file.FilePath = filePath.String
	}

	// Get ProjectID from the many-to-many relationship
	projectStmt := `
		SELECT project_id FROM files_projects WHERE file_id = $1 LIMIT 1
	`
	err = m.DB.QueryRow(projectStmt, id).Scan(&file.ProjectID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &file, nil
}

func (m *FileModel) GetByProject(projectID uuid.UUID) ([]*File, error) {
	stmt := `
		SELECT f.id, f.name, f.file_path, f.storage_location, f.owner_id
		FROM files f
		JOIN files_projects fp ON f.id = fp.file_id
		WHERE fp.project_id = $1
	`

	rows, err := m.DB.Query(stmt, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*File{}
	for rows.Next() {
		var file File
		var description, filePath sql.NullString
		
		err := rows.Scan(
			&file.ID,
			&file.Name,
			&filePath,
			&file.StorageLocation,
			&file.UserID,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			file.Description = description.String
		}
		
		if filePath.Valid {
			file.FilePath = filePath.String
		}

		file.ProjectID = projectID
		files = append(files, &file)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

func (m *FileModel) UpdateStatus(id uuid.UUID, status string) error {
	var stmt string
	var args []interface{}

	if status == "processed" {
		stmt = `
			UPDATE files
			SET status = $1, processed_at = NOW()
			WHERE id = $2
		`
		args = []interface{}{status, id}
	} else {
		stmt = `
			UPDATE files
			SET status = $1
			WHERE id = $2
		`
		args = []interface{}{status, id}
	}

	_, err := m.DB.Exec(stmt, args...)
	return err
}