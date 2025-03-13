// models/projects.go
package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID             uuid.UUID
	Name           string
	Description    string
	UserID         uuid.UUID
	ExternalID     string
	Created        time.Time
	Updated        time.Time
	DocumentCount  int
}

type ProjectModel struct {
	DB *sql.DB
}

func NewProjectModel(db *sql.DB) *ProjectModel {
	return &ProjectModel{DB: db}
}

func (m *ProjectModel) Insert(name, description string, userID uuid.UUID) (uuid.UUID, error) {
	var projectID uuid.UUID
	
	stmt := `
        INSERT INTO projects (name, description, user_id, created, updated)
        VALUES ($1, $2, $3, NOW(), NOW())
        RETURNING id
    `
	
	err := m.DB.QueryRow(stmt, name, description, userID).Scan(&projectID)
	if err != nil {
		return uuid.Nil, err
	}
	
	// After creating the project, link it to the user in users_projects
	linkStmt := `
        INSERT INTO users_projects (user_id, project_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `
	
	_, err = m.DB.Exec(linkStmt, userID, projectID)
	if err != nil {
		// Log but don't fail if linking fails
		return projectID, err
	}
	
	return projectID, nil
}

func (m *ProjectModel) Get(id uuid.UUID) (*Project, error) {
	stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.external_id, 
               p.created, p.updated,
               (SELECT COUNT(*) FROM files_projects fp WHERE fp.project_id = p.id) AS document_count
        FROM projects p
        WHERE p.id = $1
    `
	
	var project Project
	var externalID sql.NullString
	
	err := m.DB.QueryRow(stmt, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.UserID,
		&externalID,
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
	
	if externalID.Valid {
		project.ExternalID = externalID.String
	}
	
	return &project, nil
}

func (m *ProjectModel) GetByUserID(userID uuid.UUID) ([]*Project, error) {
	stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.external_id, 
               p.created, p.updated,
               (SELECT COUNT(*) FROM files_projects fp WHERE fp.project_id = p.id) AS document_count
        FROM projects p
        JOIN users_projects up ON p.id = up.project_id
        WHERE up.user_id = $1
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
		var externalID sql.NullString
		
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.UserID,
			&externalID,
			&project.Created,
			&project.Updated,
			&project.DocumentCount,
		)
		if err != nil {
			return nil, err
		}
		
		if externalID.Valid {
			project.ExternalID = externalID.String
		}
		
		projects = append(projects, &project)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return projects, nil
}

func (m *ProjectModel) UpdateExternalID(id uuid.UUID, externalID string) error {
	stmt := `
        UPDATE projects
        SET external_id = $1, updated = NOW()
        WHERE id = $2
    `
	
	_, err := m.DB.Exec(stmt, externalID, id)
	return err
}

func (m *ProjectModel) GetAll() ([]*Project, error) {
	stmt := `
        SELECT p.id, p.name, p.description, p.user_id, p.external_id, 
               p.created, p.updated,
               (SELECT COUNT(*) FROM files_projects fp WHERE fp.project_id = p.id) AS document_count
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
		var externalID sql.NullString
		
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.UserID,
			&externalID,
			&project.Created,
			&project.Updated,
			&project.DocumentCount,
		)
		if err != nil {
			return nil, err
		}
		
		if externalID.Valid {
			project.ExternalID = externalID.String
		}
		
		projects = append(projects, &project)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return projects, nil
}