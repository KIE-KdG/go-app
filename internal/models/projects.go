package models

import (
	"database/sql"
	"time"
)

// Project represents a project with an optional creation date.
type Project struct {
    ID           string     `db:"id" json:"id"`
    Name         string     `db:"name" json:"name"`
    CreationDate *time.Time `db:"creation_date" json:"creation_date"`
}

// UserMetadata represents metadata for a user.
type UserMetadata struct {
    ID   string `db:"id" json:"id"`
    Name string `db:"name" json:"name"`
    Role string `db:"role" json:"role"`
}

// DatabaseMetadata contains metadata about a database connection.
type DatabaseMetadata struct {
    ID               string `db:"id" json:"id"`
    SourceConnString string `db:"source_conn_string" json:"source_conn_string"`
    DBType           string `db:"db_type" json:"db_type"`
    ProjectID        string `db:"project_id" json:"project_id"`
}

// Rule represents a rule with an optional integer ID.
type Rule struct {
    ID          *int   `db:"id" json:"id"`
    Description string `db:"description" json:"description"`
    RuleType    string `db:"rule_type" json:"rule_type"`
    ProjectID   string `db:"project_id" json:"project_id"`
}

// TableMetadata represents metadata for a table.
type TableMetadata struct {
    ID          string `db:"id" json:"id"`
    Schema      string `db:"schema" json:"schema"`
    TableName   string `db:"table_name" json:"table_name"`
    Description string `db:"description" json:"description"`
    DatabaseID  string `db:"database_id" json:"database_id"`
}

// ColumnMetadata represents metadata for a column.
type ColumnMetadata struct {
    ID          string `db:"id" json:"id"`
    Name        string `db:"name" json:"name"`
    DataType    string `db:"datatype" json:"datatype"`
    Description string `db:"description" json:"description"`
    TableID     string `db:"table_id" json:"table_id"`
}

type ProjectModel struct {
		DB *sql.DB
}

func (m *ProjectModel) Insert(name string) error {
	stmt := `INSERT INTO projects (name, creation_date) VALUES(?, datetime('now'))`

	statement, err := m.DB.Prepare(stmt)
	if err != nil {
		return err
	}

	_, err = statement.Exec(name)
	if err != nil {
		return err
	}

	return nil
}

func (m *ProjectModel) Get(id string) (*Project, error) {
	stmt := `SELECT id, name, creation_date FROM projects WHERE id = ?`

	row := m.DB.QueryRow(stmt, id)

	project := &Project{}

	err := row.Scan(&project.ID, &project.Name, &project.CreationDate)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (m *ProjectModel) GetAll() ([]*Project, error) {
	stmt := `SELECT id, name, creation_date FROM projects`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []*Project{}

	for rows.Next() {
		project := &Project{}

		err := rows.Scan(&project.ID, &project.Name, &project.CreationDate)
		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	return projects, nil
}