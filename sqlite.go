package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// DB acts as the global connection pool for the application lifecycle.
var DB *sql.DB

//go:embed schema.sql
var schemaSQL string

// Connect initializes the SQLite driver, configures performance pragmas,
// and executes the schema migration suite.
func Connect() (*sql.DB, error) {
	// Initialize connection to the local SQLite file.
	// SQLite will automatically create the file if it does not exist.
	db, err := sql.Open("sqlite3", "./quests.db")
	if err != nil {
		return nil, fmt.Errorf("database: failed to open connection: %w", err)
	}

	// Verify connectivity to the database file.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database: connectivity check failed: %w", err)
	}

	// Configure SQLite for high-concurrency and data safety.
	// WAL (Write-Ahead Logging) allows simultaneous reads and writes.
	// Foreign Key enforcement is explicitly enabled to maintain relational integrity.
	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`
	if _, err := db.Exec(pragmas); err != nil {
		return nil, fmt.Errorf("database: failed to apply performance pragmas: %w", err)
	}

	// Execute idempotent schema migrations.
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("database: schema migration failed: %w", err)
	}

	return db, nil
}

// GetCategories retrieves active quest categories for the 'Quest Forge' interface.
func GetCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT id, name, color_hex FROM categories WHERE is_archived = 0 ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("database: category retrieval failed: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ColorHex); err != nil {
			return nil, fmt.Errorf("database: category scan failed: %w", err)
		}
		categories = append(categories, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("database: error during category iteration: %w", err)
	}
	return categories, nil
}

// GetUsers retrieves all active users for quest assignment.
func GetUsers(db *sql.DB) ([]User, error) {
	query := "SELECT id, name, dopamine_streak FROM users ORDER BY name ASC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("database: user retrieval failed: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DopamineStreak); err != nil {
			return nil, fmt.Errorf("database: user scan failed: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

// ----- Completed Quests -----
// CorralCompletedQuests shifts finished 'One-Time' tasks into 'Archived' status.
func CorralCompletedQuests(db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	// One time quests
	query := `
		UPDATE quests 
		SET status = 'Archived' 
		WHERE quest_type = 'One-Time' 
		AND status = 'Pending'
		AND id IN (SELECT quest_id FROM quest_completions);`

	result, err := tx.Exec(query)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	rows, _ := result.RowsAffected()
	err = tx.Commit()
	return rows, err
}

// createTables executes the schema creation.
func createTables(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to apply embedded schema: %w", err)
	}
	return nil
}
