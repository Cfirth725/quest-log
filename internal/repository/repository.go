// Package repository coordinates relational data access, transaction scoping,
// and immutable ledger operations for the active database engine.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ====================================================================
// -- BOUNTY BOARD (ACTIVE TRAFFIC) DAO LAYER --
// ====================================================================

// GetActiveQuests retrieves all non-deleted, actionable tasks for a specific user.
func GetActiveQuests(ctx context.Context, db *sql.DB, userID int, momentumMode bool) ([]QuestResponse, error) {
	momentumFilter := ""
	if momentumMode {
		momentumFilter = "AND q.is_non_negotiable = 1"
	}

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
			category_name ASC,
			q.is_non_negotiable DESC,
			q.created_at ASC;
	`, momentumFilter)

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("dao failure: GetActiveQuests query failed: %w", err)
	}
	defer rows.Close()

	var activeQuests []QuestResponse
	for rows.Next() {
		var q QuestResponse
		err := rows.Scan(
			&q.ID, &q.Title, &q.Difficulty, &q.BaseXP, &q.QuestType, &q.IsNonNegotiable, &q.Status, &q.CreatedAt,
			&q.CategoryName, &q.CategoryColor,
		)
		if err != nil {
			return nil, fmt.Errorf("dao failure: scanning quest row: %w", err)
		}
		activeQuests = append(activeQuests, q)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("dao failure: row streaming broken mid-process: %w", err)
	}

	return activeQuests, nil
}

// CompleteQuest executes an atomic database transaction to lock a quest, append
// ledger rewards, record histories, and increment user engagement streaks.
func CompleteQuest(ctx context.Context, db *sql.DB, questID int, completingUserID int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("dao transactional fault: transaction failed to open: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	var baseXP int

	err = tx.QueryRowContext(ctx, "SELECT status, base_xp FROM quests WHERE id = ? FOR UPDATE", questID).Scan(&currentStatus, &baseXP)
	if err != nil {
		return fmt.Errorf("dao safety check block: verification query failed: %w", err)
	}

	if currentStatus == "Completed" {
		return nil
	}

	var currentStreak int
	err = tx.QueryRowContext(ctx, "SELECT dopamine_streak FROM users WHERE id = ?", completingUserID).Scan(&currentStreak)
	if err != nil {
		return fmt.Errorf("dao context block: profile metric retrieval failed: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE quests SET status = 'Completed', last_completed_at = DATETIME('now', 'localtime') WHERE id = ?", questID)
	if err != nil {
		return fmt.Errorf("dao modification error: quest state write aborted: %w", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO quest_completions (quest_id, completed_by_user_id, xp_awarded) VALUES (?, ?, ?)", questID, completingUserID, baseXP)
	if err != nil {
		return fmt.Errorf("dao ledger fault: asset insertion denied on immutable registry: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET dopamine_streak = dopamine_streak + 1 WHERE id = ?", completingUserID)
	if err != nil {
		return fmt.Errorf("dao modification error: user profile streak gain rejected: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("dao critical boundary: write transaction execution commit rejected: %w", err)
	}

	return nil
}

// CreateQuest handles the server-side sanitized write injection for new quest entries.
func CreateQuest(ctx context.Context, db *sql.DB, title string, categoryID int, difficulty int, xp int, nonNegotiable int, ownerID int, qType string, interval sql.NullInt64, resetDay int) error {
	query := `INSERT INTO quests (title, category_id, difficulty, base_xp, is_non_negotiable, status, owner_id, quest_type, repeat_interval_days, reset_day_of_week)
        VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?)`

	_, err := db.ExecContext(ctx, query, title, categoryID, difficulty, xp, nonNegotiable, ownerID, qType, interval, resetDay)
	return err
}

// ====================================================================
// -- THE CHRONICLE (HISTORICAL DATA ROLLUPS) DAO LAYER --
// ====================================================================

// ChronicleCompletedQuests sweeps and soft-archives all completed One-Time quests.
func ChronicleCompletedQuests(ctx context.Context, db *sql.DB) (int64, error) {
	query := `
		UPDATE quests 
		SET status = 'Archived' 
		WHERE status = 'Completed' 
		AND quest_type = 'One-Time' 
		AND (owner_id = 1 OR owner_id = 0);`

	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("dao structural update error: chronicler archiving sweep failed: %w", err)
	}
	return result.RowsAffected()
}

// GetWeeklySummary queries total XP allocations and aggregates itemized wins.
func GetWeeklySummary(ctx context.Context, db *sql.DB, userID int) (ChronicleSummary, error) {
	var summary ChronicleSummary

	err := db.QueryRowContext(ctx, `
		SELECT 
			COUNT(id), 
			COALESCE(SUM(xp_awarded), 0) 
		FROM quest_completions 
		WHERE completed_by_user_id = ? 
		AND datetime(completed_at) >= datetime('now', '-7 days', 'localtime')`, userID).Scan(&summary.QuestCount, &summary.TotalXP)

	if err != nil {
		return summary, fmt.Errorf("dao aggregation block: weekly volume compilation failed: %w", err)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT q.title, c.name, c.color_hex, qc.xp_awarded, qc.completed_at
		FROM quest_completions qc
		JOIN quests q ON qc.quest_id = q.id
		JOIN categories c ON q.category_id = c.id
		WHERE qc.completed_by_user_id = ?
		AND datetime(qc.completed_at) >= datetime('now', '-7 days', 'localtime')
		ORDER BY qc.completed_at DESC`, userID)

	if err != nil {
		return summary, fmt.Errorf("dao query fault: weekly itemization parsing failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r QuestCompletionRow
		err := rows.Scan(&r.Title, &r.CategoryName, &r.ColorHex, &r.XPAwarded, &r.CompletedAt)
		if err != nil {
			return summary, fmt.Errorf("dao stream parsing failure: scanning victory logs: %w", err)
		}
		summary.RecentWins = append(summary.RecentWins, r)
	}

	return summary, nil
}

// GenerateWeeklyChronicleReport parses metric logs to compute completion split dynamics.
func GenerateWeeklyChronicleReport(ctx context.Context, db *sql.DB) (*OperationalReport, error) {
	report := &OperationalReport{
		EndDate:   time.Now(),
		StartDate: time.Now().AddDate(0, 0, -7),
	}

	splitQuery := `
		SELECT 
			COUNT(CASE WHEN q.quest_type = 'One-Time' THEN 1 END) as one_time_count,
			COUNT(CASE WHEN q.quest_type != 'One-Time' THEN 1 END) as recurring_count
		FROM quest_completions qc
		JOIN quests q ON qc.quest_id = q.id
		WHERE qc.completed_at >= datetime('now', '-7 days', '-4 hours')
		  AND q.deleted_at IS NULL`

	err := db.QueryRowContext(ctx, splitQuery).Scan(&report.OneTimeCompleted, &report.RecurringCompleted)
	if err != nil {
		return nil, fmt.Errorf("dao analytical crash: high-level balance computation failed: %w", err)
	}

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
		ORDER BY execution_count DESC, category_name ASC;`

	rows, err := db.QueryContext(ctx, habitQuery)
	if err != nil {
		return nil, fmt.Errorf("dao analytical crash: interval count computation failure: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h HabitFrequency
		if err := rows.Scan(&h.Title, &h.CategoryName, &h.ExecutionCount); err != nil {
			return nil, err
		}
		report.HabitExecutionLog = append(report.HabitExecutionLog, h)
	}

	return report, nil
}

// ====================================================================
// -- STATE MACHINE MODIFICATIONS & TAXONOMY MANAGEMENT --
// ====================================================================

// SoftDeleteQuest marks a task record as retired by appending a timestamp token.
func SoftDeleteQuest(ctx context.Context, db *sql.DB, id int) error {
	_, err := db.ExecContext(ctx, "UPDATE quests SET deleted_at = datetime('now', 'localtime') WHERE id = ?", id)
	return err
}

// DowngradeToOneTime forces an item's cycle parameters down to simple single lifecycle execution rules.
func DowngradeToOneTime(ctx context.Context, db *sql.DB, id int) error {
	_, err := db.ExecContext(ctx, "UPDATE quests SET quest_type = 'One-Time' WHERE id = ?", id)
	return err
}

// InsertCategory adds a new relational taxonomy tracking item.
func InsertCategory(ctx context.Context, db *sql.DB, name string, color string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO categories (name, color_hex) VALUES (?, ?)", name, color)
	return err
}

// GetQuestCountByCategory runs a safety sweep to find downstream blocking dependencies before asset changes.
func GetQuestCountByCategory(ctx context.Context, db *sql.DB, id string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM quests WHERE category_id = ? AND deleted_at IS NULL", id).Scan(&count)
	return count, err
}

// DeleteCategoryByID applies a final data drop on a custom taxonomy object.
func DeleteCategoryByID(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
	return err
}

// GetCategories outputs all functional categories unassigned to archival flags.
func GetCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, owner_id, name, color_hex, is_archived FROM categories WHERE is_archived = 0 ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.OwnerID, &c.Name, &c.ColorHex, &c.IsArchived); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetUsers captures active user metric arrays.
func GetUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, dopamine_streak FROM users ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DopamineStreak); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
