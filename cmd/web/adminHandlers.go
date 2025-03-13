package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kdg/be/lab/internal/validator"
	"net/http"


	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type adminPanelForm struct {
	Chunk               bool   `form:"chunk"`
	ChunkMethod         string `form:"chunkMethod"`
	ChunkCount          string `form:"chunkCount"`
	validator.Validator `form:"-"`
}

// File upload metadata structure that matches the expected format from the client
type FileUploadMetadata struct {
	Name            string   `json:"name"`
	Roles           []string `json:"roles"`
	StorageLocation string   `json:"storage_location"`
	ProjectID       string   `json:"project_id"`
	OwnerID         string   `json:"owner_id,omitempty"` // Added server-side
	CSRFToken       string   `json:"csrf_token"`
}

// Response from the external service
type FileUploadResponse struct {
	Status   string          `json:"status"`
	Message  string          `json:"message,omitempty"`
	Error    string          `json:"error,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Step     string          `json:"step,omitempty"`
}

func (app *application) adminPanel(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = adminPanelForm{}
	
	// Fetch projects for the dropdown
	projects, err := app.projects.GetAll()
	if err != nil {
		app.serverError(w, err)
		return
	}
	data.Projects = projects
	
	app.render(w, http.StatusOK, "admin.tmpl.html", data)
}

// WebSocket handler for file uploads
func (app *application) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated
	if !app.isAuthenticated(r) {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.errorLog.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()
	
	// Connect to external WebSocket service
	otherServer := "ws://localhost" + app.chatPort.Port + "/ws/upload"
	externalWS, _, err := websocket.DefaultDialer.Dial(otherServer, nil)
	if err != nil {
		app.errorLog.Printf("Failed to connect to external service: %v", err)
		sendError(ws, "Failed to connect to document processing service")
		return
	}
	defer externalWS.Close()
	
	app.infoLog.Println("WebSocket connection established for file upload")
	
	// First message should be metadata
	_, metadata, err := ws.ReadMessage()
	if err != nil {
		app.errorLog.Printf("Error reading metadata: %v", err)
		return
	}
	
	// Parse metadata
	var fileMetadata FileUploadMetadata
	if err := json.Unmarshal(metadata, &fileMetadata); err != nil {
		app.errorLog.Printf("Invalid metadata format: %v", err)
		sendError(ws, "Invalid metadata format")
		return
	}
	
	// Verify CSRF token
	if !app.validateCSRFToken(r, fileMetadata.CSRFToken) {
		app.errorLog.Printf("Invalid CSRF token")
		sendError(ws, "Invalid CSRF token")
		return
	}
	
	// Add the user ID to metadata
	fileMetadata.OwnerID = userID.String()
	
	// Remove CSRF token before forwarding (security measure)
	fileMetadata.CSRFToken = ""
	
	// Forward file chunks to external service
	app.forwardFileUpload(ws, externalWS, fileMetadata)
}

// Forward file upload from client to external service
func (app *application) forwardFileUpload(clientWS, externalWS *websocket.Conn, metadata FileUploadMetadata) {
	// Send metadata to external service
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		app.errorLog.Printf("Error marshaling metadata: %v", err)
		sendError(clientWS, "Internal server error")
		return
	}
	
	// Buffer to accumulate file chunks
	var fileBuffer bytes.Buffer
	var chunkCount int
	
	// Receive file chunks from client and forward to external service
	for {
		messageType, message, err := clientWS.ReadMessage()
		if err != nil {
			app.errorLog.Printf("Error reading from client: %v", err)
			return
		}
		
		// Check if we've received the "EOF" marker
		if messageType == websocket.BinaryMessage && bytes.Equal(message, []byte("EOF")) {
			break
		}
		
		// Add chunk to buffer
		if messageType == websocket.BinaryMessage {
			fileBuffer.Write(message)
			chunkCount++
			
			// Send progress updates back to client
			progressMsg := map[string]interface{}{
				"status":      "uploading",
				"chunk_count": chunkCount,
				"message":     fmt.Sprintf("Received chunk %d (%d bytes)", chunkCount, len(message)),
			}
			if err := sendJSON(clientWS, progressMsg); err != nil {
				app.errorLog.Printf("Error sending progress update: %v", err)
			}
		}
	}
	
	// Now we have the complete file in fileBuffer
	app.infoLog.Printf("Received complete file (%d bytes) in %d chunks", fileBuffer.Len(), chunkCount)
	
	// Send the file to external service in chunks
	fileBytes := fileBuffer.Bytes()
	chunkSize := 64 * 1024 // 64KB chunks
	
	// Progress tracker
	totalChunks := (fileBuffer.Len() + chunkSize - 1) / chunkSize
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > fileBuffer.Len() {
			end = fileBuffer.Len()
		}
		
		// Send chunk to external service
		if err := externalWS.WriteMessage(websocket.BinaryMessage, fileBytes[start:end]); err != nil {
			app.errorLog.Printf("Error sending chunk to external service: %v", err)
			sendError(clientWS, "Error sending file to processing service")
			return
		}
		
		// Send progress to client
		progress := int((float64(i+1) / float64(totalChunks)) * 100)
		progressMsg := map[string]interface{}{
			"status":   "forwarding",
			"progress": progress,
			"message":  fmt.Sprintf("Forwarding to processing service: %d%%", progress),
		}
		if err := sendJSON(clientWS, progressMsg); err != nil {
			app.errorLog.Printf("Error sending progress update: %v", err)
		}
	}
	
	// Send EOF marker to external service
	if err := externalWS.WriteMessage(websocket.BinaryMessage, []byte("EOF")); err != nil {
		app.errorLog.Printf("Error sending EOF to external service: %v", err)
		sendError(clientWS, "Error finalizing upload")
		return
	}
	
	// Send metadata to external service
	if err := externalWS.WriteMessage(websocket.TextMessage, metadataBytes); err != nil {
		app.errorLog.Printf("Error sending metadata to external service: %v", err)
		sendError(clientWS, "Error sending file metadata")
		return
	}
	
	// Forward responses from external service back to client
	for {
		_, message, err := externalWS.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				app.infoLog.Println("External service closed connection normally")
			} else {
				app.errorLog.Printf("Error reading from external service: %v", err)
			}
			return
		}
		
		// Forward the message to the client
		if err := clientWS.WriteMessage(websocket.TextMessage, message); err != nil {
			app.errorLog.Printf("Error sending message to client: %v", err)
			return
		}
	}
}

// Handler for document processing
func (app *application) handleDocumentProcessing(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL
	projectID := r.PathValue("id")
	if projectID == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	// Verify user is authenticated
	if !app.isAuthenticated(r) {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.errorLog.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()
	
	// Connect to external WebSocket service
	externalWS, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://external-api-host/ws/embed_chunks/%s", projectID), nil)
	if err != nil {
		app.errorLog.Printf("Failed to connect to external service: %v", err)
		sendError(ws, "Failed to connect to document processing service")
		return
	}
	defer externalWS.Close()
	
	app.infoLog.Printf("Processing documents for project: %s", projectID)
	
	// Forward messages from external service to client
	for {
		_, message, err := externalWS.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				app.infoLog.Println("External service closed connection normally")
			} else {
				app.errorLog.Printf("Error reading from external service: %v", err)
			}
			return
		}
		
		// Forward the message to the client
		if err := ws.WriteMessage(websocket.TextMessage, message); err != nil {
			app.errorLog.Printf("Error sending message to client: %v", err)
			return
		}
	}
}

// Helper function to send error response
func sendError(ws *websocket.Conn, message string) {
	errMsg := map[string]string{
		"status": "error",
		"error":  message,
	}
	sendJSON(ws, errMsg)
}

// Helper function to send JSON response
func sendJSON(ws *websocket.Conn, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.TextMessage, jsonBytes)
}

// Validate CSRF token
func (app *application) validateCSRFToken(r *http.Request, token string) bool {
	// In a real application, you would validate the token against the session
	// This is a simplified implementation
	return token != ""
}