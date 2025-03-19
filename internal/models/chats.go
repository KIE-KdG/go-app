// models/chats.go
package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Messages     []Message
	Created      time.Time
	LastActivity time.Time
}

type ChatModel struct {
	DB *sql.DB
}

func NewChatModel(db *sql.DB) *ChatModel {
	return &ChatModel{DB: db}
}

func (m *ChatModel) Insert(userID uuid.UUID) (uuid.UUID, error) {
	chatID := uuid.New()

	stmt := `
		INSERT INTO chats (id, user_id, created, last_activity)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id
	`

	
	err := m.DB.QueryRow(stmt, chatID, userID).Scan(&chatID)
	if err != nil {
		return uuid.Nil, err
	}

	return chatID, nil
}

func (m *ChatModel) RetrieveByUserId(userId uuid.UUID) ([]*Chat, error) {
	stmt := `
		SELECT id, user_id, created, last_activity
		FROM chats
		WHERE user_id = $1
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
		
		chats = append(chats, c)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return chats, nil
}

func (m *ChatModel) GetByID(id uuid.UUID) (*Chat, error) {
	stmt := `
		SELECT id, user_id, created, last_activity
		FROM chats
		WHERE id = $1
	`

	row := m.DB.QueryRow(stmt, id)

	c := &Chat{}
	err := row.Scan(&c.ID, &c.UserID, &c.Created, &c.LastActivity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return c, nil
}

func (m *ChatModel) UpdateLastActivity(chatID uuid.UUID) error {
	stmt := `
		UPDATE chats
		SET last_activity = NOW()
		WHERE id = $1
	`
	
	_, err := m.DB.Exec(stmt, chatID)
	return err
}