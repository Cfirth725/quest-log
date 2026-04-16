package main

import (
	"database/sql"
	"fmt"

	// The blank identifier '_' is used here because we don't call this package's functions directly.
	// We just need it to run its 'init()' function to register itself with Go's 'database/sql' package.
	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a connection to the SQLite database, configures it for production use,
// and ensures all required tables exist.
func Connect() (*sql.DB, error) {
	// 1. Open the connection to the file (creates quests.db if it doesn't exist)
	db, err := sql.Open("sqlite3", "./quests.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 2. Ping the database to ensure the connection is actually valid
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 3. Configure SQLite for performance and data integrity
	// WAL (Write-Ahead Logging) greatly improves concurrent read/write speeds.
	// Foreign Keys must be explicitly turned on in SQLite to enforce our schema rules.
	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`
	if _, err := db.Exec(pragmas); err != nil {
		return nil, fmt.Errorf("failed to set database pragmas: %w", err)
	}

	// 4. Run auto-migrations (create tables if they don't exist)
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
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

	_, err := db.Exec(schema)
	return err
}