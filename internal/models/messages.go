// models/messages.go
package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID         int
	ChatID     uuid.UUID
	SenderType string
	Content    string
	Timestamp  time.Time
}

type MessageModel struct {
	DB *sql.DB
}

func NewMessageModel(db *sql.DB) *MessageModel {
	return &MessageModel{DB: db}
}

func (m *MessageModel) Insert(chatID uuid.UUID, senderType, content string) error {
	stmt := `
		INSERT INTO messages (chat_id, sender_type, content, timestamp)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := m.DB.Exec(stmt, chatID, senderType, content)
	if err != nil {
		return err
	}

	return nil
}

func (m *MessageModel) GetByChatID(chatID uuid.UUID) ([]*Message, error) {
	stmt := `
		SELECT id, chat_id, sender_type, content, timestamp
		FROM messages
		WHERE chat_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := m.DB.Query(stmt, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		msg := &Message{}
		err = rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderType,
			&msg.Content,
			&msg.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}