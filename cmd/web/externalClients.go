package main

import (
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
	Name []string `json:"name"`
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
func (c *ExternalAPIClient) CreateDatabaseSchema(dbID uuid.UUID) (*SchemaResponse, error) {
	// Prepare empty request payload (as per original implementation)
	reqData := SchemaRequest{}

	// Create API request
	apiReq := APIRequest{
		Method:      http.MethodPost,
		URL:         c.buildURL("/api/databases/%s/schemas", dbID),
		RequestBody: reqData,
	}

	// Send request and parse response
	var response SchemaResponse
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *ExternalAPIClient) GetDatabaseSchemas(dbID uuid.UUID) (*SrcSchemaResponse, error) {
	apiReq := APIRequest{
		Method: http.MethodGet,
		URL:    c.buildURL("/api/databases/%s/src-schemas", dbID),
	}

	var response SrcSchemaResponse
	err := c.sendAPIRequest(apiReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
