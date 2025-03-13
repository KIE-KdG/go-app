package models

import (
    "database/sql"
    "errors"
    "time"

    "github.com/google/uuid"
)

// Project represents a container for documents and other resources
type Project struct {
    ID          uuid.UUID
    Name        string
    Description string
    UserID      uuid.UUID
    ExternalID   string
    Created     time.Time
    Updated     time.Time
    DocumentCount int // Used for UI display
}

type ProjectModel struct {
    DB *sql.DB
}

// Insert creates a new project for a user
func (m *ProjectModel) Insert(name, description string, userID uuid.UUID) (uuid.UUID, error) {
    // Generate a new UUID for the project
    projectID := uuid.New()

    stmt := `
        INSERT INTO projects (id, name, description, user_id, created, updated)
        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    _, err := m.DB.Exec(stmt, projectID, name, description, userID)
    if err != nil {
        return uuid.Nil, err
    }

    return projectID, nil
}

// Get retrieves a project by its ID
func (m *ProjectModel) Get(id uuid.UUID) (*Project, error) {
    stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.created, p.updated,
               (SELECT COUNT(*) FROM files WHERE project_id = p.id) AS document_count
        FROM projects p
        WHERE p.id = ?
    `
    
    var project Project
    err := m.DB.QueryRow(stmt, id).Scan(
        &project.ID,
        &project.Name,
        &project.Description,
        &project.UserID,
        &project.Created,
        &project.Updated,
        &project.DocumentCount,
    )
    
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNoRecord
        }
        return nil, err
    }
    
    return &project, nil
}

// GetAll retrieves all projects
func (m *ProjectModel) GetAll() ([]*Project, error) {
    stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.created, p.updated,
               (SELECT COUNT(*) FROM files WHERE project_id = p.id) AS document_count
        FROM projects p
        ORDER BY p.updated DESC
    `
    
    rows, err := m.DB.Query(stmt)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    projects := []*Project{}
    
    for rows.Next() {
        var project Project
        err := rows.Scan(
            &project.ID,
            &project.Name,
            &project.Description,
            &project.UserID,
            &project.Created,
            &project.Updated,
            &project.DocumentCount,
        )
        if err != nil {
            return nil, err
        }
        
        projects = append(projects, &project)
    }
    
    if err = rows.Err(); err != nil {
        return nil, err
    }
    
    return projects, nil
}

// GetByUserID retrieves all projects for a specific user
func (m *ProjectModel) GetByUserID(userID uuid.UUID) ([]*Project, error) {
    stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.created, p.updated,
               (SELECT COUNT(*) FROM files WHERE project_id = p.id) AS document_count
        FROM projects p
        WHERE p.user_id = ?
        ORDER BY p.updated DESC
    `
    
    rows, err := m.DB.Query(stmt, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    projects := []*Project{}
    
    for rows.Next() {
        var project Project
        err := rows.Scan(
            &project.ID,
            &project.Name,
            &project.Description,
            &project.UserID,
            &project.Created,
            &project.Updated,
            &project.DocumentCount,
        )
        if err != nil {
            return nil, err
        }
        
        projects = append(projects, &project)
    }
    
    if err = rows.Err(); err != nil {
        return nil, err
    }
    
    return projects, nil
}

// Update modifies an existing project
func (m *ProjectModel) Update(id uuid.UUID, name, description string) error {
    stmt := `
        UPDATE projects
        SET name = ?, description = ?, updated = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    
    _, err := m.DB.Exec(stmt, name, description, id)
    if err != nil {
        return err
    }
    
    return nil
}

// Delete removes a project and its associated data
func (m *ProjectModel) Delete(id uuid.UUID) error {
    // Begin a transaction to ensure all related data is deleted
    tx, err := m.DB.Begin()
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            tx.Rollback()
            return
        }
        err = tx.Commit()
    }()
    
    // First delete files (if we have a files table)
    _, err = tx.Exec("DELETE FROM files WHERE project_id = ?", id)
    if err != nil {
        return err
    }
    
    // Then delete the project
    _, err = tx.Exec("DELETE FROM projects WHERE id = ?", id)
    if err != nil {
        return err
    }
    
    return nil
}

func (m *ProjectModel) UpdateExternalID(id uuid.UUID, externalID string) error {
    stmt := `
        UPDATE projects
        SET external_id = ?
        WHERE id = ?
    `
    
    _, err := m.DB.Exec(stmt, externalID, id)
    return err
}