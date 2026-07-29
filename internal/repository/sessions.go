// Package repository coordinates relational data access, transaction scoping,
// and immutable ledger operations for the active database engine.
package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// ====================================================================
// -- SECURE SESSION STATE MANAGEMENT --
// ====================================================================

// GenerateSessionToken creates a cryptographically secure 32-byte hex string
// to serve as an unguessable authentication token for browser cookies.
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("cryptographic entropy failure: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession mints a new secure session token tied to a target user_id
// and persists it to the database with a strict expiration window.
func CreateSession(ctx context.Context, db *sql.DB, userID int, duration time.Duration) (*Session, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("session token generation block: %w", err)
	}

	expiresAt := time.Now().Add(duration)

	query := `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`
	_, err = db.ExecContext(ctx, query, token, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("session persistence transaction failure: %w", err)
	}

	return &Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}, nil
}

// GetSessionByToken retrieves an active session record and verifies that it
// has not passed its expiration time window.
func GetSessionByToken(ctx context.Context, db *sql.DB, token string) (*Session, error) {
	query := `
		SELECT token, user_id, expires_at, created_at
		FROM sessions
		WHERE token = ? AND expires_at > ?
	`
	row := db.QueryRowContext(ctx, query, token, time.Now())

	var sess Session
	err := row.Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &sess, nil
}

// DeleteSession revokes an active session token from the persistence engine upon logout.
func DeleteSession(ctx context.Context, db *sql.DB, token string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("session revocation transaction failure: %w", err)
	}
	return nil
}
