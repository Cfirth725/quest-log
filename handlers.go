package main

import (
	"log"
	"net/http"
	"strings"
)

// --- Quest Management Routes (The Forge) ---
// Route: GET /newquest
// Renders the quest creation interface with dynamic category data
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

// GET /settings
func handleSettings(w http.ResponseWriter, r *http.Request) {
	categories, err := GetCategories(DB)
	if err != nil {
		log.Printf("Internal Error: %v", err)
	}

	data := struct{ Categories []Category }{Categories: categories}
	RenderTemplate(w, "settings", data)
}

// POST /categories/create
func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	color := r.FormValue("color")

	if name == "" {
		http.Error(w, "Name required", http.StatusBadRequest)
		return
	}

	_, err := DB.Exec("INSERT INTO categories (name, color_hex) VALUES (?, ?)", name, color)
	if err != nil {
		log.Printf("DB Error: %v", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// POST /categories/delete
func handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("category_id")

	// THE SAFETY GUARD
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM quests WHERE category_id = ?", id).Scan(&count)
	if err != nil {
		log.Printf("DB Error: %v", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Conflict: Category has active quests.", http.StatusConflict)
		return
	}

	_, err = DB.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		log.Printf("DB Error: %v", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
