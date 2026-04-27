package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Experience constants defining the reward economy for quest tiers.
const (
	XP_DUCK  = 1
	XP_SHEEP = 5
	XP_COW   = 10
)

// RenderTemplate handles the assembly and execution of HTML templates,
// incorporating the base layout and localized partials.
func RenderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	basePath := "templates/layouts/base.html"
	pagePath := fmt.Sprintf("templates/%s.html", tmplName)

	// Initialize template with base layout and the specific view
	tmpl, err := template.ParseFiles(basePath, pagePath)
	if err != nil {
		log.Printf("Internal Error: Failed to parse template files: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Load optional partials for quest card rendering
	_, err = tmpl.ParseGlob("templates/partials/*.html")
	if err != nil {
		log.Printf("Partial Load Warning: %v", err)
	}

	// Execute the "base" template which acts as the entry point
	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("Execution Error: Failed to render template: %v", err)
	}
}

// ViewPastureHandler retrieves active quests and renders the primary dashboard view.
func ViewPastureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Fetching with CurrentUserID = 1 (System Owner)
		activeQuests, err := GetActiveQuests(r.Context(), db, 1)
		if err != nil {
			log.Printf("Database Error: Failed to retrieve active quests: %v", err)
			http.Error(w, "Failed to load quests", http.StatusInternalServerError)
			return
		}

		RenderTemplate(w, "pasture", activeQuests)
	}
}

// ----- New Quest Logic -----
// handleCreateQuest processes the 'Quest Forge' form submission, applying
// input validation and calculating reward values before persistence.
func handleCreateQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Form Error: Failed to parse quest submission: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract form inputs
	title := r.FormValue("title")
	categoryID := r.FormValue("category_id")
	difficulty := r.FormValue("difficulty")
	ownerIDStr := r.FormValue("owner_id")

	// Convert and Validate owner_id
	ownerID, err := strconv.Atoi(ownerIDStr)
	if err != nil {
		log.Printf("Security Alert: Non-integer owner_id received: %s", ownerIDStr)
		http.Error(w, "Invalid user assignment format.", http.StatusBadRequest)
		return
	}

	// --- OWNER SANITY CHECK ---
	// Rule: ID must be 0 (Household) or a positive integer existing in our system.
	// !! For now, since we only have IDs 1 and 2, we check if it's within a valid range.
	if ownerID < 0 || ownerID > 2 {
		log.Printf("Security Alert: Unauthorized owner_id attempt: %d", ownerID)
		http.Error(w, "Unauthorized User Assignment.", http.StatusForbidden)
		return
	}

	// --- GHOST GUARD: Input Sanitization ---
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		log.Println("Validation Warning: Blocked empty quest title submission.")
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	// --- HARD-CODED ECONOMY: XP Calculation ---
	var calculatedXP int
	switch difficulty {
	case "1":
		calculatedXP = XP_DUCK
	case "2":
		calculatedXP = XP_SHEEP
	case "3":
		calculatedXP = XP_COW
	default:
		log.Printf("Security Alert: Invalid difficulty value received: %s", difficulty)
		http.Error(w, "Invalid difficulty level.", http.StatusBadRequest)
		return
	}

	// --- PRIORITY SHIELD: Triage Handling ---
	priorityValue := r.FormValue("is_non_negotiable")
	isNonNegotiable := 0
	if priorityValue == "1" {
		isNonNegotiable = 1
	}

	// Database Injection
	query := `INSERT INTO quests (title, category_id, difficulty, base_xp, is_non_negotiable, status, owner_id, quest_type)
        VALUES (?, ?, ?, ?, ?, 'active', ?, 'One-Time')`

	_, err = DB.Exec(query, cleanTitle, categoryID, difficulty, calculatedXP, isNonNegotiable, ownerID)
	if err != nil {
		log.Printf("Database Error: Failed to insert new quest: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ----- Quest Complete Logic -----
// handleCompleteQuest updates a quest status to 'Completed' and triggers
// the logic for the Weekly Corral archiving.
func handleCompleteQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Get the Quest ID from the form
	questIDStr := r.FormValue("quest_id")
	if questIDStr == "" {
		log.Println("Validation Error: Complete request received without Quest ID.")
		http.Error(w, "Missing Quest ID", http.StatusBadRequest)
		return
	}

	questID, err := strconv.Atoi(questIDStr)
	if err != nil {
		log.Printf("Conversion Error: Invalid Quest ID %s", questIDStr)
		http.Error(w, "Invalid Quest ID", http.StatusBadRequest)
		return
	}

	// 2. Identify the User (Hardcoded to 1 for now)
	userID := 1

	// 3. CALL THE REPOSITORY FUNCTION
	// This is the "Magic Link" that triggers the XP and Corral logic
	err = CompleteQuest(r.Context(), DB, questID, userID)
	if err != nil {
		log.Printf("❌ Repository Error: %v", err)
		http.Error(w, "Could not finalize quest completion", http.StatusInternalServerError)
		return
	}

	// Redirect to clear the POST state and refresh the Pasture
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
