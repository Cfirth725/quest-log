// Package repository coordinates relational data access, transaction scoping,
// and immutable ledger operations for the active database engine.
package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// ====================================================================
// -- ACCOUNT IDENTITY & PREFERENCE REPOSITORY --
// ====================================================================

// GetUserByID queries the relational database for a specific system participant
// using their unique primary key identifier.
func GetUserByID(ctx context.Context, db *sql.DB, id int) (*User, error) {
	query := `
		SELECT 
			id, 
			name, 
			COALESCE(pin_hash, ''), 
			COALESCE(theme_preference, 'obsidian'), 
			dopamine_streak
		FROM users
		WHERE id = ?
	`
	var u User
	err := db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.PinHash,
		&u.ThemePreference,
		&u.DopamineStreak,
	)
	if err != nil {
		return nil, fmt.Errorf("user profile resolution failure for ID %d: %w", id, err)
	}
	return &u, nil
}

// GetAllUsers retrieves all registered user profiles to populate system navigation,
// profile switcher matrices, and form context bindings.
func GetAllUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	query := `
		SELECT 
			id, 
			name, 
			COALESCE(pin_hash, ''), 
			COALESCE(theme_preference, 'obsidian'), 
			dopamine_streak
		FROM users
		ORDER BY id ASC
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query system user accounts: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.PinHash,
			&u.ThemePreference,
			&u.DopamineStreak,
		)
		if err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

// UpdateUserTheme persists a participant's preferred visual color tokens (e.g., 'obsidian')
// to maintain consistency across sessions and devices.
func UpdateUserTheme(ctx context.Context, db *sql.DB, userID int, theme string) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET theme_preference = ? WHERE id = ?", theme, userID)
	if err != nil {
		return fmt.Errorf("user theme update transaction failure: %w", err)
	}
	return nil
}

// SetUserPIN updates or initializes a participant's hashed PIN for gateway authentication.
func SetUserPIN(ctx context.Context, db *sql.DB, userID int, pinHash string) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET pin_hash = ? WHERE id = ?", pinHash, userID)
	if err != nil {
		return fmt.Errorf("user PIN update transaction failure: %w", err)
	}
	return nil
}

// IncrementDopamineStreak increments a user's active completion streak count upon task resolution.
func IncrementDopamineStreak(ctx context.Context, db *sql.DB, userID int) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET dopamine_streak = dopamine_streak + 1 WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("dopamine streak increment transaction failure: %w", err)
	}
	return nil
}
