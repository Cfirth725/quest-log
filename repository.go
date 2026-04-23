package main

import (
	"context"
	"database/sql"
	"fmt"
)

// GetActiveQuests retrieves all pending tasks for a specific user, including
// shared household entries. It performs a LEFT JOIN with the categories table
// to ensure the UI has access to thematic metadata (naming and color-hex).
func GetActiveQuests(ctx context.Context, db *sql.DB, userID int) ([]QuestResponse, error) {
	query := `
		SELECT 
			q.id, q.title, q.difficulty, q.base_xp, q.quest_type, q.is_non_negotiable, q.status, q.created_at,
			c.name, c.color_hex
		FROM quests q
		LEFT JOIN categories c ON q.category_id = c.id
		WHERE (q.owner_id = ? OR q.owner_id = 0)
		AND q.status != 'Completed'
		ORDER BY q.is_non_negotiable DESC, q.created_at ASC
	`

	// Context-aware query execution to handle timeouts or client cancellations
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

	// Post-iteration error check to ensure stream was not interrupted
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quest rows: %w", err)
	}

	return activeQuests, nil
}

// CompleteQuest executes a high-integrity transaction to finalize a task.
// It calculates dynamic XP rewards based on user streaks and records the
// event in an immutable ledger for historical auditing (The Weekly Corral).
func CompleteQuest(ctx context.Context, db *sql.DB, questID int, completingUserID int) error {
	// Initialize a transaction to ensure atomic execution.
	// This prevents partial writes where a quest is marked done but no XP is awarded.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: transaction initiation failed: %w", err)
	}
	// Defensive defer: Rollback will execute unless the transaction is explicitly committed.
	defer tx.Rollback()

	// 1. Validation: Verify quest existence and prevent duplicate completion
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
		return fmt.Errorf("repository: quest [%d] already finalized")
	}

	// 2. Multiplier Logic: Retrieve current user streak from the persistent store
	var currentStreak int
	err = tx.QueryRowContext(ctx, `
		SELECT dopamine_streak 
		FROM users 
		WHERE id = ?`, completingUserID).Scan(&currentStreak)
	if err != nil {
		return fmt.Errorf("repository: user streak retrieval failed: %w", err)
	}

	// 3. XP Calculation Logic
	earnedXP := baseXP + currentStreak

	// 4. Update Quest State: Mark as 'Completed' and update the temporal record
	_, err = tx.ExecContext(ctx, `
		UPDATE quests 
		SET status = 'Completed', last_completed_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, questID)
	if err != nil {
		return fmt.Errorf("repository: quest status update failed: %w", err)
	}

	// 5. Immutable Ledger Write: Record completion for telemetry and reporting
	_, err = tx.ExecContext(ctx, `
		INSERT INTO quest_completions (quest_id, completed_by_user_id, xp_awarded) 
		VALUES (?, ?, ?)`, questID, completingUserID, earnedXP)
	if err != nil {
		return fmt.Errorf("repository: ledger entry failed: %w", err)
	}

	// 6. User Reward Application: Increment streak/XP values
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET dopamine_streak = dopamine_streak + ? 
		WHERE id = ?`, earnedXP, completingUserID)
	if err != nil {
		return fmt.Errorf("repository: user reward application failed: %w", err)
	}

	// Commit finalized state change to the database
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("repository: final transaction commit failed: %w", err)
	}

	return nil
}
