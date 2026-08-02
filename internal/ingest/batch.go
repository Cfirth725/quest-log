// ====================================================================
// -- INGESTION DOMAIN: TRANSACTIONAL BATCH INGESTION DRIVER --
// ====================================================================

// Package ingest coordinates external payload parsing, string sanitization,
// and schema verification before records hit the transactional DAO layer.
package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// ====================================================================
// -- DOMAIN STRUCTURES & TELEMETRY DTOs --
// ====================================================================

// IngestBatchResult provides execution metrics on the completed batch operation.
type IngestBatchResult struct {
	TotalProcessed    int
	QuestsMinted      int
	CategoriesCreated int
}

// ====================================================================
// -- TRANSACTIONAL BATCH EXECUTION ENGINE --
// ====================================================================

// ExecuteBatchIngestion processes a slice of ExtractedQuest records within a single atomic database transaction.
func ExecuteBatchIngestion(ctx context.Context, db *sql.DB, defaultOwnerID int, quests []ExtractedQuest) (IngestBatchResult, error) {
	log.Printf("[INIT] Initializing batch transaction sweep for %d quest record(s)...", len(quests))

	var result IngestBatchResult
	result.TotalProcessed = len(quests)

	if len(quests) == 0 {
		log.Printf("[IDLE] Empty payload array detected. Exiting batch process loop...")
		return result, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[ERROR] Transactional boundary fault: failed initializing batch context: %v", err)
		return result, fmt.Errorf("transaction fault: failed to initialize batch ingestion transaction: %w", err)
	}
	defer tx.Rollback()

	log.Printf("[REALTIME] Building category and user registry maps within active transaction scope...")

	// Cache category map (Name -> ID)
	categoryMap, err := loadCategoryMapTx(ctx, tx)
	if err != nil {
		log.Printf("[ERROR] System cache fault: category map construction aborted: %v", err)
		return result, fmt.Errorf("cache error: failed to build category registry map: %w", err)
	}

	// Cache user map (Name -> ID) for case-insensitive owner resolution
	userMap, err := loadUserMapTx(ctx, tx)
	if err != nil {
		log.Printf("[ERROR] System cache fault: user map construction aborted: %v", err)
		return result, fmt.Errorf("cache error: failed to build user registry map: %w", err)
	}

	log.Printf("[REALTIME] Processing %d payload records through category/owner resolution & SQL execution...", len(quests))

	for _, q := range quests {
		categoryID, created, err := resolveCategoryIDTx(ctx, tx, categoryMap, q.CategoryName)
		if err != nil {
			log.Printf("[ERROR] Category resolution fault on '%s': %v", q.CategoryName, err)
			return result, fmt.Errorf("resolution error for category '%s': %w", q.CategoryName, err)
		}
		if created {
			result.CategoriesCreated++
		}

		// Dynamically resolve owner (ID or Name case-insensitive)
		resolvedOwnerID := resolveOwnerID(q.OwnerRaw, userMap, defaultOwnerID)

		insertQuery := `
			INSERT INTO quests (
				title, category_id, difficulty, base_xp, is_non_negotiable, 
				status, owner_id, quest_type, repeat_interval_days, reset_day_of_week
			) VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?)`

		_, err = tx.ExecContext(ctx, insertQuery,
			q.Title, categoryID, q.Difficulty, q.BaseXP, q.IsNonNegotiable,
			resolvedOwnerID, q.QuestType, q.RepeatIntervalDays, q.ResetDayOfWeek,
		)
		if err != nil {
			log.Printf("[ERROR] DAO execution fault: failed inserting quest record '%s': %v", q.Title, err)
			return result, fmt.Errorf("db execution error inserting quest '%s': %w", q.Title, err)
		}

		result.QuestsMinted++
	}

	log.Printf("[REALTIME] Executing atomic commit on batch payload...")

	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] Transactional commit fault: batch quest import rejected: %v", err)
		return result, fmt.Errorf("transaction commit fault: failed committing batch quest import: %w", err)
	}

	log.Printf("[OK] Batch Ingestion Complete: Minted %d quests (%d new categories established)",
		result.QuestsMinted, result.CategoriesCreated)

	return result, nil
}

// ====================================================================
// -- CATEGORY & OWNER RESOLUTION DAO HELPER FUNCTIONS --
// ====================================================================

// loadCategoryMapTx fetches existing active categories and returns a case-insensitive lookup map.
func loadCategoryMapTx(ctx context.Context, tx *sql.Tx) (map[string]int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, LOWER(name) FROM categories WHERE is_archived = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		categoryMap[name] = id
	}
	return categoryMap, nil
}

// loadUserMapTx fetches active users and populates a case-insensitive name-to-ID lookup map.
func loadUserMapTx(ctx context.Context, tx *sql.Tx) (map[string]int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, LOWER(name) FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := map[string]int{
		"household": 0,
		"shared":    0,
		"home":      0,
	}

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		userMap[name] = id
	}
	return userMap, nil
}

// resolveOwnerID evaluates incoming owner input (int, float64, or string) against database users.
func resolveOwnerID(rawOwner interface{}, userMap map[string]int, defaultOwnerID int) int {
	if rawOwner == nil {
		return defaultOwnerID
	}

	switch v := rawOwner.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		clean := strings.TrimSpace(v)
		if clean == "" {
			return defaultOwnerID
		}

		// Try numeric string parse first ("1", "2", "0")
		if num, err := strconv.Atoi(clean); err == nil {
			return num
		}

		// Case-insensitive lookup against registered names
		lookup := strings.ToLower(clean)
		if matchedID, found := userMap[lookup]; found {
			return matchedID
		}
	}

	return defaultOwnerID
}

// resolveCategoryIDTx returns the ID for a category name, creating it on-the-fly if it does not exist.
func resolveCategoryIDTx(ctx context.Context, tx *sql.Tx, categoryMap map[string]int, categoryName string) (id int, created bool, err error) {
	cleanName := strings.TrimSpace(categoryName)
	if cleanName == "" {
		cleanName = "Uncategorized"
	}
	lookupKey := strings.ToLower(cleanName)

	if existingID, found := categoryMap[lookupKey]; found {
		return existingID, false, nil
	}

	log.Printf("[REALTIME] On-the-fly taxonomy minting triggered: Category '%s'", cleanName)

	// Category doesn't exist yet—mint it on-the-fly
	defaultColor := "#9F7AEA" // Default signature purple
	res, err := tx.ExecContext(ctx, "INSERT INTO categories (name, color_hex, owner_id) VALUES (?, ?, 1)", cleanName, defaultColor)
	if err != nil {
		return 0, false, fmt.Errorf("failed creating category '%s': %w", cleanName, err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, false, fmt.Errorf("failed retrieving new category ID for '%s': %w", cleanName, err)
	}

	intID := int(newID)
	categoryMap[lookupKey] = intID // Cache for remainder of transaction scope
	return intID, true, nil
}
