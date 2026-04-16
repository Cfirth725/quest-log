package main

import (
    "fmt"
	"database/sql"
	"context"
)

// GetActiveQuests fetches all pending quests for a user, including shared household quests.
// It joins the categories table to provide the frontend with the correct category name and color.
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

	// Pass context to the query to allow for graceful timeouts
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active quests: %w", err)
	}
	defer rows.Close()

	var activeQuests []QuestResponse

	for rows.Next() {
		var q QuestResponse
		// Scan maps the SQL columns directly to the struct fields in the exact order queried
		err := rows.Scan(
			&q.ID, &q.Title, &q.Difficulty, &q.BaseXP, &q.QuestType, &q.IsNonNegotiable, &q.Status, &q.CreatedAt,
			&q.CategoryName, &q.CategoryColor,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quest row: %w", err)
		}
		activeQuests = append(activeQuests, q)
	}

	// Always check for errors that occurred during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quest rows: %w", err)
	}

	return activeQuests, nil
}


// CompleteQuest handles the transactional logic of marking a task done,
// calculating XP, updating streaks, and writing to the historical ledger.
func CompleteQuest(ctx context.Context, db *sql.DB, questID int, completingUserID int) error {
	// BeginTx starts a transaction. If any step fails, we rollback to prevent partial data writes.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback. If the transaction is successfully committed later, this does nothing.
	defer tx.Rollback()

	// 1. Fetch current quest state to prevent double-dipping and get Base XP
	var currentStatus string
	var baseXP int

	err = tx.QueryRowContext(ctx, `
		SELECT status, base_xp
		FROM quests 
		WHERE id = ?`, questID).Scan(&currentStatus, &baseXP)
	if err != nil {
		return fmt.Errorf("failed to fetch quest details: %w", err)
	}

	if currentStatus == "Completed" {
		return fmt.Errorf("quest is already completed")
	}

	// 2. Fetch user's current dopamine streak for the multiplier
	var currentStreak int
	err = tx.QueryRowContext(ctx, `
		SELECT dopamine_streak 
		FROM users 
		WHERE id = ?`, completingUserID).Scan(&currentStreak)
	if err != nil {
		return fmt.Errorf("failed to fetch user streak: %w", err)
	}

	// 3. Calculate Final XP
	earnedXP := baseXP + currentStreak

	// 4. Update the Quest (Mark complete and log the timestamp)
	_, err = tx.ExecContext(ctx, `
		UPDATE quests 
		SET status = 'Completed', last_completed_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, questID)
	if err != nil {
		return fmt.Errorf("failed to update quest status: %w", err)
	}

	// 5. Write to the Immutable Ledger (For the Weekly Corral UI)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO quest_completions (quest_id, completed_by_user_id, xp_awarded) 
		VALUES (?, ?, ?)`, questID, completingUserID, earnedXP)
	if err != nil {
		return fmt.Errorf("failed to log quest completion: %w", err)
	}

	// 6. Reward the User
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET dopamine_streak = dopamine_streak + ? 
		WHERE id = ?`, earnedXP, completingUserID)
	if err != nil {
		return fmt.Errorf("failed to update user streak: %w", err)
	}

	// Commit permanently saves all the changes made in this transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}