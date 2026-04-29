package main

import (
	"log"
	"net/http"
	"strings"
)

// --- Quest Management Routes (The Forge) ---
// handleNewQuest serves the 'Quest Forge' creation interface.
// It aggregates necessary reference data, including categories and users,
// to populate dynamic form elements.
func handleNewQuest(w http.ResponseWriter, r *http.Request) {
	categories, err := GetCategories(DB)
	if err != nil {
		log.Printf("Internal Error: Failed to fetch categories for The Forge: %v", err)
	}

	users, err := GetUsers(DB)
	if err != nil {
		log.Printf("Internal Error: Failed to fetch users: %v", err)
	}
	data := ForgeData{
		Categories: categories,
		Users:      users,
	}

	RenderTemplate(w, "new_quest", data)
}

// handleSettings renders the administrative dashboard.
// It allows users to manage the system's organizational hierarchy,
// specifically the category and user definitions.
func handleSettings(w http.ResponseWriter, r *http.Request) {
	categories, err := GetCategories(DB)
	if err != nil {
		log.Printf("Internal Error: %v", err)
	}

	data := struct{ Categories []Category }{Categories: categories}
	RenderTemplate(w, "settings", data)
}

// handleCreateCategory processes submissions for new thematic groupings.
// It enforces basic input sanitization and ensures categorical names
// meet the system's non-empty string requirements.
func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	color := r.FormValue("color")

	if name == "" {
		http.Error(w, "Validation Error: Category name is required.", http.StatusBadRequest)
		return
	}

	_, err := DB.Exec("INSERT INTO categories (name, color_hex) VALUES (?, ?)", name, color)
	if err != nil {
		log.Printf("Database Error: Failed to create category: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// handleDeleteCategory facilitates the removal of organizational categories.
// It implements a 'Safety Guard' to prevent the deletion of categories
// that still have associated quests, preserving relational integrity.
func handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("category_id")

	// RELATIONAL INTEGRITY CHECK: Verify no orphaned quests will be created.
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM quests WHERE category_id = ?", id).Scan(&count)
	if err != nil {
		log.Printf("Database Error: Integrity check failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		// HTTP 409 Conflict is the standard response for state violations.
		http.Error(w, "Conflict: Cannot delete category with active dependencies.", http.StatusConflict)
		return
	}

	_, err = DB.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		log.Printf("Database Error: Failed to remove category: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// ----- Corral Logic -----
// handleCorralQuests triggers the bulk archival process for finished tasks.
// This endpoint is the primary mechanism for clearing the pasture while
// maintaining a permanent record in the completion ledger.
func handleCorralQuests(w http.ResponseWriter, r *http.Request) {
	// Strict Method Enforcement: Idempotent changes must use POST.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count, err := CorralCompletedQuests(DB)
	if err != nil {
		log.Printf("Corral Error: %v", err)
		http.Error(w, "Failed to corral quests", http.StatusInternalServerError)
		return
	}

	log.Printf("Success: %d quests moved to The Corral.", count)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleViewCorral renders the historical reporting dashboard.
// It retrieves the Weekly Summary metrics to provide users with a
// bird's-eye view of their recent accomplishments.
func handleViewCorral(w http.ResponseWriter, r *http.Request) {
	// Context is passed from the request to allow for cancellation propogation.
	// For now, using User ID 1
	summary, err := GetWeeklySummary(r.Context(), DB, 1)
	if err != nil {
		log.Printf("Corral View Error: %v", err)
		http.Error(w, "Failed to load the Corral", http.StatusInternalServerError)
		return
	}

	RenderTemplate(w, "corral", summary)
}
