package models

import (
	"database/sql"
	"time"
)

// Message represents a single message in a chat
type Message struct {
	ID        int
	ChatID    int
	Sender    User
	Content   string
	Timestamp time.Time
}

// MessageModel handles database operations for messages
type MessageModel struct {
	DB *sql.DB
}

// Insert adds a new message to a chat
func (m *MessageModel) Insert(chatID int, senderID int, content string) error {
	stmt := `
		INSERT INTO messages (chat_id, sender_id, content, timestamp)
		VALUES (?, ?, ?, ?)
	`
	statement, err := m.DB.Prepare(stmt)
	if err != nil {
		return err
	}
	
	_, err = statement.Exec(chatID, senderID, content, time.Now())
	if err != nil {
		return err
	}
	
	return nil
}

// GetByChatID retrieves all messages for a specific chat
func (m *MessageModel) GetByChatID(chatID int) ([]*Message, error) {
	stmt := `
		SELECT m.id, m.chat_id, m.content, m.timestamp, 
		       u.id, u.name, u.email
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.chat_id = ?
		ORDER BY m.timestamp ASC
	`
	
	rows, err := m.DB.Query(stmt, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	messages := []*Message{}
	for rows.Next() {
		msg := Message{}
		user := User{}
		
		err = rows.Scan(
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