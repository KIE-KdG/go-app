package main

import (
	"encoding/json"
	"fmt"
	"kdg/be/lab/internal/validator"
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

type schemaCreate struct {
	DbID                string   `form:"db_id"`
	ProjectID           string   `form:"project_id"`
	SchemaName          []string `form:"schemaName"`
	validator.Validator `form:"-"`
}

// Schema registration endpoint (saves schema to metadata)
func (app *application) databaseSchemaPost(w http.ResponseWriter, r *http.Request) {
	var schemaForm schemaCreate
	err := app.decodePostForm(r, &schemaForm)
	if err != nil {
		app.errorLog.Printf("Form decode error: %v", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	schemaForm.CheckField(len(schemaForm.SchemaName) > 0, "schemaName", "Please select at least one schema")

	projectID, err := uuid.Parse(schemaForm.ProjectID)
	if err != nil {
		app.errorLog.Printf("Invalid project ID: %v", err)
		schemaForm.AddNonFieldError("Invalid project ID")
		
		// Create proper form structure
		formData := projectForms{
			SchemaForm: schemaSetupForm{
				Name: schemaForm.SchemaName,
				Validator: schemaForm.Validator,
			},
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	// Parse database ID
	dbID, err := uuid.Parse(schemaForm.DbID)
	if err != nil {
		app.errorLog.Printf("Invalid database ID: %v", err)
		schemaForm.AddNonFieldError("Invalid database ID")
		
		// Create proper form structure
		formData := projectForms{
			SchemaForm: schemaSetupForm{
				Name: schemaForm.SchemaName,
				Validator: schemaForm.Validator,
			},
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	if !schemaForm.Valid() {
		// Create proper form structure
		formData := projectForms{
			SchemaForm: schemaSetupForm{
				Name: schemaForm.SchemaName,
				Validator: schemaForm.Validator,
			},
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	// Now call the API with both the database ID and schema name
	_, err = app.externalAPI.CreateDatabaseSchema(dbID, schemaForm.SchemaName)
	if err != nil {
		app.errorLog.Printf("Database schema creation error: %v", err)
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Schema successfully registered in metadata")
	http.Redirect(w, r, fmt.Sprintf("/project/view/%s", projectID), http.StatusSeeOther)
}

// API endpoint to get tables for a specific schema ID
func (app *application) getSchemaTablesHandler(w http.ResponseWriter, r *http.Request) {
	// Get schema ID from URL
	params := httprouter.ParamsFromContext(r.Context())
	schemaIDStr := params.ByName("id")

	// Validate input
	if schemaIDStr == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Parse UUID
	schemaID, err := uuid.Parse(schemaIDStr)
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

	// Call the external API to get tables for this schema
	tables, err := app.externalAPI.GetSchemaTables(schemaID)
	if err != nil {
		app.serverError(w, fmt.Errorf("error fetching tables: %w", err))
		return
	}

	// Return tables as JSON
	w.Header().Set(contentTypeHeader, applicationJSON)
	json.NewEncoder(w).Encode(tables)
}

// API endpoint to save selected tables and columns
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