// models/users.go
package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             uuid.UUID
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModel struct {
	DB *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{DB: db}
}

func (m *UserModel) Insert(name, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	email = strings.TrimSpace(strings.ToLower(email))

	// We're mapping to the 'users' table in the 'public' schema
	// Also using RETURNING to get the UUID assigned by the database
	stmt := `
        INSERT INTO users (id, name, email, hashed_password, created, role)
        VALUES ($1, $2, $3, $4, NOW(), 'user')
        RETURNING id
    `

    userId := uuid.New()
	var id uuid.UUID
	err = m.DB.QueryRow(stmt, userId, name, email, string(hashedPassword)).Scan(&id)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			// Handle unique constraint violations
			if pgErr.Code == "23505" && strings.Contains(pgErr.Message, "email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

func (m *UserModel) Authenticate(email, password string) (uuid.UUID, error) {
	var id uuid.UUID
	var hashedPassword []byte

	email = strings.TrimSpace(strings.ToLower(email))

	// Query the users table for the email
	stmt := "SELECT id, hashed_password FROM users WHERE email = $1"

	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrInvalidCredentials
		}
		return uuid.Nil, err
	}

	// Compare the hashed password with the provided password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return uuid.Nil, ErrInvalidCredentials
		}
		return uuid.Nil, err
	}

	return id, nil
}

func (m *UserModel) Get(id uuid.UUID) (*User, error) {
	stmt := `
        SELECT id, name, email, hashed_password, created
        FROM users 
        WHERE id = $1
    `
	
	var user User
	err := m.DB.QueryRow(stmt, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.HashedPassword,
		&user.Created,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}
	
	return &user, nil
}

func (m *UserModel) Exists(id uuid.UUID) (bool, error) {
	var exists bool
	stmt := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	
	err := m.DB.QueryRow(stmt, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	
	return exists, nil
}