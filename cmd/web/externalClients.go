package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ExternalAPIClient handles communication with external services
type ExternalAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

// ProjectRequest represents the data sent to create a project in the external API
type ProjectRequest struct {
	Name string `json:"name"`
}

// ProjectResponse represents the response from the external API
type ProjectResponse struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

// ProjectDatabaseRequest represents the data sent to create a project database
type ProjectDatabaseRequest struct {
	ProjectID      uuid.UUID `json:"project_id"`
	DbSourceString string    `json:"source_db_conn"`
	DbType         string    `json:"db_type"`
}

// ProjectDatabaseResponse represents the response from creating a project database
type ProjectDatabaseResponse struct {
	ProjectID      uuid.UUID `json:"project_id"`
	DbSourceString string    `json:"source_db_conn"`
	DbType         string    `json:"db_type"`
}

// SchemaRequest represents the data sent to create a database schema
type SchemaRequest struct {
	Name       string    `json:"name"`
	DatabaseID uuid.UUID `json:"database_id"`
}

// SchemaResponse represents the response from creating a schema
type SchemaResponse struct {
	SchemaName string    `json:"name"`
	SchemaID   uuid.UUID `json:"schema_id"`
}

type SrcSchemaResponse struct {
	Name []string `json:"name,omitempty"`
}

type SchemaExplenationRequest struct {
	ProjectID  uuid.UUID `json:"project_id"`
	SchemaName string    `json:"schema_name"`
}

type SchemaExplenationResponse struct {
	TableID     uuid.UUID `json:"table_id"`
	SchemaName  string    `json:"schema_name"`
	TableName   string    `json:"table_name"`
	Description string    `json:"description"`
}

// DbConnectionRequestModel represents the database connection details
type DbConnectionRequestModel struct {
	SourceDbConn string `json:"source_db_conn"`
	DbType       string `json:"db_type"`
}

// TableExplanationModel represents a table with its explanation
type TableExplanationModel struct {
	SchemaName  string `json:"schema_name"`
	TableName   string `json:"table_name"`
	Explanation string `json:"explanation"`
}

// SchemaExplanationRequest represents the request to create tables in a schema
type SchemaExplanationRequest struct {
	ConnModel    DbConnectionRequestModel `json:"conn_model"`
	SchemaName   string                   `json:"schema_name"`
	ProjectID    uuid.UUID                `json:"project_id"`
	Explanations []TableExplanationModel  `json:"explanations"`
}

// TableExplanationResponse represents the response from creating tables
type TableExplanationResponse struct {
	SchemaName  string    `json:"schema_name"`
	TableID     uuid.UUID `json:"table_id"`
	TableName   string    `json:"table_name"`
	Description string    `json:"description"`
}

// NewExternalAPIClient creates a new API client
func NewExternalAPIClient(baseURL string) *ExternalAPIClient {
	return &ExternalAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateExternalProject sends a project creation request to the external API
func (c *ExternalAPIClient) CreateExternalProject(userID uuid.UUID, name string) (*ProjectResponse, error) {
	// Prepare the request payload
	reqData := ProjectRequest{
		Name: name,
	}

	// Create API request
	apiReq := APIRequest{
		Method:      http.MethodPost,
		URL:         c.buildURL("/api/users/%s/projects", userID),
		RequestBody: reqData,
	}

	// Send request and parse response
	var response ProjectResponse
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CreateProjectDatabase sends a project database creation request to the external API
func (c *ExternalAPIClient) CreateProjectDatabase(projectID uuid.UUID, connString, dbType string) (*ProjectDatabaseResponse, error) {
	// Prepare the request payload
	reqData := ProjectDatabaseRequest{
		ProjectID:      projectID,
		DbSourceString: connString,
		DbType:         dbType,
	}

	// Create API request
	apiReq := APIRequest{
		Method:      http.MethodPost,
		URL:         c.buildURL("/api/projects/%s/databases", projectID),
		RequestBody: reqData,
	}

	// Send request and parse response
	var response ProjectDatabaseResponse
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CreateDatabaseSchema sends a database schema creation request to the external API
func (c *ExternalAPIClient) CreateDatabaseSchema(dbID uuid.UUID, schemaName []string) (*SchemaResponse, error) {
	// The API expects a list of schema requests
	var reqData []SchemaRequest

	for _, schema := range schemaName {
		reqData = append(reqData, SchemaRequest{
			Name: schema,
			// Note: DatabaseID is not needed in the request body since it's in the URL
		})
	}

	// Create API request
	apiReq := APIRequest{
		Method:      http.MethodPost,
		URL:         c.buildURL("/api/databases/%s/schemas", dbID),
		RequestBody: reqData,
	}

	// Send request and parse response
	// The API returns a list of SchemaResponse objects, but we're only sending one
	// schema request, so we'll only get one response
	var responses []SchemaResponse
	err := c.sendAPIRequest(apiReq, &responses)
	if err != nil {
		return nil, err
	}

	// Make sure we got at least one response
	if len(responses) == 0 {
		return nil, fmt.Errorf("no schema response received from API")
	}

	// Return the first (and only) response
	return &responses[0], nil
}

func (c *ExternalAPIClient) GetDatabaseSchemas(dbID uuid.UUID) (*[]string, error) {
	apiReq := APIRequest{
		Method: http.MethodGet,
		URL:    c.buildURL("/api/databases/%s/src-schemas", dbID),
	}

	var response []string
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, err
	}

	fmt.Printf("schemas found for DB: %s, %s", dbID, response)

	return &response, nil
}

func (c *ExternalAPIClient) GetSchemaTables(schemaID string) ([]TableInfo, error) {
	apiReq := APIRequest{
		Method: http.MethodGet,
		URL:    c.buildURL("/api/schemas/%s/tables-detailed", schemaID),
	}

	var response []TableInfo
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, fmt.Errorf("error fetching schema tables: %w", err)
	}

	return response, nil
}

// SaveSelectedTables sends selected tables and columns to the API
func (c *ExternalAPIClient) SaveSelectedTables(projectID uuid.UUID, selectedTables []map[string]interface{}) error {
	reqData := map[string]interface{}{
		"project_id": projectID.String(),
		"tables":     selectedTables,
	}

	apiReq := APIRequest{
		Method:      http.MethodPost,
		URL:         c.buildURL("/api/projects/%s/selected-tables", projectID),
		RequestBody: reqData,
	}

	err := c.sendAPIRequest(apiReq, nil)
	if err != nil {
		return fmt.Errorf("error saving selected tables: %w", err)
	}

	return nil
}