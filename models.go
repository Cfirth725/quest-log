package main

import (
	"time"
)

// QuestResponse represents a flattened data structure for the UI.
// It joins Quest and Category data into a single object to simplify
// template rendering and JSON API delivery.
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

// Category defines the thematic grouping for quests.
// It includes metadata for UI rendering (ColorHex) and visibility state (IsArchived).
type Category struct {
	ID         int    `json:"id"`
	OwnerID    int    `json:"owner_id"` // 0 indicates a system-wide or shared category
	Name       string `json:"name"`
	ColorHex   string `json:"color_hex"`
	IsArchived bool   `json:"is_archived"`
}

// ForgeData holds all the reference data needed to populate the Quest Forge forms.
type ForgeData struct {
	Categories []Category
	Users      []User
}

// User represents a person in the household.
type User struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	DopamineStreak int    `json:"dopamine_streak"`
}

// Corral Related structs
type CorralSummary struct {
	TotalXP    int
	QuestCount int
	RecentWins []QuestCompletionRow
}

type QuestCompletionRow struct {
	Title        string
	CategoryName string
	ColorHex     string
	XPAwarded    int
	CompletedAt  string
}
