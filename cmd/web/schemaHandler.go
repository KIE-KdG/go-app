package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// Constants to avoid duplication
const (
	contentTypeHeader = "Content-Type"
	applicationJSON   = "application/json"
)

// TableColumn represents a column in a database table
type TableColumn struct {
	ColumnID          string `json:"column_id"`
	ColumnName        string `json:"column_name"`
	ColumnDatatype    string `json:"column_datatype"`
	ColumnExplanation string `json:"column_explanation"`
}

// TableInfo represents detailed information about a database table
type TableInfo struct {
	TableID          string        `json:"table_id"`
	TableName        string        `json:"table_name"`
	TableDescription string        `json:"table_description"`
	Columns          []TableColumn `json:"columns"`
}

// getSchemaTablesAPI fetches tables for a specific schema from external API
func (app *application) getSchemaTablesAPI(w http.ResponseWriter, r *http.Request) {
	// Get schema name from URL
	params := httprouter.ParamsFromContext(r.Context())
	schemaName := params.ByName("schema_name")

	// Validate input
	if schemaName == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	// Get the project database
	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		app.errorLog.Printf("Invalid project ID: %v", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Find the project database
	projectDatabase, err := app.projectDatabase.GetByProjectID(projectID)
	if err != nil {
		app.errorLog.Printf("Failed to get project database: %v", err)
		app.serverError(w, err)
		return
	}

	// Fetch schema ID using the schema name and database ID
	schemaID, err := app.schemas.GetSchemaIDByName(schemaName, projectDatabase.ID)
	if err != nil {
		app.errorLog.Printf("Failed to get schema ID for schema %s: %v", schemaName, err)
		app.serverError(w, err)
		return
	}

	// Use the external API client to get tables for the schema
	tables, err := app.externalAPI.GetSchemaTables(schemaID)
	if err != nil {
		app.errorLog.Printf("Error fetching tables for schema %s: %v", schemaName, err)
		app.serverError(w, fmt.Errorf("error fetching tables: %w", err))
		return
	}

	// Return JSON response
	w.Header().Set(contentTypeHeader, applicationJSON)
	err = json.NewEncoder(w).Encode(tables)
	if err != nil {
		app.serverError(w, err)
	}
}

// Handler for selected tables
func (app *application) saveProjectTables(w http.ResponseWriter, r *http.Request) {
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var requestData struct {
		ProjectID string `json:"project_id"`
		Tables    []struct {
			TableID    string `json:"table_id"`
			SchemaName string `json:"schema_name"`
			TableName  string `json:"table_name"`
			Columns    []struct {
				ColumnID   string `json:"column_id"`
				ColumnName string `json:"column_name"`
			} `json:"columns"`
		} `json:"tables"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Parse project ID
	_, err = uuid.Parse(requestData.ProjectID)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// For now, just return success
	// In a production app, you'd forward this to your external API
	
	// Build success response
	responseData := map[string]interface{}{
		"status":  "success",
		"message": "Selected tables saved successfully",
	}

	// Return JSON response
	w.Header().Set(contentTypeHeader, applicationJSON)
	json.NewEncoder(w).Encode(responseData)
}