package models

import (
    "database/sql"
    "errors"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/mattn/go-sqlite3"
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

func (m *UserModel) Insert(name, email, password string) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return err
    }

    email = strings.TrimSpace(strings.ToLower(email))
    
    // Generate a new UUID for the user
    userID := uuid.New()

    stmt := `INSERT INTO users (id, name, email, hashed_password, created)
    VALUES(?, ?, ?, ?, datetime('now'))`

    statement, err := m.DB.Prepare(stmt)
    if err != nil {
        return err
    }

    _, err = statement.Exec(userID, name, email, string(hashedPassword))
    if err != nil {
        sqliteErr := err.(*sqlite3.Error)
        if errors.As(err, &sqliteErr) {
            if sqliteErr.Code == sqlite3.ErrConstraint {
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

    stmt := "SELECT id, hashed_password FROM users WHERE email = ?"

    err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return uuid.Nil, ErrInvalidCredentials
        } else {
            return uuid.Nil, err
        }
    }

    err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
    if err != nil {
        if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
            return uuid.Nil, ErrInvalidCredentials
        } else {
            return uuid.Nil, err
        }
    }

    return id, nil
}

func (m *UserModel) Exists(id uuid.UUID) (bool, error) {
    var exists bool
    
    stmt := "SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)"
    
    err := m.DB.QueryRow(stmt, id).Scan(&exists)
    if err != nil {
        return false, err
    }
    
    return exists, nil
}

func (m *UserModel) GetByID(id uuid.UUID) (*User, error) {
    var user User
    
    stmt := "SELECT id, name, email, hashed_password, created FROM users WHERE id = ?"
    
    err := m.DB.QueryRow(stmt, id).Scan(&user.ID, &user.Name, &user.Email, &user.HashedPassword, &user.Created)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNoRecord
        }
        return nil, err
    }
    
    return &user, nil
}