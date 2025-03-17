package models

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// Schema represents a database schema
type Schema struct {
	ID         uuid.UUID
	Name       string
	DatabaseID uuid.UUID
}

// SchemaModel handles database operations for schemas
type SchemaModel struct {
	DB *sql.DB
}

// NewSchemaModel creates a new SchemaModel
func NewSchemaModel(db *sql.DB) *SchemaModel {
	return &SchemaModel{DB: db}
}

// GetSchemaIDByName retrieves the schema ID for a given schema name
func (m *SchemaModel) GetSchemaIDByName(schemaName string, databaseID uuid.UUID) (uuid.UUID, error) {
	stmt := `
		SELECT id 
		FROM schemas 
		WHERE name = $1 AND database_id = $2 
		LIMIT 1
	`

	var schemaID uuid.UUID
	err := m.DB.QueryRow(stmt, schemaName, databaseID).Scan(&schemaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrNoRecord
		}
		return uuid.Nil, err
	}

	return schemaID, nil
}

// GetSchemaByName retrieves full schema details by name
func (m *SchemaModel) GetSchemaByName(schemaName string, databaseID uuid.UUID) (*Schema, error) {
	stmt := `
		SELECT id, name, database_id 
		FROM schemas 
		WHERE name = $1 AND database_id = $2 
		LIMIT 1
	`

	var schema Schema
	err := m.DB.QueryRow(stmt, schemaName, databaseID).Scan(
		&schema.ID, 
		&schema.Name, 
		&schema.DatabaseID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return &schema, nil
}

// ListSchemasByDatabaseID retrieves all schemas for a given database ID
func (m *SchemaModel) ListSchemasByDatabaseID(databaseID uuid.UUID) ([]Schema, error) {
	stmt := `
		SELECT id, name, database_id 
		FROM schemas 
		WHERE database_id = $1
		ORDER BY name
	`

	rows, err := m.DB.Query(stmt, databaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []Schema
	for rows.Next() {
		var schema Schema
		err := rows.Scan(
			&schema.ID, 
			&schema.Name, 
			&schema.DatabaseID,
		)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}