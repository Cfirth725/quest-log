// Package database coordinates low-level infrastructure bindings, performance pragmas,
// and automated sector compaction scripts for the persistent storage engine.
package database

import (
	"database/sql"
	"log"
)

// ====================================================================
// -- AUTOMATED QUEST LIFECYCLE (THE MASTER SPAWNER) --
// ====================================================================

// RunMasterSpawner evaluates all non-active recurring tasks against their temporal
// constraints and systematically reactivates them for the upcoming daily cycle.
func RunMasterSpawner(db *sql.DB) {
	log.Println("[INIT] Starting Master Spawner background automation loop...")

	// 1. Daily tasks check
	dailies, err := processDailies(db)
	if err != nil {
		log.Printf("[ERROR] Master Spawner exception handling Daily quest matrix: %v", err)
	} else {
		log.Printf("[OK] Master Spawner successfully reactivated %d Daily contracts", dailies)
	}

	// 2. Repeating tasks check
	intervals, err := processIntervals(db)
	if err != nil {
		log.Printf("[ERROR] Master Spawner exception handling Interval quest matrix: %v", err)
	} else {
		log.Printf("[OK] Master Spawner successfully reactivated %d Interval contracts", intervals)
	}

	// 3. Weekly tasks check
	weeklies, err := processWeeklies(db)
	if err != nil {
		log.Printf("[ERROR] Master Spawner exception handling Weekly quest matrix: %v", err)
	} else {
		log.Printf("[OK] Master Spawner successfully reactivated %d Weekly contracts", weeklies)
	}

	log.Println("[IDLE] Master Spawner automation sweep finished. Entering listening pool.")
}

// ====================================================================
// -- CADENCE EVALUATION SUB-ENGINES --
// ====================================================================

// processDailies reactivates 'Daily' tasks that have not been completed
// within the current daily boundary (resetting strictly at 04:00 AM EDT).
func processDailies(db *sql.DB) (int, error) {
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Daily' 
		  AND status != 'active'
		  AND id NOT IN (
			  SELECT quest_id FROM quest_completions 
			  WHERE completed_at >= datetime('now', '-4 hours', 'start of day', '+4 hours')
		  );`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// processIntervals evaluates 'Repeating' tasks against their specific
// recurrence windows using integer day differences.
func processIntervals(db *sql.DB) (int, error) {
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Repeating' 
		  AND status != 'active'
		  AND last_completed_at IS NOT NULL
		  AND (
			ROUND(
				julianday('now', '-4 hours', 'start of day') - 
				julianday(datetime(last_completed_at, '-4 hours'), 'start of day')
			)
		  ) >= repeat_interval_days;`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// / processWeeklies handles tasks that reset once a week on their specified reset_day_of_week.
func processWeeklies(db *sql.DB) (int, error) {
	// 1. Matches today's day of the week (0 = Sunday, 1 = Monday, 2 = Tuesday, etc.)
	// 2. Ensures the quest hasn't already been completed in the last 6 days.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Weekly' 
		  AND status != 'active'
		  AND CAST(strftime('%w', 'now', '-4 hours') AS INTEGER) = reset_day_of_week
		  AND id NOT IN (
			  SELECT quest_id FROM quest_completions 
			  WHERE completed_at >= datetime('now', '-4 hours', '-6 days')
		  );`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
