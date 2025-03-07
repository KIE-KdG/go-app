package models

import (
	"database/sql"
	"time"
)

// Chat represents a conversation between users
type Chat struct {
	ID          int
	UserID      int
	Messages    []Message
	Created     time.Time
	LastActivity time.Time
}

// ChatModel handles database operations for chats
type ChatModel struct {
	DB *sql.DB
}

// Insert creates a new chat for a user
func (m *ChatModel) Insert(userID int) (int, error) {
	stmt := `
		INSERT INTO chats (user_id, created, last_activity)
		VALUES (?, ?, ?)
	`
	statement, err := m.DB.Prepare(stmt)
	if err != nil {
		return 0, err
	}
	
	now := time.Now()
	result, err := statement.Exec(userID, now, now)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	
	return int(id), nil
}

// RetrieveByUserId gets all chats for a specific user
func (m *ChatModel) RetrieveByUserId(userId int) ([]*Chat, error) {
	stmt := `
		SELECT id, user_id, created, last_activity
		FROM chats
		WHERE user_id = ?
		ORDER BY last_activity DESC
	`
	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	chats := []*Chat{}
	for rows.Next() {
		c := &Chat{}
		err = rows.Scan(&c.ID, &c.UserID, &c.Created, &c.LastActivity)
		if err != nil {
			return nil, err
		}
		
		// We'll load messages separately via MessageModel
		chats = append(chats, c)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return chats, nil
}

// GetByID retrieves a single chat by its ID
func (m *ChatModel) GetByID(id int) (*Chat, error) {
	stmt := `
		SELECT id, user_id, created, last_activity
		FROM chats
		WHERE id = ?
	`
	
	row := m.DB.QueryRow(stmt, id)
	
	c := &Chat{}
	err := row.Scan(&c.ID, &c.UserID, &c.Created, &c.LastActivity)
	if err != nil {
		return nil, err
	}
	
	return c, nil
}

// UpdateLastActivity updates the last_activity timestamp of a chat
func (m *ChatModel) UpdateLastActivity(chatID int) error {
	stmt := `
		UPDATE chats
		SET last_activity = ?
		WHERE id = ?
	`
	
	_, err := m.DB.Exec(stmt, time.Now(), chatID)
	return err
}

// Delete removes a chat and its messages (if CASCADE is set up)
func (m *ChatModel) Delete(id int) error {
	stmt := `DELETE FROM chats WHERE id = ?`
	_, err := m.DB.Exec(stmt, id)
	return err
}