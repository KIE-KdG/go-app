package models

import (
	"database/sql"
	"time"
)

type Chat struct {
	ID           int
	UserID       int
	Messages     []Message
	Created      time.Time
	LastActivity time.Time
}

type ChatModel struct {
	DB *sql.DB
}

func (m *ChatModel) Insert(userID int) error {

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

func (m *ChatModel) retrieveByUserId(userId int) ([]*Chat, error) {

	stmt := ""

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := []*Chat{}

	for rows.Next(){
		c := &Chat{}

		err = rows.Scan(&c.ID, &c.Messages, &c.Created)
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