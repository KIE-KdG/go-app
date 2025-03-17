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
	// Get schema ID from URL
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
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Find the project database
	projectDatabase, err := app.projectDatabase.GetByProjectID(projectID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// We need to get the tables from the schema. First, need to check if this schema exists
	// in the database at all. We shouldn't create schemas that don't exist.
	
	// Get all available schemas in the database
	availableSchemas, err := app.externalAPI.GetDatabaseSchemas(projectDatabase.ID)
	if err != nil {
		app.serverError(w, fmt.Errorf("failed to retrieve schemas: %w", err))
		return
	}
	
	// Check if the requested schema exists in the available schemas
	schemaExists := false
	for _, schema := range *availableSchemas {
		if schema == schemaName {
			schemaExists = true
			break
		}
	}
	
	if !schemaExists {
		app.clientError(w, http.StatusNotFound)
		return
	}
	
	// Now we know the schema exists in the database, so we can try to get tables
	var schemaID uuid.UUID
	
	// If we have the schema in our system already, get its ID
	schemaID, err = app.schemas.GetSchemaIDByName(schemaName, projectDatabase.ID)
	if err != nil {
		// If schema reference doesn't exist in our system yet, we need its ID from the external API
		// This is just to get a reference to the schema, not to create it
		// We're using an existing schema from the database
		schemaResponse, err := app.externalAPI.CreateDatabaseSchema(projectDatabase.ID, []string{schemaName})
		if err != nil {
			app.serverError(w, fmt.Errorf("failed to reference schema: %w", err))
			return
		}
		
		schemaID = schemaResponse.SchemaID
	}

	// Use the external API client to get tables for the schema
	tables, err := app.externalAPI.GetSchemaTables(schemaID)
	if err != nil {
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

// saveProjectTables handles saving the selected tables and columns for a project
func (app *application) saveProjectTables(w http.ResponseWriter, r *http.Request) {
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	// Define the request structure to parse the JSON request
	type columnSelection struct {
		ColumnID   string `json:"column_id"`
		ColumnName string `json:"column_name"`
	}

	type tableSelection struct {
		TableID    string            `json:"table_id"`
		SchemaName string            `json:"schema_name"`
		TableName  string            `json:"table_name"`
		Columns    []columnSelection `json:"columns"`
	}

	type requestData struct {
		ProjectID string          `json:"project_id"`
		Tables    []tableSelection `json:"tables"`
	}

	// Parse request body
	var reqData requestData
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Parse project ID
	projectID, err := uuid.Parse(reqData.ProjectID)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Verify project exists and belongs to the user
	project, err := app.projects.Get(projectID)
	if err != nil {
		app.notFound(w)
		return
	}
	
	if project.UserID != userID {
		app.clientError(w, http.StatusForbidden)
		return
	}

	// Convert to the format expected by the external API
	var selectedTables []map[string]interface{}
	for _, table := range reqData.Tables {
		tableObj := map[string]interface{}{
			"table_id":    table.TableID,
			"schema_name": table.SchemaName,
			"table_name":  table.TableName,
			"columns":     table.Columns,
		}
		selectedTables = append(selectedTables, tableObj)
	}

	// Forward to the external API
	err = app.externalAPI.SaveSelectedTables(projectID, selectedTables)
	if err != nil {
		app.serverError(w, fmt.Errorf("error saving tables: %w", err))
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Selected tables saved successfully",
	}

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}