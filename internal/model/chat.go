package model

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type ChatPort struct {
	Port string
}

type Chat struct {
	Message string `json:"question"`
}

type WebSocketRequest struct {
	Message  string `json:"question"`
	DBUsed   bool   `json:"dbUsed"`
	DocsUsed bool   `json:"docsUsed"`
}

func (c *ChatPort) ForwardMessage(message string) (string, error) {
	otherServer := "ws://localhost" + c.Port + "/ws/documents/search"
	conn, _, err := websocket.DefaultDialer.Dial(otherServer, nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		return "", err
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}

	return string(msg), nil
}


func (c *ChatPort) ForwardMessageWithStream(message string, dbUsed, docsUsed bool) (<-chan string, error) {
	otherServer := "ws://localhost" + c.Port + "/ws/documents/search"
	conn, _, err := websocket.DefaultDialer.Dial(otherServer, nil)
	if err != nil {
		return nil, err
	}

	req := WebSocketRequest{
		Message:  message,
    DBUsed:   dbUsed,
    DocsUsed: docsUsed,
	}

	jsonMsg, err := json.Marshal(req)
	if err != nil {
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
