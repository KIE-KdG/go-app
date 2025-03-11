package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type LLMStreamResponse struct {
	Status  string `json:"status,omitempty"`
	Result  string `json:"result,omitempty"`
	Details string `json:"details,omitempty"`
}

type FinalResponse struct {
	Status     string          `json:"status,omitempty"`
	Answer     string          `json:"answer,omitempty"`
	GeoJSON    json.RawMessage `json:"geoJSON,omitempty"` // Field for GeoJSON data
	Interrupted bool           `json:"interrupted,omitempty"` // New field to indicate interruption
}

type WebSocketRequest struct {
	Message   string `json:"message"`
	DBUsed    bool   `json:"dbUsed"`
	DocsUsed  bool   `json:"docsUsed"`
	Interrupt bool   `json:"interrupt"` // New field to signal interruption
}

func (app *application) handleConnections(w http.ResponseWriter, r *http.Request) {
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

	// Start a goroutine to handle interrupt messages from client
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
					if err := sendJSON(ws, FinalResponse{Interrupted: true}); err != nil {
						app.errorLog.Println("error sending interrupt acknowledgment:", err)
					}
				}
				interruptMutex.Unlock()
				continue // Skip the rest of the loop for interrupts
			}

			// Regular message handling
			app.infoLog.Printf("Received message: %s, DB: %t, Docs: %t", req.Message, req.DBUsed, req.DocsUsed)

			chatUUID, err := uuid.Parse(chatID)
			if err != nil {
				app.errorLog.Printf("Could not parse into UUID: %s", chatID)
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

				promptResponse, err := app.chatPort.ForwardMessageWithStream(req.Message, req.DBUsed, req.DocsUsed)
				if err != nil {
					app.errorLog.Printf("Error forwarding message: %v", err)
					if err := sendJSON(ws, FinalResponse{Status: "Error: " + err.Error()}); err != nil {
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
						if finalResp.Answer != "" {
							finalAnswer = finalResp.Answer
						}

						if err := sendJSON(ws, finalResp); err != nil {
							app.errorLog.Println("write error:", err)
							break processLoop
						}
					case <-interrupt:
						// Handle interruption
						app.infoLog.Println("Processing interrupted")
						
						// Send one final message indicating interruption
						finalResp := FinalResponse{
							Status: "Generation interrupted by user.",
							Answer: finalAnswer, // Include any partial answer we have
						}
						
						if err := sendJSON(ws, finalResp); err != nil {
							app.errorLog.Println("write error:", err)
						}
						
						break processLoop
					}
				}

				// Only save to database if we have a complete answer
				if finalAnswer != "" {
					app.messages.Insert(chatUUID, "You", req.Message)
					app.messages.Insert(chatUUID, "AI", finalAnswer)
					app.chats.UpdateLastActivity(chatUUID)
				}
			}()
		}
	}()

	// Keep the connection open
	<-r.Context().Done()
}

// parseWSRequest unmarshals the JSON message into a WebSocketRequest.
func parseWSRequest(msg []byte) (WebSocketRequest, error) {
	var req WebSocketRequest
	err := json.Unmarshal(msg, &req)
	return req, err
}

// sendJSON marshals the given data and sends it as a TextMessage.
func sendJSON(ws *websocket.Conn, data interface{}) error {
	jsonRes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.TextMessage, jsonRes)
}

// Process the prompt and return a FinalResponse
func (app *application) processPrompt(prompt string) FinalResponse {
	var streamResp LLMStreamResponse
	var response FinalResponse

	if err := json.Unmarshal([]byte(prompt), &streamResp); err != nil {
			app.errorLog.Println("error unmarshaling LLM response:", err)
			// Fallback: return the raw prompt as a status.
			response = FinalResponse{Status: prompt}
	} else if streamResp.Status != "" {
			// For status updates, don't include GeoJSON
			response = FinalResponse{Status: streamResp.Status}
	} else if streamResp.Result != "" && streamResp.Details != "" {
			// For final answers, include both answer and GeoJSON
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(streamResp.Details), &detailsMap); err == nil {
					if answer, ok := detailsMap["answer"].(string); ok {
							response = FinalResponse{Answer: answer}
							
							// Include the dummy GeoJSON only with the final answer
							dummyGeoJSON, err := app.getDummyGeoJSON()
							if err == nil {
									response.GeoJSON = dummyGeoJSON
							} else {
									app.errorLog.Printf("Failed to load dummy GeoJSON: %v", err)
							}
					}
			}
	}
	
	return response
}

func (app *application) getDummyGeoJSON() (json.RawMessage, error) {
	// Static GeoJSON example that should definitely work
	staticGeoJSON := `{
			"type": "FeatureCollection",
			"features": [
					{
							"type": "Feature",
							"properties": {
									"name": "Test Location"
							},
							"geometry": {
									"type": "Point",
									"coordinates": [4.9041, 52.3676]
							}
					},
					{
							"type": "Feature",
							"properties": {
									"name": "Test Line"
							},
							"geometry": {
									"type": "LineString",
									"coordinates": [
											[4.9041, 52.3676],
											[4.8500, 52.3500]
									]
							}
					}
			]
	}`

	return []byte(staticGeoJSON), nil
}