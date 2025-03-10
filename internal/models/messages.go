package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message represents a single message in a chat
type Message struct {
	ID         int
	ChatID     uuid.UUID
	SenderType string
	Sender     User
	Content    string
	Timestamp  time.Time
}

// MessageModel handles database operations for messages
type MessageModel struct {
	DB *sql.DB
}

// Insert adds a new message to a chat
func (m *MessageModel) Insert(chatID uuid.UUID, senderType, content string) error {
	stmt := `
			INSERT INTO messages (chat_id, sender_type, content, timestamp)
			VALUES (?, ?, ?, ?)
	`
	statement, err := m.DB.Prepare(stmt)
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer statement.Close()

	_, err = statement.Exec(chatID, senderType, content, time.Now())
	if err != nil {
		return fmt.Errorf("error executing insert statement: %w", err)
	}

	return nil
}

// GetByChatID retrieves all messages for a specific chat
func (m *MessageModel) GetByChatID(chatID uuid.UUID) ([]*Message, error) {
	stmt := `
		SELECT id, chat_id, content, timestamp, sender_type
		FROM messages
		WHERE chat_id = ?
		ORDER BY timestamp ASC
	`

	rows, err := m.DB.Query(stmt, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		msg := Message{}
		err = rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.Content,
			&msg.Timestamp,
			&msg.SenderType, // field indicating 'user' or 'AI'
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetByID retrieves a specific message by its ID
func (m *MessageModel) GetByID(id int) (*Message, error) {
	stmt := `
		SELECT m.id, m.chat_id, m.content, m.timestamp, 
		       u.id, u.name, u.email
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = ?
	`

	row := m.DB.QueryRow(stmt, id)

	msg := &Message{}
	user := User{}

	err := row.Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.Content,
		&msg.Timestamp,
		&user.ID,
		&user.Name,
		&user.Email,
	)

	if err != nil {
		return nil, err
	}

	msg.Sender = user
	return msg, nil
}

// Delete removes a message from the database
func (m *MessageModel) Delete(id int) error {
	stmt := `DELETE FROM messages WHERE id = ?`
	_, err := m.DB.Exec(stmt, id)
	return err
}
