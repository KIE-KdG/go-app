package model

import (
	"github.com/gorilla/websocket"
)

type ChatPort struct {
	Port string
}

type Chat struct {
	Message string `json:"message"`
}

func (c *ChatPort) ForwardMessage(message string) (string, error) {

	otherServer := "ws://localhost" + c.Port + "/ws/search"
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