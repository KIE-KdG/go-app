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
	// Get database ID and schema name from URL
	params := httprouter.ParamsFromContext(r.Context())
	dbIDStr := params.ByName("db_id")
	schemaName := params.ByName("schema_name")

	// Validate inputs
	if dbIDStr == "" || schemaName == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Parse schema ID
	schemaID, err := uuid.Parse(schemaName)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	// Use our ExternalAPIClient to get the schema tables
	tablesResponse, err := app.externalAPI.GetSchemaTables(schemaID)
	if err != nil {
		app.errorLog.Printf("Error fetching tables for schema %s: %v", schemaName, err)
		app.serverError(w, fmt.Errorf("error fetching tables: %w", err))
		return
	}

	// Return JSON response
	w.Header().Set(contentTypeHeader, applicationJSON)
	json.NewEncoder(w).Encode(tablesResponse)
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
	projectID, err := uuid.Parse(requestData.ProjectID)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Convert tables to the format expected by SaveSelectedTables
	selectedTables := make([]map[string]interface{}, len(requestData.Tables))
	for i, table := range requestData.Tables {
		selectedTables[i] = map[string]interface{}{
			"table_id":     table.TableID,
			"schema_name":  table.SchemaName,
			"table_name":   table.TableName,
			"columns":      table.Columns,
		}
	}

	// Use our ExternalAPIClient to save the selected tables
	err = app.externalAPI.SaveSelectedTables(projectID, selectedTables)
	if err != nil {
		app.errorLog.Printf("Error saving selected tables: %v", err)
		app.serverError(w, fmt.Errorf("error saving tables: %w", err))
		return
	}
	
	// Build success response
	responseData := map[string]interface{}{
		"status":  "success",
		"message": "Selected tables saved successfully",
	}

	// Return JSON response
	w.Header().Set(contentTypeHeader, applicationJSON)
	json.NewEncoder(w).Encode(responseData)
}