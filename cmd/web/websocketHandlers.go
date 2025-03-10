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
	Status string `json:"status,omitempty"`
	Answer string `json:"answer,omitempty"`
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

		app.messages.Insert(chatUUID, "You", req.Message)

		for prompt := range promptResponse {
			app.infoLog.Print(prompt)
			finalResp := app.processPrompt(prompt)
			
			// assume 0 is bot/ai/llm
			if finalResp.Answer != "" {
				app.messages.Insert(chatUUID, "AI", finalResp.Answer)
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

// processPrompt converts the raw prompt string into a FinalResponse.
func (app *application) processPrompt(prompt string) FinalResponse {
	var streamResp LLMStreamResponse
	if err := json.Unmarshal([]byte(prompt), &streamResp); err != nil {
		app.errorLog.Println("error unmarshaling LLM response:", err)
		// Fallback: return the raw prompt as a status.
		return FinalResponse{Status: prompt}
	}

	if streamResp.Status != "" {
		return FinalResponse{Status: streamResp.Status}
	} else if streamResp.Result != "" && streamResp.Details != "" {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal([]byte(streamResp.Details), &detailsMap); err == nil {
			if answer, ok := detailsMap["answer"].(string); ok {
				return FinalResponse{Answer: answer}
			}
		}
	}
	return FinalResponse{}
}

// sendJSON marshals the given data and sends it as a TextMessage.
func sendJSON(ws *websocket.Conn, data interface{}) error {
	jsonRes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.TextMessage, jsonRes)
}
