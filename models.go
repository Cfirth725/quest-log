package main

import (
	"database/sql"
	"time"
)

// QuestResponse represents a flattened data structure tailored for the user interface.
// It aggregates Quest and Category attributes to minimize database round-trips
// and simplify frontend template logic.
type QuestResponse struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	CategoryName    string    `json:"category_name"`
	CategoryColor   string    `json:"category_color"`
	Difficulty      int       `json:"difficulty"`
	BaseXP          int       `json:"base_xp"`
	QuestType       string    `json:"quest_type"`
	IsNonNegotiable bool      `json:"is_non_negotiable"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// Category defines the thematic grouping used to organize tasks.
// It contains metadata for visual rendering and supports soft-archiving
// to maintain historical data integrity.
type Category struct {
	ID         int    `json:"id"`
	OwnerID    int    `json:"owner_id"` // ID 0 is reserved for shared/household categories.
	Name       string `json:"name"`
	ColorHex   string `json:"color_hex"`
	IsArchived bool   `json:"is_archived"`
}

// ForgeData serves as a transport object for populating quest creation forms.
// It bundles available categories and users to provide a unified context for the UI.
type ForgeData struct {
	Categories []Category
	Users      []User
}

// User represents a system participant. It tracks identity and behavioral
// metrics, such as the DopamineStreak used for gamified reward multipliers.
type User struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	DopamineStreak int    `json:"dopamine_streak"`
}

// CorralSummary provides an aggregated view of user performance metrics
// over a rolling window (typically 7 days), used for the "Weekly Corral" report.
type CorralSummary struct {
	TotalXP    int
	QuestCount int
	RecentWins []QuestCompletionRow
}

// QuestCompletionRow describes a single historical record of a finished task.
// It is primarily used for displaying immutable ledger entries in the summary views.
type QuestCompletionRow struct {
	Title        string
	CategoryName string
	ColorHex     string
	XPAwarded    int
	CompletedAt  string
}

// Quest is the fundamental domain model for a task within the system.
// It supports multiple lifecycles (One-Time, Daily, Repeating) and uses
// SQL null-types to handle optional temporal data from the SQLite engine.
type Quest struct {
	ID                 int           `json:"id"`
	CategoryID         int           `json:"category_id"`
	Title              string        `json:"title"`
	Difficulty         int           `json:"difficulty"` // Normalized effort levels: 1 (Duck), 2 (Sheep), 3 (Cow).
	BaseXP             int           `json:"base_xp"`    // Economy values: 1, 5, or 10 XP.
	IsNonNegotiable    bool          `json:"is_non_negotiable"`
	Status             string        `json:"status"`
	QuestType          string        `json:"quest_type"` // Options: 'One-Time', 'Daily', 'Repeating'.
	RepeatIntervalDays sql.NullInt64 `json:"repeat_interval_days"`
	CreatedAt          string        `json:"created_at"`
	LastCompletedAt    sql.NullTime  `json:"last_completed_at"`
}

// SpawnerResult captures the telemetry of a background automation cycle.
// It tracks the volume of successfully reactivated quests and logs any
// transactional errors encountered.
type SpawnerResult struct {
	QuestsSpawned int
	Errors        []error
}

// Setting Page Data
type SettingsPageData struct {
	Categories []Category
	Quests     []QuestResponse
}
