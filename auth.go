package main

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Username       string
	PasswordHash   string
	FailedAttempts int
	LockedUntil    sql.NullTime
}

func RegisterUser(db *sql.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, string(hash))
	return err
}

func LoginUser(db *sql.DB, username, password string) (*User, error) {
	var u User

	err := db.QueryRow("SELECT id, username, password_hash, failed_attempts, locked_until FROM users WHERE username = $1", username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FailedAttempts, &u.LockedUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &u, nil
}
