package models

import (
	"database/sql"
	"time"
)

type Message struct {
	ID        int
	ChatID    int
	Sender    User
	Content   string
	Timestamp time.Time
}

type MessageModel struct {
	DB *sql.DB
}

func (m *MessageModel) Insert(userID int) error {

	stmt := ""

	statement, err := m.DB.Prepare(stmt)
	if err != nil {
		return err
	}

	_, err = statement.Exec(userID)
	if err != nil {
		return err
	}

	return nil
}