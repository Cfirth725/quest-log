package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GetActiveQuests retrieves all actionable tasks for a specific user.
// If momentumMode is enabled, it further restricts the result set to only
// show 'Non-Negotiable' priority tasks to reduce cognitive load.
func GetActiveQuests(ctx context.Context, db *sql.DB, userID int, momentumMode bool) ([]QuestResponse, error) {
	// 1. Define the optional filter
	momentumFilter := ""
	if momentumMode {
		momentumFilter = "AND q.is_non_negotiable = 1"
	}

	// 2. Inject the category-first sorting rules into the SQL query template
	query := fmt.Sprintf(`
		SELECT 
			q.id, q.title, q.difficulty, q.base_xp, q.quest_type, q.is_non_negotiable, q.status, q.created_at,
			COALESCE(c.name, 'Uncategorized') AS category_name, 
			COALESCE(c.color_hex, '#4A5568') AS color_hex
		FROM quests q
		LEFT JOIN categories c ON q.category_id = c.id
		WHERE (q.owner_id = ? OR q.owner_id = 0)
		  AND q.deleted_at IS NULL
		  %s 
		  AND (
			q.status = 'active' 
			OR (q.status = 'Completed' AND q.quest_type = 'One-Time')
		  )
		ORDER BY 
			CASE WHEN q.status = 'active' THEN 0 ELSE 1 END ASC,
			category_name ASC,              -- Dynamic Category Grouping Cluster
			q.is_non_negotiable DESC,       -- Pin non-negotiables inside that group
			q.created_at DESC
	`, momentumFilter)

	// Using QueryContext to ensure database operations respect application lifecycle
	// and client-side timeouts.
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active quests: %w", err)
	}
	defer rows.Close()

	var activeQuests []QuestResponse

	for rows.Next() {
		var q QuestResponse
		// Direct scanning into struct fields; order must precisely match the SELECT clause
		err := rows.Scan(
			&q.ID, &q.Title, &q.Difficulty, &q.BaseXP, &q.QuestType, &q.IsNonNegotiable, &q.Status, &q.CreatedAt,
			&q.CategoryName, &q.CategoryColor,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quest row: %w", err)
		}
		activeQuests = append(activeQuests, q)
	}

	// Always check rows.Err after iteration to detect issues that occurred mid-stream.
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quest rows: %w", err)
	}

	return activeQuests, nil
}

// CompleteQuest executes a high-integrity transaction to finalize a task.
// It enforces state transitions, calculates XP rewards based on user data,
// and records an immutable ledger entry for historical auditing and reporting.
func CompleteQuest(ctx context.Context, db *sql.DB, questID int, completingUserID int) error {
	// Transactions are used to ensure atomicity; if any step fails, the entire
	// completion event is rolled back to prevent data corruption.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: transaction initiation failed: %w", err)
	}
	defer tx.Rollback()

	// 1. Integrity Check: Ensure the quest exists and is not already finalized.
	var currentStatus string
	var baseXP int

	err = tx.QueryRowContext(ctx, `
		SELECT status, base_xp
		FROM quests 
		WHERE id = ?`, questID).Scan(&currentStatus, &baseXP)
	if err != nil {
		return fmt.Errorf("repository: state verification failed: %w", err)
	}

	if currentStatus == "Completed" {
		return nil // Avoid redundant processing if already completed.
	}

	// 2. User Context: Retrieve current metrics for reward calculation.
	var currentStreak int
	err = tx.QueryRowContext(ctx, `
		SELECT dopamine_streak 
		FROM users 
		WHERE id = ?`, completingUserID).Scan(&currentStreak)
	if err != nil {
		return fmt.Errorf("repository: user streak retrieval failed: %w", err)
	}

	// 3. Economy Logic: Apply multipliers or streaks to the base XP here in the future.
	earnedXP := baseXP

	// 4. State Transition: Move quest to 'Completed' and update temporal metadata.
	// This timestamp is critical for the Master Spawner's interval logic.
	_, err = tx.ExecContext(ctx, `
		UPDATE quests 
		SET status = 'Completed', last_completed_at = DATETIME('now', 'localtime') 
		WHERE id = ?`, questID)
	if err != nil {
		return fmt.Errorf("repository: quest status update failed: %w", err)
	}

	// 5. Audit Trail: Insert completion record into the immutable ledger.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO quest_completions (quest_id, completed_by_user_id, xp_awarded) 
		VALUES (?, ?, ?)`, questID, completingUserID, earnedXP)
	if err != nil {
		return fmt.Errorf("repository: ledger entry failed: %w", err)
	}

	// 6. Reward Application: Commit the XP gains to the user's persistent record.
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET dopamine_streak = dopamine_streak + 1 
		WHERE id = ?`, completingUserID)
	if err != nil {
		return fmt.Errorf("repository: user reward application failed: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("repository: final transaction commit failed: %w", err)
	}

	return nil
}

// ArchiveCompletedQuests transitions 'One-Time' tasks from 'Completed' to 'Archived'.
// This serves as the "Weekly Reset" mechanism to declutter the active pasture
// while preserving 'Daily' and 'Repeating' lifecycles for the Master Spawner.
func ArchiveCompletedQuests(ctx context.Context, db *sql.DB, userID int) (int64, error) {
	query := `
		UPDATE quests 
		SET status = 'Archived' 
		WHERE status = 'Completed' 
		AND quest_type = 'One-Time' 
		AND (owner_id = ? OR owner_id = 0)
	`
	result, err := db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("repository: failed to archive completed quests: %w", err)
	}

	return result.RowsAffected()
}

// GetWeeklySummary aggregates completion data over a rolling 7-day window.
// It provides a high-level overview of total XP and itemized wins for the
// user's "Weekly Corral" dashboard.
func GetWeeklySummary(ctx context.Context, db *sql.DB, userID int) (CorralSummary, error) {
	var summary CorralSummary

	// 1. Aggregate Totals: Calculate total volume and currency earned.
	err := db.QueryRowContext(ctx, `
    SELECT 
        COUNT(id), 
        COALESCE(SUM(xp_awarded), 0) 
    FROM quest_completions 
    WHERE completed_by_user_id = ? 
    AND datetime(completed_at) >= datetime('now', '-7 days', 'localtime')`, userID).Scan(&summary.QuestCount, &summary.TotalXP)

	if err != nil {
		return summary, fmt.Errorf("repository: summary aggregation failed: %w", err)
	}

	// 2. Itemized Ledger: Retrieve individual victory records with thematic metadata.
	rows, err := db.QueryContext(ctx, `
    	SELECT q.title, c.name, c.color_hex, qc.xp_awarded, qc.completed_at
    	FROM quest_completions qc
    	JOIN quests q ON qc.quest_id = q.id
    	JOIN categories c ON q.category_id = c.id
    	WHERE qc.completed_by_user_id = ?
    	AND datetime(qc.completed_at) >= datetime('now', '-7 days', 'localtime')
    	ORDER BY qc.completed_at DESC`, userID)

	if err != nil {
		return summary, fmt.Errorf("repository: win list query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r QuestCompletionRow
		err := rows.Scan(&r.Title, &r.CategoryName, &r.ColorHex, &r.XPAwarded, &r.CompletedAt)
		if err != nil {
			return summary, fmt.Errorf("repository: win scan failed: %w", err)
		}
		summary.RecentWins = append(summary.RecentWins, r)
	}

	return summary, nil
}

// SoftDeleteQuest marks a specific quest as archived by setting a timestamp.
func SoftDeleteQuest(ctx context.Context, db *sql.DB, id int) error {
	query := `UPDATE quests SET deleted_at = datetime('now', 'localtime') WHERE id = ?`

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository: failed to soft delete quest %d: %w", id, err)
	}

	return nil
}

// DowngradeToOneTime changes a repeating/daily quest into a one-time quest.
func DowngradeToOneTime(ctx context.Context, db *sql.DB, id int) error {
	query := `UPDATE quests SET quest_type = 'One-Time' WHERE id = ?`
	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository: failed to downgrade quest %d: %w", id, err)
	}
	return nil
}

// GenerateWeeklyCorralReport compiles metrics and execution counts for a 7-day rolling window
func GenerateWeeklyCorralReport(db *sql.DB) (*OperationalReport, error) {
	report := &OperationalReport{
		EndDate: time.Now(),
		// Looking back exactly 7 days, adjusted for your 4 AM game-day line
		StartDate: time.Now().AddDate(0, 0, -7),
	}

	// 1. Compute high-level operational splits using your quest_completions logs
	splitQuery := `
		SELECT 
			COUNT(CASE WHEN q.quest_type = 'One-Time' THEN 1 END) as one_time_count,
			COUNT(CASE WHEN q.quest_type != 'One-Time' THEN 1 END) as recurring_count
		FROM quest_completions qc
		JOIN quests q ON qc.quest_id = q.id
		WHERE qc.completed_at >= datetime('now', '-7 days', '-4 hours')
		  AND q.deleted_at IS NULL
	`

	err := db.QueryRow(splitQuery).Scan(
		&report.OneTimeCompleted,
		&report.RecurringCompleted,
	)
	if err != nil {
		return nil, err
	}

	// 2. Compute exact execution frequencies for recurring task loops
	habitQuery := `
		SELECT 
			q.title,
			COALESCE(c.name, 'Uncategorized') AS category_name,
			COUNT(qc.id) AS execution_count
		FROM quest_completions qc
		JOIN quests q ON qc.quest_id = q.id
		LEFT JOIN categories c ON q.category_id = c.id
		WHERE q.quest_type != 'One-Time'
		  AND qc.completed_at >= datetime('now', '-7 days', '-4 hours')
		  AND q.deleted_at IS NULL
		GROUP BY q.title, category_name
		ORDER BY execution_count DESC, category_name ASC
	`

	rows, err := db.Query(habitQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h HabitFrequency
		if err := rows.Scan(&h.Title, &h.CategoryName, &h.ExecutionCount); err != nil {
			return nil, err
		}
		report.HabitExecutionLog = append(report.HabitExecutionLog, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return report, nil
}
