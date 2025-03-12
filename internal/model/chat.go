package model

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type ChatPort struct {
	Port string
}

// ChatRequest represents the new schema for chat requests
type ChatRequest struct {
	Question   string `json:"question"`
	Message    string `json:"message,omitempty"` // For backward compatibility
	DBUsed     bool   `json:"dbUsed"`
	DocsUsed   bool   `json:"docsUsed"`
	DatabaseID string `json:"database_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
}

// ForwardMessageWithStream sends a message to the connected websocket server
// and returns a channel that streams the responses back.
// The channel will be closed when the response is complete or if there's an error.
func (c *ChatPort) ForwardMessageWithStream(
	message string,
	dbUsed bool,
	docsUsed bool,
	databaseID string,
	userID string,
	chatID string,
) (<-chan string, error) {
	otherServer := "ws://localhost" + c.Port + "/ws/chat"
	conn, _, err := websocket.DefaultDialer.Dial(otherServer, nil)
	if err != nil {
		return nil, err
	}

	// Use Question for new schema and Message for backward compatibility
	req := ChatRequest{
		Question:   message,
		Message:    message,
		DBUsed:     dbUsed,
		DocsUsed:   docsUsed,
		DatabaseID: databaseID,
		UserID:     userID,
		ChatID:     chatID,
	}

	jsonMsg, err := json.Marshal(req)
	if err != nil {
		conn.Close()
		return nil, err
	}

	respChan := make(chan string)

	go func() {
		defer conn.Close()
		defer close(respChan)

		// Write the initial message to the upstream server.
		if err := conn.WriteMessage(websocket.TextMessage, jsonMsg); err != nil {
			return
		}

		// Continuously read messages and send them through the channel.
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			
			respChan <- string(msg)
		}
	}()

	return respChan, nil
}