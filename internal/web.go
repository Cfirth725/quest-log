package internal

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Experience constants define the reward economy for various quest tiers.
// These serve as the ground truth for XP distribution across the system.
const (
	XP_DUCK  = 1
	XP_SHEEP = 5
	XP_COW   = 10
)

// RenderTemplate orchestrates the assembly and execution of HTML templates.
// It merges the base layout with specific view files and injects shared partials
// for modular UI components like quest cards.
func RenderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	basePath := "templates/layouts/base.html"
	pagePath := fmt.Sprintf("templates/%s.html", tmplName)

	// Initialize the template engine with core layout files.
	tmpl, err := template.ParseFiles(basePath, pagePath)
	if err != nil {
		log.Printf("Internal Error: Failed to parse template files: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Load shared UI partials to support reusable components across different views.
	_, err = tmpl.ParseGlob("templates/partials/*.html")
	if err != nil {
		log.Printf("Partial Load Warning: %v", err)
	}

	// Execute the "base" template, which serves as the primary HTML wrapper.
	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("Execution Error: Failed to render template: %v", err)
	}
}

// ViewPastureHandler coordinates the retrieval of active tasks and manages the
// dashboard's display state. It supports a 'Momentum Mode' filter to assist
// with cognitive load management during high-pressure barometric events.
func ViewPastureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the Momentum Mode toggle from the URL query parameters.
		momentumMode := r.URL.Query().Get("momentum") == "true"

		// Retrieve tasks from the repository. Defaulting to UserID 1 for the system owner.
		activeQuests, err := GetActiveQuests(r.Context(), db, 1, momentumMode)
		if err != nil {
			log.Printf("Database Error: Failed to retrieve active quests: %v", err)
			http.Error(w, "Failed to load quests", http.StatusInternalServerError)
			return
		}

		// Encapsulate the result set and the current filter state into an anonymous
		// struct to provide a unified context for the HTML template engine.
		data := struct {
			Quests       []QuestResponse
			MomentumMode bool
		}{
			Quests:       activeQuests,
			MomentumMode: momentumMode,
		}

		// Render the primary dashboard view using the combined data context.
		RenderTemplate(w, "pasture", data)
	}
}

// ----- New Quest Logic -----
// HandleCreateQuest processes 'Quest Forge' submissions. It enforces a strict
// "Ghost Guard" policy for input validation and ensures that economy-impacting
// values (like XP) are calculated server-side rather than accepted from the client.
func HandleCreateQuest(w http.ResponseWriter, r *http.Request) {
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

	// Extract client-side inputs for transformation.
	title := r.FormValue("title")
	categoryID := r.FormValue("category_id")
	difficulty := r.FormValue("difficulty")
	ownerIDStr := r.FormValue("owner_id")
	questType := r.FormValue("quest_type")
	intervalStr := r.FormValue("repeat_interval_days")
	resetDayStr := r.FormValue("reset_day_of_week")

	// Validate Ownership: Ensures assignments remain within authorized system ranges.
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
	// Data Sanitization: Prevent empty titles or malformed strings from persisting.
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		log.Println("Validation Warning: Blocked empty quest title submission.")
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	// --- HARD-CODED ECONOMY: XP Calculation ---
	// Economy Enforcement: Determine XP based on validated difficulty tiers.
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
	// Triage Logic: Determine if the quest qualifies for Priority Shield status.
	isNonNegotiable := 0
	if r.FormValue("is_non_negotiable") == "1" {
		isNonNegotiable = 1
	}

	// Lifecycle Validation: Ensure the quest type adheres to the database schema.
	validTypes := map[string]bool{"One-Time": true, "Daily": true, "Repeating": true, "Weekly": true}
	if !validTypes[questType] {
		log.Printf("Security Alert: Invalid quest_type: %s", questType)
		http.Error(w, "Invalid quest type.", http.StatusBadRequest)
		return
	}

	// Temporal Configuration: Handle interval metadata for recurring lifecycles.
	var interval sql.NullInt64
	if questType == "Repeating" && intervalStr != "" {
		val, err := strconv.Atoi(intervalStr)
		if err == nil && val > 0 {
			interval = sql.NullInt64{Int64: int64(val), Valid: true}
		} else {
			interval = sql.NullInt64{Int64: 1, Valid: true}
		}
	}

	// Weekly Configuration: Parse target reset day (Defaulting to 0 = Sunday if missing/malformed)
	resetDayOfWeek := 0
	if questType == "Weekly" && resetDayStr != "" {
		val, err := strconv.Atoi(resetDayStr)
		if err == nil && val >= 0 && val <= 6 {
			resetDayOfWeek = val
		}
	}

	// Persistent Injection: Commit the quest to the system of record.
	query := `INSERT INTO quests (title, category_id, difficulty, base_xp, is_non_negotiable, status, owner_id, quest_type, repeat_interval_days, reset_day_of_week)
        VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?)`

	_, err = DB.Exec(query, cleanTitle, categoryID, difficulty, calculatedXP, isNonNegotiable, ownerID, questType, interval, resetDayOfWeek)
	if err != nil {
		log.Printf("Database Error: Failed to insert new quest: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ----- Quest Complete Logic -----
// HandleCompleteQuest facilitates the transition of a task from 'active' to 'Completed'.
// It acts as the bridge between the HTTP request and the transactional repository logic.
func HandleCompleteQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Identify the User (Hardcoded to 1 for now)
	userID := 1

	// Call the transaction-wrapped repository logic to handle XP and audit logging.
	err = CompleteQuest(r.Context(), DB, questID, userID)
	if err != nil {
		log.Printf("❌ Repository Error: %v", err)
		http.Error(w, "Could not finalize quest completion", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
