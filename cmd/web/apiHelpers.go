package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// APIRequest represents a generic API request
type APIRequest struct {
	Method      string      // HTTP method (GET, POST, etc.)
	URL         string      // Full URL for the request
	RequestBody interface{} // Request payload (will be marshaled to JSON)
	Headers     map[string]string // Custom headers
}

// sendAPIRequest is a generic function to send requests to external APIs
func (c *ExternalAPIClient) sendAPIRequest(req APIRequest, responsePtr interface{}) error {
	// Marshal request body if provided
	var bodyBytes []byte
	var err error
	
	if req.RequestBody != nil {
		bodyBytes, err = json.Marshal(req.RequestBody)
		if err != nil {
			return fmt.Errorf("error marshaling request: %w", err)
		}
	}

	// Create the request
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Set any custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Send the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.handleErrorResponse(resp)
	}

	// Parse response if a pointer was provided
	if responsePtr != nil {
		if err := json.NewDecoder(resp.Body).Decode(responsePtr); err != nil {
			return fmt.Errorf("error parsing response: %w", err)
		}
	}

	return nil
}

// handleErrorResponse extracts error details from an API error response
func (c *ExternalAPIClient) handleErrorResponse(resp *http.Response) error {
	var errorResp struct {
		Detail string `json:"detail"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if errorResp.Detail != "" {
		return fmt.Errorf("API error: %s", errorResp.Detail)
	}

	return fmt.Errorf("API returned error status: %d", resp.StatusCode)
}

// buildURL constructs a URL with the base URL and path components
func (c *ExternalAPIClient) buildURL(pathFormat string, args ...interface{}) string {
	path := fmt.Sprintf(pathFormat, args...)
	return fmt.Sprintf("%s%s", c.baseURL, path)
}