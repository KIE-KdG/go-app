package models

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type ProjectDatabase struct {
	ID               uuid.UUID
	ConnectionString string
	DbType           string
}
type ProjectDatabaseModel struct {
	DB *sql.DB
}

func NewProjectDatabaseModel(db *sql.DB) *ProjectDatabaseModel {
	return &ProjectDatabaseModel{DB: db}
}

func (m *ProjectDatabaseModel) GetByProjectID(id uuid.UUID) (*ProjectDatabase, error) {
	stmt := `
	SELECT 
	d.id as database_id, 
	d.source_conn_string, 
	d.db_type 
	FROM projects as p 
	JOIN databases as d 
	ON p.id = d. project_id 
	WHERE p.id = $1
	`

	var projectDatabase ProjectDatabase

	err := m.DB.QueryRow(stmt, id).Scan(
		&projectDatabase.ID,
		&projectDatabase.ConnectionString,
		&projectDatabase.DbType,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}
	
	return &projectDatabase, nil
}
