package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
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
func (c *ExternalAPIClient) CreateExternalProject(userID string, name string) (*ProjectResponse, error) {
    // Prepare the request payload
    reqData := ProjectRequest{
        Name: name,
    }
    
    reqBody, err := json.Marshal(reqData)
    if err != nil {
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }
    
    // Build the URL
    url := fmt.Sprintf("%s/api/users/%s/projects", c.baseURL, userID)
    
    // Create the request
    req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }
    
    // Set headers
    req.Header.Set("Content-Type", "application/json")
    
    // Send the request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error sending request: %w", err)
    }
    defer resp.Body.Close()
    
    // Check response status
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        var errorResp struct {
            Detail string `json:"detail"`
        }
        
        if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
            return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
        }
        
        return nil, fmt.Errorf("API error: %s", errorResp.Detail)
    }
    
    // Parse response
    var projectResp ProjectResponse
    if err := json.NewDecoder(resp.Body).Decode(&projectResp); err != nil {
        return nil, fmt.Errorf("error parsing response: %w", err)
    }
    
    return &projectResp, nil
}