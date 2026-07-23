// ====================================================================
// -- INGESTION DOMAIN: BATCH TRANSACTION INTEGRATION TESTS --
// ====================================================================

package ingest

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ====================================================================
// -- TEST FIXTURES & IN-MEMORY DATABASE SETUP --
// ====================================================================

// setupTestDB initializes an in-memory SQLite instance provisioned with active system schemas.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("setup fault: failed opening in-memory db: %v", err)
	}

	schema := `
	CREATE TABLE categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id INTEGER DEFAULT 1,
		name TEXT UNIQUE NOT NULL,
		color_hex TEXT DEFAULT '#9F7AEA',
		is_archived INTEGER DEFAULT 0
	);

	CREATE TABLE quests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		category_id INTEGER REFERENCES categories(id),
		difficulty INTEGER NOT NULL,
		base_xp INTEGER NOT NULL,
		is_non_negotiable INTEGER DEFAULT 0,
		status TEXT DEFAULT 'active',
		owner_id INTEGER DEFAULT 1,
		quest_type TEXT DEFAULT 'One-Time',
		repeat_interval_days INTEGER,
		reset_day_of_week INTEGER DEFAULT 0
	);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("setup fault: failed initializing database schema: %v", err)
	}

	return db
}

// ====================================================================
// -- TRANSACTIONAL BATCH INGESTION SUITE --
// ====================================================================

func TestExecuteBatchIngestion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jsonInput := `[
		{
			"title": "Configure Home Lab Monitoring",
			"category": "DevOps",
			"difficulty": "hard",
			"quest_type": "One-Time",
			"is_non_negotiable": true
		},
		{
			"title": "Daily Walks with the Pups",
			"category": "Health",
			"difficulty": 1,
			"quest_type": "Daily"
		},
		{
			"title": "Weekly Server Backup Sweep",
			"category": "DevOps",
			"difficulty": "medium",
			"quest_type": "Weekly",
			"reset_day_of_week": 0
		}
	]`

	// 1. Pipeline Extraction Phase
	extracted, err := ParseJSONPayload([]byte(jsonInput))
	if err != nil {
		t.Fatalf("extraction fault: failed parsing test json payload: %v", err)
	}

	// 2. Transactional Batch Execution Phase
	ctx := context.Background()
	result, err := ExecuteBatchIngestion(ctx, db, 1, extracted)
	if err != nil {
		t.Fatalf("execution fault: batch ingestion transaction failed: %v", err)
	}

	// 3. Telemetry & Metric Assertions
	if result.QuestsMinted != 3 {
		t.Errorf("assertion fault: expected 3 quests minted, got %d", result.QuestsMinted)
	}

	// Verify two unique categories ("DevOps" and "Health") were auto-minted
	if result.CategoriesCreated != 2 {
		t.Errorf("assertion fault: expected 2 categories created, got %d", result.CategoriesCreated)
	}

	// 4. Persistence Direct State Sweep
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM quests WHERE category_id = (SELECT id FROM categories WHERE name = 'DevOps')").Scan(&count)
	if err != nil {
		t.Fatalf("state verification fault: failed querying persistent quest count: %v", err)
	}

	if count != 2 {
		t.Errorf("assertion fault: expected 2 quests under 'DevOps' category, got %d", count)
	}
}
