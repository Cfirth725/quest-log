package main

import (
	"database/sql"
	"log"
)

// RunMasterSpawner orchestrates the automated reactivation of tasks.
// It serves as the central execution point for both daily resets and
// interval-based recurring tasks, logging telemetry for each cycle.
func RunMasterSpawner(db *sql.DB) {
	log.Println("--- Starting Master Spawner Cycle ---")

	// Trigger the 4:03 AM hard-reset logic for Daily tasks.
	dailies, err := processDailies(db)
	if err != nil {
		log.Printf("ERROR in Dailies: %v", err)
	} else {
		log.Printf("SUCCESS: Spawned %d Daily quests.", dailies)
	}

	// Trigger the elapsed-time check for Repeating tasks.
	intervals, err := processIntervals(db)
	if err != nil {
		log.Printf("ERROR in Intervals: %v", err)
	} else {
		log.Printf("SUCCESS: Spawned %d Interval quests.", intervals)
	}
	log.Println("--- Master Spawner Cycle Complete ---")
}

// processDailies reactivates 'Daily' tasks that have not been completed
// within the current daily boundary (resetting at 04:00 local time).
// It uses an atomic subquery to prevent redundant activations.
func processDailies(db *sql.DB) (int, error) {
	// The +4 hours offset ensures the 'Daily' window aligns with the
	// early morning reset rather than UTC midnight.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Daily' 
		  AND status != 'active'
		  AND id NOT IN (
			  SELECT quest_id FROM quest_completions 
			  WHERE completed_at > datetime('now', 'start of day', '+4 hours')
		  );`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// processIntervals evaluates 'Repeating' tasks against their specific
// recurrence windows. It calculates the delta between the current time
// and the last completion record using Julian Day conversion.
func processIntervals(db *sql.DB) (int, error) {
	// julianday() provides a high-precision decimal count of days,
	// allowing for accurate comparison against the repeat_interval_days column.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Repeating' 
		  AND status != 'active'
		  AND (julianday('now') - julianday(last_completed_at)) >= repeat_interval_days;`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
