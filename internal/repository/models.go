// Package repository coordinates relational data access, transaction scoping,
// and immutable ledger operations for the active database engine.
package repository

import (
	"database/sql"
	"time"
)

// ====================================================================
// -- DATA TRANSPORT & TRANSFORMATION ENTITIES --
// ====================================================================

// QuestResponse represents a flattened data structure tailored for the user interface.
// It aggregates Quest and Category attributes to minimize database round-trips
// and simplify frontend template logic.
type QuestResponse struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	CategoryName    string    `json:"category_name"`
	CategoryColor   string    `json:"category_color"`
	Difficulty      int       `json:"difficulty"` // 1: Easy, 2: Medium, 3: Hard
	BaseXP          int       `json:"base_xp"`
	QuestType       string    `json:"quest_type"`
	IsNonNegotiable bool      `json:"is_non_negotiable"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// ForgeData serves as a transport object for populating quest creation forms.
// It bundles available categories and users to provide a unified context for the UI.
type ForgeData struct {
	Categories []Category
	Users      []User
}

// SettingsPageData aggregates management layouts for the administrative view.
type SettingsPageData struct {
	Categories []Category
	Quests     []QuestResponse
}

// ====================================================================
// -- TAXONOMY & ACCOUNT LAYER IDENTITY ENTITIES --
// ====================================================================

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

// User represents a system participant. It tracks identity and behavioral
// metrics, such as the DopamineStreak used for gamified reward multipliers.
type User struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	DopamineStreak int    `json:"dopamine_streak"`
}

// ====================================================================
// -- CHRONICLE LEDGER & ANALYTICS SUMMARY ENTITIES --
// ====================================================================

// ChronicleSummary provides an aggregated view of user performance metrics
// over a rolling window (typically 7 days), used for the "The Chronicle" review engine.
type ChronicleSummary struct {
	TotalXP    int
	QuestCount int
	RecentWins []QuestCompletionRow
	Report     *OperationalReport
}

// QuestCompletionRow describes a single historical record of a finished task.
// It is primarily used for displaying immutable ledger entries in the summary views.
type QuestCompletionRow struct {
	Title        string
	CategoryName string
	ColorHex     string
	XPAwarded    int
	CompletedAt  time.Time
}

// OperationalReport holds high-level split metrics and fine-grained habit counts
// gathered from the weekly compilation sequence.
type OperationalReport struct {
	StartDate          time.Time
	EndDate            time.Time
	OneTimeCompleted   int
	RecurringCompleted int
	HabitExecutionLog  []HabitFrequency
}

// HabitFrequency tracks how many times a specific recurring task was completed in the window.
type HabitFrequency struct {
	Title          string
	CategoryName   string
	ExecutionCount int
}

// ====================================================================
// -- CORE TRANSACTIONAL QUEST ENGINE MODELS --
// ====================================================================

// Quest is the fundamental domain model for a task within the system of record.
// It supports multiple lifecycles (One-Time, Daily, Repeating, Weekly) and uses
// SQL null-types to handle optional temporal data from the SQLite engine.
type Quest struct {
	ID                 int           `json:"id"`
	OwnerID            int           `json:"owner_id"`
	CategoryID         int           `json:"category_id"`
	Title              string        `json:"title"`
	Difficulty         int           `json:"difficulty"`
	BaseXP             int           `json:"base_xp"`
	IsNonNegotiable    bool          `json:"is_non_negotiable"`
	Status             string        `json:"status"`
	QuestType          string        `json:"quest_type"`
	RepeatIntervalDays sql.NullInt64 `json:"repeat_interval_days"`
	ResetDayOfWeek     int           `json:"reset_day_of_week"`
	CreatedAt          time.Time     `json:"created_at"`
	LastCompletedAt    sql.NullTime  `json:"last_completed_at"`
	DeletedAt          sql.NullTime  `json:"deleted_at"`
}
