package models

import (
    "database/sql"
    "errors"
    "time"

    "github.com/google/uuid"
)

// File represents a document uploaded to the system
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

// Insert adds a new file record to the database
func (m *FileModel) Insert(file *File) error {
    // If no ID is provided, generate one
    if file.ID == uuid.Nil {
        file.ID = uuid.New()
    }

    stmt := `
        INSERT INTO files (
            id, name, description, file_path, mime_type, size, 
            role, storage_location, project_id, user_id, 
            uploaded_at, status
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
    `

    _, err := m.DB.Exec(
        stmt,
        file.ID,
        file.Name,
        file.Description,
        file.FilePath,
        file.MimeType,
        file.Size,
        file.Role,
        file.StorageLocation,
        file.ProjectID,
        file.UserID,
        file.Status,
    )

    return err
}

// Get retrieves a file by its ID
func (m *FileModel) Get(id uuid.UUID) (*File, error) {
    stmt := `
        SELECT id, name, description, file_path, mime_type, size,
               role, storage_location, project_id, user_id,
               uploaded_at, processed_at, status
        FROM files
        WHERE id = ?
    `

    var file File
    err := m.DB.QueryRow(stmt, id).Scan(
        &file.ID,
        &file.Name,
        &file.Description,
        &file.FilePath,
        &file.MimeType,
        &file.Size,
        &file.Role,
        &file.StorageLocation,
        &file.ProjectID,
        &file.UserID,
        &file.UploadedAt,
        &file.ProcessedAt,
        &file.Status,
    )

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNoRecord
        }
        return nil, err
    }

    return &file, nil
}

// GetByProject retrieves all files for a specific project
func (m *FileModel) GetByProject(projectID uuid.UUID) ([]*File, error) {
    stmt := `
        SELECT id, name, description, file_path, mime_type, size,
               role, storage_location, project_id, user_id,
               uploaded_at, processed_at, status
        FROM files
        WHERE project_id = ?
        ORDER BY uploaded_at DESC
    `

    rows, err := m.DB.Query(stmt, projectID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    files := []*File{}

    for rows.Next() {
        var file File
        err := rows.Scan(
            &file.ID,
            &file.Name,
            &file.Description,
            &file.FilePath,
            &file.MimeType,
            &file.Size,
            &file.Role,
            &file.StorageLocation,
            &file.ProjectID,
            &file.UserID,
            &file.UploadedAt,
            &file.ProcessedAt,
            &file.Status,
        )
        if err != nil {
            return nil, err
        }

        files = append(files, &file)
    }

    if err = rows.Err(); err != nil {
        return nil, err
    }

    return files, nil
}

// UpdateStatus updates the status of a file
func (m *FileModel) UpdateStatus(id uuid.UUID, status string) error {
    var stmt string
    var args []interface{}

    if status == "processed" {
        stmt = `
            UPDATE files
            SET status = ?, processed_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
        args = []interface{}{status, id}
    } else {
        stmt = `
            UPDATE files
            SET status = ?
            WHERE id = ?
        `
        args = []interface{}{status, id}
    }

    _, err := m.DB.Exec(stmt, args...)
    return err
}