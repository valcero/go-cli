package main

import (
	"database/sql"
	"errors"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Username       string
	PasswordHash   string
	FailedAttempts int
	LockedUntil    sql.NullTime
	TotpEnabled    bool
	TotpSecret     sql.NullString
	CreatedAt      time.Time
	LastLogin      sql.NullTime
}

type Session struct {
	ID        string
	UserID    int
	ExpiresAt time.Time
}

func RegisterUser(db *sql.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, string(hash))
	return err
}

func LoginUser(db *sql.DB, username, password, totpCode string) (*User, *Session, error) {
	var u User

	err := db.QueryRow("SELECT id, username, password_hash, failed_attempts, locked_until, totp_enabled, totp_secret, created_at, last_login FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FailedAttempts, &u.LockedUntil, &u.TotpEnabled, &u.TotpSecret, &u.CreatedAt, &u.LastLogin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, err
	}

	if u.LockedUntil.Valid && u.LockedUntil.Time.After(time.Now()) {
		return nil, nil, errors.New("account is locked, please try again later")
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		u.FailedAttempts++
		if u.FailedAttempts >= 3 {
			lockTime := time.Now().Add(15 * time.Minute)
			db.Exec("UPDATE users SET failed_attempts = $1, locked_until = $2 WHERE id = $3", u.FailedAttempts, lockTime, u.ID)
			return nil, nil, errors.New("account locked due to too many failed attempts")
		}
		db.Exec("UPDATE users SET failed_attempts = $1 WHERE id = $2", u.FailedAttempts, u.ID)
		return nil, nil, errors.New("invalid credentials")
	}

	if u.TotpEnabled {
		if totpCode == "" {
			return &u, nil, errors.New("totp_required")
		}
		valid := totp.Validate(totpCode, u.TotpSecret.String)
		if !valid {
			return nil, nil, errors.New("invalid 2FA code")
		}
	}

	db.Exec("UPDATE users SET failed_attempts = 0, locked_until = NULL, last_login = CURRENT_TIMESTAMP WHERE id = $1", u.ID)

	expiresAt := time.Now().Add(1 * time.Hour)
	var sessionID string
	err = db.QueryRow("INSERT INTO sessions (user_id, expires_at) VALUES ($1, $2) RETURNING id", u.ID, expiresAt).Scan(&sessionID)
	if err != nil {
		return nil, nil, err
	}

	session := &Session{
		ID:        sessionID,
		UserID:    u.ID,
		ExpiresAt: expiresAt,
	}

	// Update the local struct to reflect the immediate DB change
	u.LastLogin = sql.NullTime{Time: time.Now(), Valid: true}

	return &u, session, nil
}

func LogoutUser(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = $1", sessionID)
	return err
}

func Enable2FA(db *sql.DB, userID int, secret string) error {
	_, err := db.Exec("UPDATE users SET totp_secret = $1, totp_enabled = TRUE WHERE id = $2", secret, userID)
	return err
}

func Disable2FA(db *sql.DB, userID int) error {
	_, err := db.Exec("UPDATE users SET totp_secret = NULL, totp_enabled = FALSE WHERE id = $1", userID)
	return err
}
