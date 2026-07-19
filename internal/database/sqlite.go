// Package database coordinates low-level infrastructure bindings, performance pragmas,
// and automated sector compaction scripts for the persistent storage engine.
package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ====================================================================
// -- GLOBAL STORAGE PERSISTENCE CAPABILITIES --
// ====================================================================

// DB serves as the primary network connection pool for the application runtime.
var DB *sql.DB

// schemaSQL utilizes Go's embed directive to package the 'schema.sql' file
// directly into the binary, ensuring consistent deployments across environments.
//
//go:embed schema.sql
var schemaSQL string

// ====================================================================
// -- INFRASTRUCTURE INITIALIZATION & CONNECTIVITY LAYER --
// ====================================================================

// Connect initializes the SQLite driver, configures performance pragmas,
// and executes the schema migration suite using context lifecycles.
func Connect(ctx context.Context) (*sql.DB, error) {
	log.Println("[INIT] Extracting data persistence boundaries from host environment")
	dbPath := os.Getenv("DB_PATH")

	if dbPath == "" {
		dbPath = "./data/quests.db"
	}

	dsn := fmt.Sprintf("%s?_loc=Local&parseTime=true&_foreign_keys=on", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("database infrastructure failure: open sequence aborted: %w", err)
	}

	// Verify the physical file handle is reachable using a context-aware ping
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database connectivity check failed: hardware node unreachable: %w", err)
	}
	log.Println("[OK] Hardware database engine connection verified via ping")

	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		return nil, fmt.Errorf("database pragma configuration injection failure: %w", err)
	}
	log.Println("[INIT] Relational constraints and WAL mode performance pragmas injected")

	if err := createTables(ctx, db); err != nil {
		return nil, fmt.Errorf("database schema migration block: migration execution failed: %w", err)
	}
	log.Println("[OK] Idempotent application schema verification complete")

	return db, nil
}

// createTables applies the raw SQL schema embedded at compile-time contextually.
func createTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("database deployment block: failed to execute embedded schema: %w", err)
	}
	return nil
}

// ====================================================================
// -- ENGINE HYGIENE & STORAGE COMPACTION LEDGER --
// ====================================================================

// OptimizeDatabase captures physical file allocations, executes a context-aware
// VACUUM compaction loop, and flushes historical data records out of live indices.
func OptimizeDatabase(ctx context.Context, db *sql.DB) {
	log.Println("[IDLE] Storage Maintenance: Commencing engine hygiene optimization sweep...")

	// ----- PHASE 0: Data Pruning Ledger -----
	cutoff := time.Now().AddDate(0, 0, -14).Format("2006-01-02 15:04:05")

	pruneQuery := `DELETE FROM quest_completions WHERE completed_at < ?;`
	result, err := db.ExecContext(ctx, pruneQuery, cutoff)
	if err != nil {
		log.Printf("[ERROR] Storage Maintenance Warning: Data pruning ledger execution failed: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("[OK] Data Pruning Ledger successfully purged %d expired quest completions from the archive", rowsAffected)
		}
	}

	// ----- PHASE 1: Telemetry Phase -----
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/quests.db"
	}

	preInfo, err := os.Stat(dbPath)
	var preBytes int64
	if err == nil {
		preBytes = preInfo.Size()
		log.Printf("[REALTIME] Pre-compaction storage allocation: %d bytes", preBytes)
	}

	start := time.Now()

	var freePages int
	err = db.QueryRowContext(ctx, "PRAGMA freelist_count;").Scan(&freePages)
	if err != nil {
		log.Printf("[ERROR] Telemetry Engine unable to parse page freelist allocation metrics: %v", err)
	}

	if freePages > 100 {
		log.Printf("[REALTIME] Free page threshold exceeded (%d unallocated pages). Initiating disk space reclamation...", freePages)

		// ----- PHASE 2: Execution Phase -----
		_, err = db.ExecContext(ctx, "VACUUM;")
		if err != nil {
			log.Printf("[ERROR] Critical storage optimization sweep abort encountered: %v", err)
			return
		}
	} else {
		duration := time.Since(start)
		log.Printf("[OK] Storage structure optimized. Compaction skipped (Free pages: %d). Evaluation duration: %v", freePages, duration)
		return
	}

	duration := time.Since(start)

	// ----- PHASE 3: Verification Phase -----
	postInfo, err := os.Stat(dbPath)
	if err != nil {
		log.Printf("[ERROR] Telemetry Engine unable to verify post-compaction storage footprint: %v", err)
		return
	}

	postBytes := postInfo.Size()
	bytesSaved := preBytes - postBytes

	log.Printf("[OK] Storage engine compaction cycle finalized successfully in %v", duration)
	log.Printf("[REALTIME] Post-compaction allocation: %d bytes (Recovered: %d sectors/bytes)", postBytes, bytesSaved)
}
