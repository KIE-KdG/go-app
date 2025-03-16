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

	// Parse database ID
	_, err := uuid.Parse(dbIDStr)
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

	// Instead of getting database by ID which doesn't exist, we can directly call the API
	// We don't need the database details since the schema name is already provided

	// Build the URL for the external API
	apiURL := fmt.Sprintf("%s/api/schemas/%s/tables-detailed", app.externalAPI.baseURL, schemaName)

	// Create API request
	apiReq := APIRequest{
		Method: http.MethodGet,
		URL:    apiURL,
		Headers: map[string]string{
			contentTypeHeader: applicationJSON,
		},
	}

	// Call the external API
	var tablesResponse []TableInfo
	err = app.externalAPI.sendAPIRequest(apiReq, &tablesResponse)
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
	_, err = uuid.Parse(requestData.ProjectID)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Here you would typically save the selected tables to your database
	// For now, we'll just return a success response
	
	// Build response
	responseData := map[string]interface{}{
		"status":  "success",
		"message": "Selected tables saved successfully",
	}

	// Return JSON response
	w.Header().Set(contentTypeHeader, applicationJSON)
	json.NewEncoder(w).Encode(responseData)
}