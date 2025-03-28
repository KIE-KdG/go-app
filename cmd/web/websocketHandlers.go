package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

// WebSocketRequest handles incoming requests from clients
type WebSocketRequest struct {
	Question   string `json:"question,omitempty"` // New schema
	Message    string `json:"message,omitempty"`  // For backward compatibility
	DBUsed     bool   `json:"dbUsed"`
	DocsUsed   bool   `json:"docsUsed"`
	DatabaseID string `json:"database_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	ProjectID  string `json:"project_id,omitempty"`
	Interrupt  bool   `json:"interrupt"` // For interrupt functionality
}

// ChatIntermediateResponse for status updates
type ChatIntermediateResponse struct {
	Status string `json:"status"`
}

// ChatFinalResponse for text responses
type ChatFinalResponse struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}

// GeoObject represents a single GeoJSON feature collection
type GeoObject struct {
	Type     string          `json:"type"`
	Features json.RawMessage `json:"features"`
}

// ChatGeoJsonResponse for geographic data
type ChatGeoJsonResponse struct {
	GeoObjects map[string]GeoObject `json:"geo_objects"`
}

// FinalResponse combines all response types for the client
type FinalResponse struct {
	Status      string               `json:"status,omitempty"`
	Answer      string               `json:"answer,omitempty"`      // For backward compatibility
	Response    string               `json:"response,omitempty"`    // New schema
	GeoJSON     json.RawMessage      `json:"geoJSON,omitempty"`     // For backward compatibility
	GeoObjects  map[string]GeoObject `json:"geo_objects,omitempty"` // New schema
	Interrupted bool                 `json:"interrupted,omitempty"`
}

func (app *application) handleConnections(w http.ResponseWriter, r *http.Request) {
	userID := app.userIdFromSession(r)
	params := httprouter.ParamsFromContext(r.Context())
	chatID := params.ByName("id")
	if chatID == "" {
		app.errorLog.Println("chatID not found in URL parameters")
		http.Error(w, "Chat ID not found", http.StatusInternalServerError)
		return
	}



	app.infoLog.Printf("Chat ID: %s", chatID)

	// Proceed with WebSocket upgrade and handling...
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer ws.Close()

	// Interrupt channel to signal when to stop processing
	interrupt := make(chan struct{})
	// Mutex to protect the interrupt channel from concurrent access
	var interruptMutex sync.Mutex
	// Flag to track if we're currently processing a prompt
	var isProcessing bool = false

	// Start a goroutine to handle messages from client
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				app.errorLog.Println("read error:", err)
				break
			}

			req, err := parseWSRequest(msg)
			if err != nil {
				app.errorLog.Println("parse error:", err)
				continue
			}

			// Handle interrupt signal
			if req.Interrupt {
				app.infoLog.Println("Received interrupt signal")
				interruptMutex.Lock()
				if isProcessing {
					// Send interrupt signal by closing the channel
					close(interrupt)
					// Send acknowledgment back to client
					if err := sendWSJSON(ws, FinalResponse{Interrupted: true}); err != nil {
						app.errorLog.Println("error sending interrupt acknowledgment:", err)
					}
				}
				interruptMutex.Unlock()
				continue // Skip the rest of the loop for interrupts
			}

			// Get message from either Question or Message field for backward compatibility
			message := req.Question
			if message == "" {
				message = req.Message
			}

			// Regular message handling
			app.infoLog.Printf("Received question: %s, DB: %t, Docs: %t", message, req.DBUsed, req.DocsUsed)

			chatUUID, ok := app.parseUUID(w, chatID)
			if !ok {
				continue
			}

			projectUUID, ok := app.parseUUID(w, req.ProjectID)
			if !ok {
				continue
			}

			// Create a new interrupt channel for this request
			interruptMutex.Lock()
			interrupt = make(chan struct{})
			isProcessing = true
			interruptMutex.Unlock()

			// Process the prompt in a separate goroutine
			go func() {
				defer func() {
					interruptMutex.Lock()
					isProcessing = false
					interruptMutex.Unlock()
				}()

				// Get the database ID and user ID from request or default to empty string

				dbID, err := app.projectDatabase.GetDbIDFromProject(projectUUID)
				if err != nil {
					app.errorLog.Print(err)
					return
				}

				userID := userID.String()

				app.infoLog.Printf(
					message,
					req.DBUsed,
					req.DocsUsed,
					dbID.String(),
					userID,
					chatID, // Use the chatID from the URL
					dbID.String(),)

				// Forward the message with the new schema fields
				promptResponse, err := app.chatPort.ForwardMessageWithStream(
					message,
					req.DBUsed,
					req.DocsUsed,
					dbID.String(),
					userID,
					chatID, // Use the chatID from the URL
					projectUUID.String(),
				)
				if err != nil {
					app.errorLog.Printf("Error forwarding message: %v", err)
					if err := sendWSJSON(ws, FinalResponse{Status: "Error: " + err.Error()}); err != nil {
						app.errorLog.Println("write error:", err)
					}
					return
				}

				// Variable to store the final answer for insertion into the database
				var finalAnswer string

				// Process incoming prompt responses
			processLoop:
				for {
					select {
					case prompt, ok := <-promptResponse:
						if !ok {
							// Channel closed, no more messages
							break processLoop
						}

						app.infoLog.Print(prompt)
						finalResp := app.processPrompt(prompt)

						// Store the final answer if present
						if finalResp.Response != "" {
							finalAnswer = finalResp.Response
						} else if finalResp.Answer != "" {
							finalAnswer = finalResp.Answer
						}

						if err := sendWSJSON(ws, finalResp); err != nil {
							app.errorLog.Println("write error:", err)
							break processLoop
						}
					case <-interrupt:
						// Handle interruption
						app.infoLog.Println("Processing interrupted")

						// Send one final message indicating interruption
						finalResp := FinalResponse{
							Status:      "Generation interrupted by user.",
							Answer:      finalAnswer, // Include any partial answer for backward compatibility
							Response:    finalAnswer, // Include for new schema
							Interrupted: true,
						}

						if err := sendWSJSON(ws, finalResp); err != nil {
							app.errorLog.Println("write error:", err)
						}

						break processLoop
					}
				}

				// Only save to database if we have a complete answer
				if finalAnswer != "" {
					app.messages.Insert(chatUUID, "You", message)
					app.messages.Insert(chatUUID, "AI", finalAnswer)
					app.chats.UpdateLastActivity(chatUUID)
				}
			}()
		}
	}()

	// Keep the connection open
	<-r.Context().Done()
}

// parseWSRequest unmarshals the JSON message into a WebSocketRequest
func parseWSRequest(msg []byte) (WebSocketRequest, error) {
	var req WebSocketRequest
	err := json.Unmarshal(msg, &req)
	return req, err
}

// validateGeoJSON checks if the provided data represents valid GeoJSON
func validateGeoJSON(data []byte) (bool, string) {
	// First try to parse the JSON
	var geoJSON map[string]interface{}
	if err := json.Unmarshal(data, &geoJSON); err != nil {
		return false, fmt.Sprintf("Invalid JSON: %v", err)
	}

	// Check for type field
	typeVal, hasType := geoJSON["type"]
	if !hasType {
		return false, "Missing 'type' property"
	}

	geoJSONType, ok := typeVal.(string)
	if !ok {
		return false, "'type' property is not a string"
	}

	// Validate based on the GeoJSON type
	switch geoJSONType {
	case "FeatureCollection":
		// Check if features exist and it's an array
		featuresVal, hasFeatures := geoJSON["features"]
		if !hasFeatures {
			return false, "FeatureCollection missing 'features' property"
		}

		features, ok := featuresVal.([]interface{})
		if !ok {
			return false, "FeatureCollection 'features' is not an array"
		}

		if len(features) == 0 {
			return false, "FeatureCollection has empty 'features' array"
		}

		return true, fmt.Sprintf("Valid FeatureCollection with %d features", len(features))

	case "Feature":
		// Check if geometry exists
		geometryVal, hasGeometry := geoJSON["geometry"]
		if !hasGeometry {
			return false, "Feature missing 'geometry' property"
		}

		geometry, ok := geometryVal.(map[string]interface{})
		if !ok {
			return false, "Feature 'geometry' is not an object"
		}

		geometryType, hasGeometryType := geometry["type"]
		if !hasGeometryType {
			return false, "Feature geometry missing 'type' property"
		}

		return true, fmt.Sprintf("Valid Feature with geometry type: %v", geometryType)

	case "Point", "LineString", "Polygon", "MultiPoint", "MultiLineString", "MultiPolygon":
		// Check if coordinates exist and it's an array
		coordsVal, hasCoords := geoJSON["coordinates"]
		if !hasCoords {
			return false, fmt.Sprintf("%s missing 'coordinates' property", geoJSONType)
		}

		_, ok := coordsVal.([]interface{})
		if !ok {
			return false, fmt.Sprintf("%s 'coordinates' is not an array", geoJSONType)
		}

		return true, fmt.Sprintf("Valid %s geometry", geoJSONType)

	case "GeometryCollection":
		// Check if geometries exist and it's an array
		geometriesVal, hasGeometries := geoJSON["geometries"]
		if !hasGeometries {
			return false, "GeometryCollection missing 'geometries' property"
		}

		geometries, ok := geometriesVal.([]interface{})
		if !ok {
			return false, "GeometryCollection 'geometries' is not an array"
		}

		if len(geometries) == 0 {
			return false, "GeometryCollection has empty 'geometries' array"
		}

		return true, fmt.Sprintf("Valid GeometryCollection with %d geometries", len(geometries))

	default:
		return false, fmt.Sprintf("Unknown GeoJSON type: %s", geoJSONType)
	}
}

// Process the prompt and return a FinalResponse
func (app *application) processPrompt(prompt string) FinalResponse {
	var response FinalResponse

	// Log the raw response for debugging
	app.infoLog.Printf("Raw response from upstream server: %s", prompt)

	// Try to parse as a combined response (status + response + geo_objects)
	var combinedResponse struct {
		Status     string               `json:"status,omitempty"`
		Response   string               `json:"response,omitempty"`
		GeoObjects map[string]GeoObject `json:"geo_objects,omitempty"`
	}

	if err := json.Unmarshal([]byte(prompt), &combinedResponse); err == nil {
		// If any field is populated, build the response
		if combinedResponse.Status != "" || combinedResponse.Response != "" || len(combinedResponse.GeoObjects) > 0 {
			response.Status = combinedResponse.Status

			if combinedResponse.Response != "" {
				response.Answer = combinedResponse.Response   // For backward compatibility
				response.Response = combinedResponse.Response // New schema
			}

			if len(combinedResponse.GeoObjects) > 0 {
				response.GeoObjects = combinedResponse.GeoObjects

				// Create a simplified single FeatureCollection from geo_objects
				var allFeatures []json.RawMessage

				// Log the GeoObjects details
				app.infoLog.Printf("Processing GeoObjects with %d items", len(combinedResponse.GeoObjects))

				for shapeType, geoObject := range combinedResponse.GeoObjects {
					app.infoLog.Printf("Processing '%s' with type '%s'", shapeType, geoObject.Type)

					// Ensure we can unmarshal the Features property
					var features []json.RawMessage
					if err := json.Unmarshal(geoObject.Features, &features); err != nil {
						app.errorLog.Printf("Error unmarshaling features for %s: %v", shapeType, err)
						continue
					}

					app.infoLog.Printf("Successfully parsed %d features for '%s'", len(features), shapeType)

					// Add each feature to our collection
					for _, feature := range features {
						allFeatures = append(allFeatures, feature)
					}
				}

				// Create a unified FeatureCollection
				unifiedGeoJSON := map[string]interface{}{
					"type":     "FeatureCollection",
					"features": allFeatures,
				}

				// Only proceed if we have actual features
				if len(allFeatures) > 0 {
					// Marshal to JSON for validation
					geoJSONBytes, err := json.Marshal(unifiedGeoJSON)
					if err != nil {
						app.errorLog.Printf("Error marshaling unified GeoJSON: %v", err)
					} else {
						// Validate the GeoJSON before sending it
						isValid, message := validateGeoJSON(geoJSONBytes)
						if isValid {
							app.infoLog.Printf("GeoJSON validation passed: %s", message)
							response.GeoJSON = geoJSONBytes
							app.infoLog.Printf("Created unified GeoJSON with %d features", len(allFeatures))
						} else {
							app.errorLog.Printf("GeoJSON validation failed: %s", message)
							// Don't include invalid GeoJSON in the response
						}
					}
				} else {
					app.errorLog.Printf("No valid features found in GeoObjects, not sending GeoJSON to frontend")
				}
			}

			return response
		}
	}

	// Try to parse as a ChatIntermediateResponse (just status)
	var intermediateResp ChatIntermediateResponse
	if err := json.Unmarshal([]byte(prompt), &intermediateResp); err == nil && intermediateResp.Status != "" {
		return FinalResponse{Status: intermediateResp.Status}
	}

	// Try to parse as a ChatFinalResponse (status + response)
	var finalResp ChatFinalResponse
	if err := json.Unmarshal([]byte(prompt), &finalResp); err == nil &&
		(finalResp.Status != "" || finalResp.Response != "") {
		return FinalResponse{
			Status:   finalResp.Status,
			Answer:   finalResp.Response, // For backward compatibility
			Response: finalResp.Response, // New schema
		}
	}

	// Try to parse as a ChatGeoJsonResponse (just geo_objects)
	var geoJsonResp ChatGeoJsonResponse
	if err := json.Unmarshal([]byte(prompt), &geoJsonResp); err == nil && len(geoJsonResp.GeoObjects) > 0 {
		response.GeoObjects = geoJsonResp.GeoObjects

		// Process GeoObjects into a proper FeatureCollection
		var allFeatures []json.RawMessage

		for shapeType, geoObject := range geoJsonResp.GeoObjects {
			var features []json.RawMessage
			if err := json.Unmarshal(geoObject.Features, &features); err != nil {
				app.errorLog.Printf("Error unmarshaling features for %s: %v", shapeType, err)
				continue
			}

			for _, feature := range features {
				allFeatures = append(allFeatures, feature)
			}
		}

		// Only proceed if we have actual features
		if len(allFeatures) > 0 {
			// Create unified FeatureCollection
			unifiedGeoJSON := map[string]interface{}{
				"type":     "FeatureCollection",
				"features": allFeatures,
			}

			// Marshal to JSON
			geoJSONBytes, err := json.Marshal(unifiedGeoJSON)
			if err == nil {
				// Validate before sending
				isValid, message := validateGeoJSON(geoJSONBytes)
				if isValid {
					app.infoLog.Printf("GeoJSON validation passed: %s", message)
					response.GeoJSON = geoJSONBytes
					app.infoLog.Printf("Created unified GeoJSON from ChatGeoJsonResponse with %d features", len(allFeatures))
				} else {
					app.errorLog.Printf("GeoJSON validation failed: %s", message)
				}
			} else {
				app.errorLog.Printf("Error marshaling unified GeoJSON: %v", err)
			}
		}

		return response
	}

	// Fallback: if the prompt looks like a direct GeoJSON object
	var geoJSONResponse map[string]interface{}
	if err := json.Unmarshal([]byte(prompt), &geoJSONResponse); err == nil {
		if geoJSONType, ok := geoJSONResponse["type"].(string); ok {
			if geoJSONType == "FeatureCollection" || geoJSONType == "Feature" ||
				geoJSONType == "Point" || geoJSONType == "LineString" || geoJSONType == "Polygon" ||
				geoJSONType == "MultiPoint" || geoJSONType == "MultiLineString" || geoJSONType == "MultiPolygon" ||
				geoJSONType == "GeometryCollection" {

				// Validate before sending
				isValid, message := validateGeoJSON([]byte(prompt))
				if isValid {
					app.infoLog.Printf("Direct GeoJSON validation passed: %s", message)
					response.GeoJSON = []byte(prompt)
					return response
				} else {
					app.errorLog.Printf("Direct GeoJSON validation failed: %s", message)
				}
			}
		}
	}

	// If all parsing attempts fail, return the raw prompt as a status
	return FinalResponse{Status: prompt}
}

// sendWSJSON is a helper function to send JSON over WebSocket
func sendWSJSON(ws *websocket.Conn, v interface{}) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.TextMessage, jsonBytes)
}
