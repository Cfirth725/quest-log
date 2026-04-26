package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// DB acts as the global connection pool for the application lifecycle.
var DB *sql.DB

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

// createTables executes the schema creation. Using "IF NOT EXISTS" ensures this
// only runs the first time the app boots, or if a table was accidentally deleted.
func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		dopamine_streak INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id INTEGER DEFAULT 0,
		name TEXT NOT NULL,
		color_hex TEXT,
		is_archived BOOLEAN DEFAULT 0,
		UNIQUE(owner_id, name)
	);

	CREATE TABLE IF NOT EXISTS quests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id INTEGER DEFAULT 0,
		category_id INTEGER,
		title TEXT NOT NULL,
		difficulty INTEGER CHECK( difficulty IN (1, 2, 3) ),
		base_xp INTEGER CHECK( base_xp IN (10, 25, 50) ),
		is_non_negotiable BOOLEAN DEFAULT 0,
		status TEXT DEFAULT 'Pending',
		quest_type TEXT CHECK( quest_type IN ('One-Time', 'Daily', 'Repeating') ),
		repeat_interval_days INTEGER DEFAULT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_completed_at DATETIME,
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);

	CREATE TABLE IF NOT EXISTS quest_completions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		quest_id INTEGER,
		completed_by_user_id INTEGER,
		completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		xp_awarded INTEGER,
		FOREIGN KEY(quest_id) REFERENCES quests(id)
	);

	CREATE TABLE IF NOT EXISTS gear_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		quest_id INTEGER,
		item_name TEXT NOT NULL,
		is_gathered BOOLEAN DEFAULT 0,
		FOREIGN KEY(quest_id) REFERENCES quests(id)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("database: schema execution failed: %w", err)
	}
	return nil
}
