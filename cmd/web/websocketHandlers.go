package main

import (
	"encoding/json"
	"net/http"

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
	Status  string          `json:"status,omitempty"`
	Answer  string          `json:"answer,omitempty"`
	GeoJSON json.RawMessage `json:"geoJSON,omitempty"` // New field for GeoJSON data
}

type WebSocketRequest struct {
	Message  string `json:"message"`
	DBUsed   bool   `json:"dbUsed"`
	DocsUsed bool   `json:"docsUsed"`
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
		app.infoLog.Printf("Received message: %s, DB: %t, Docs: %t", req.Message, req.DBUsed, req.DocsUsed)

		chatUUID, err := uuid.Parse(chatID)
		if err != nil {
			app.errorLog.Printf("Could not pasrse into UUID: %s", chatID)
			return
		}

		promptResponse, err := app.chatPort.ForwardMessageWithStream(req.Message, req.DBUsed, req.DocsUsed)
		if err != nil {
			app.serverError(w, err)
			return
		}

		

		for prompt := range promptResponse {
			app.infoLog.Print(prompt)
			finalResp := app.processPrompt(prompt)
			
			if finalResp.Answer != "" {
				app.messages.Insert(chatUUID, "You", req.Message)
				app.messages.Insert(chatUUID, "AI", finalResp.Answer)
				app.chats.UpdateLastActivity(chatUUID)
			}

			if err := sendJSON(ws, finalResp); err != nil {
				app.errorLog.Println("write error:", err)
				break
			}
		}
	}
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

// Update the processPrompt function to only include GeoJSON in final responses
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