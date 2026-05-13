package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

// DB serves as the primary connection pool for the application.
// It is initialized during the system startup phase and shared across handlers.
var DB *sql.DB

// schemaSQL utilizes Go's embed directive to package the 'schema.sql' file
// directly into the binary, ensuring consistent deployments across environments.
//
//go:embed schema.sql
var schemaSQL string

// Connect initializes the SQLite driver, configures performance pragmas,
// and executes the schema migration suite.
func Connect() (*sql.DB, error) {
	// 1. Pull the path from the environment variable (e.g., /app/quests.db)
	dbPath := os.Getenv("DB_PATH")

	// 2. Fallback to a local path if the environment variable isn't set
	if dbPath == "" {
		dbPath = "./data/quests.db"
	}

	// Build a robust DSN string:
	// _loc=Local: Matches DB time to the Raspberry Pi's system clock.
	// parseTime=true: Automatically converts SQL timestamps to Go time.Time objects.
	// _foreign_keys=on: Ensures relational integrity for your user/quest links.
	dsn := fmt.Sprintf("%s?_loc=Local&parseTime=true&_foreign_keys=on", dbPath)

	// 3. Use the dynamic path instead of a hardcoded string
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("database: failed to open connection: %w", err)
	}

	// 4. Verify the health of the database connection before proceeding.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database: connectivity check failed: %w", err)
	}

	// Apply SQLite PRAGMAs to optimize for the Quest Log's specific requirements:
	// 1. WAL (Write-Ahead Logging) enables concurrent read/write operations.
	// 2. NORMAL Synchronous mode balances data safety with write performance.
	// 3. Foreign Key enforcement is explicitly enabled to maintain relational integrity.
	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`
	if _, err := db.Exec(pragmas); err != nil {
		return nil, fmt.Errorf("database: failed to apply performance pragmas: %w", err)
	}

	// Synchronize the database state with the embedded schema definition.
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("database: schema migration failed: %w", err)
	}

	return db, nil
}

// GetCategories retrieves all non-archived quest categories.
// It is used to populate selection menus in the Quest Forge and dashboard filters.
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

// GetUsers retrieves a list of active household participants.
// This data is used for quest attribution and reward distribution.
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

// CorralCompletedQuests executes a bulk update to transition 'One-Time' tasks
// into an 'Archived' state. This cleanses the UI pasture while preserving
// historical records in the completions ledger.
func CorralCompletedQuests(db *sql.DB) (int64, error) {
	// A transaction is utilized here to ensure that the archival process
	// is atomic—preventing partial updates if a system failure occurs.
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("database: failed to begin archival transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE quests 
		SET status = 'Archived' 
		WHERE quest_type = 'One-Time' 
		AND status = 'Completed';`

	result, err := tx.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("database: archival update failed: %w", err)
	}

	rows, _ := result.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("database: archival commit failed: %w", err)
	}

	return rows, nil
}

// createTables applies the raw SQL schema embedded at compile-time.
// This method is idempotent, meaning it can be run safely on every startup.
func createTables(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("database: failed to execute embedded schema: %w", err)
	}
	return nil
}
