package internal

import (
	"database/sql"
	"log"
)

// RunMasterSpawner runs daily at 4:03 AM to check which quests need to be reactivated.
func RunMasterSpawner(db *sql.DB) {
	log.Println("--- Starting Master Spawner Cycle ---")

	// 1. Daily tasks check
	dailies, err := processDailies(db)
	if err != nil {
		log.Printf("ERROR in Dailies: %v", err)
	} else {
		log.Printf("SUCCESS: Spawned %d Daily quests.", dailies)
	}

	// 2. Repeating tasks check
	intervals, err := processIntervals(db)
	if err != nil {
		log.Printf("ERROR in Intervals: %v", err)
	} else {
		log.Printf("SUCCESS: Spawned %d Interval quests.", intervals)
	}

	// 3. Weekly tasks check
	weeklies, err := processWeeklies(db)
	if err != nil {
		log.Printf("ERROR in Weeklies: %v", err)
	} else {
		log.Printf("SUCCESS: Spawned %d Weekly quests.", weeklies)
	}

	log.Println("--- Master Spawner Cycle Complete ---")
}

// processDailies reactivates 'Daily' tasks that have not been completed
// within the current daily boundary (resetting at 04:00 local time).
// It uses an atomic subquery to prevent redundant activations.
func processDailies(db *sql.DB) (int, error) {
	// The NOT IN subquery protects against a midnight race condition.
	// By subtracting 4 hours first, anything completed between midnight and 3:59 AM
	// is correctly grouped with the previous calendar day's window.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Daily' 
		  AND status != 'active'
		  AND id NOT IN (
			  SELECT quest_id FROM quest_completions 
			  WHERE completed_at > datetime('now', '-4 hours', 'start of day', '+4 hours', 'localtime')
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
	// 1. Shifting backward by 4 hours standardizes both timestamps.
	// 2. 'start of day' strips off the hours/minutes/seconds so we can compare raw dates.
	// 3. CAST(... AS INTEGER) truncates fractional decimals to prevent timezone or daylight saving drift.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Repeating' 
		  AND status != 'active'
		  AND last_completed_at IS NOT NULL
		  AND (
			CAST(
				julianday('now', '-4 hours', 'start of day') - 
				julianday(last_completed_at, '-4 hours', 'start of day') 
			AS INTEGER)
		  ) >= repeat_interval_days;`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// processWeeklies handles tasks that reset once a week (e.g., weekly chores).
// It checks if the task has been completed during the current calendar week.
func processWeeklies(db *sql.DB) (int, error) {
	// 1. '%Y-%W' calculates the Year and Week Number (e.g., '2026-22').
	// 2. We shift 'now' and 'completed_at' back by 4 hours to respect the 4 AM game-day line.
	// 3. If the current week string doesn't match the last completed week string, it resets.
	query := `
		UPDATE quests 
		SET status = 'active' 
		WHERE quest_type = 'Weekly' 
		  AND status != 'active'
		  -- Check if today matches the target reset day (1 = Monday)
		  AND CAST(strftime('%w', 'now', '-4 hours') AS INTEGER) = reset_day_of_week
		  AND id NOT IN (
			  SELECT quest_id FROM quest_completions 
			  WHERE strftime('%Y-%W', completed_at, '-4 hours') = strftime('%Y-%W', 'now', '-4 hours')
		  );`

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
