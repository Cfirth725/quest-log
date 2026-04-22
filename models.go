package main

import (
	"time"
)

// QuestResponse represents the data sent to the UI, combining Quest and Category data.
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

// ----- Defines a Category -----
type Category struct {
	ID         int    `json:"id"`
	OwnerID    int    `json:"owner_id"`
	Name       string `json:"name"`
	ColorHex   string `json:"color_hex"`
	IsArchived bool   `json:"is_archived"`
}
