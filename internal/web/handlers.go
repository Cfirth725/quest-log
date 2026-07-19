// Package web coordinates HTTP routing multiplexers, input sanitization gates,
// and server-side reward economy verification middleware.
package web

import (
	"database/sql"
	"log"
	"net/http"
	"quest-log/internal/database"
	"quest-log/internal/repository"
	"strconv"
	"strings"
)

// ====================================================================
// -- REWARD ECONOMY CONSTANTS --
// ====================================================================

const (
	XP_EASY   = 1
	XP_MEDIUM = 5
	XP_HARD   = 10
)

// ====================================================================
// -- QUEST FORGE MUTATION HANDLERS --
// ====================================================================

// HandleNewQuest serves the 'Quest Forge' creation interface.
func HandleNewQuest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, err := repository.GetCategories(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch taxonomy categories for The Forge: %v", err)
	}

	users, err := repository.GetUsers(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch user records for The Forge: %v", err)
	}

	data := repository.ForgeData{
		Categories: categories,
		Users:      users,
	}

	log.Printf("[REALTIME] Rendering Quest Forge template wrapper")
	RenderTemplate(w, "new_quest", data)
}

// HandleCreateQuest processes incoming submissions from the Quest Forge form.
func HandleCreateQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		log.Printf("[ERROR] Form parser failed processing quest payload: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	categoryID, _ := strconv.Atoi(r.FormValue("category_id"))
	difficultyStr := r.FormValue("difficulty") // 1. Read raw string token
	ownerID, _ := strconv.Atoi(r.FormValue("owner_id"))
	questType := r.FormValue("quest_type")
	intervalStr := r.FormValue("repeat_interval_days")
	resetDayStr := r.FormValue("reset_day_of_week")

	// 2. Convert difficulty token to an integer safely
	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil {
		log.Printf("[ERROR] Validation rejection: invalid difficulty format payload: %s", difficultyStr)
		http.Error(w, "Invalid difficulty format.", http.StatusBadRequest)
		return
	}

	if ownerID < 0 || ownerID > 2 {
		log.Printf("[SECURE] Access block: unauthorized owner profile assignment out of range: %d", ownerID)
		http.Error(w, "Unauthorized User Assignment.", http.StatusForbidden)
		return
	}

	// --- GHOST GUARD: Input Sanitization ---
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		log.Println("[ERROR] Validation rejection: blocked empty quest title token insertion")
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	// --- HARD-CODED ECONOMY: XP CALCULATION ---
	var calculatedXP int
	switch difficulty {
	case 1:
		calculatedXP = XP_EASY
	case 2:
		calculatedXP = XP_MEDIUM
	case 3:
		calculatedXP = XP_HARD
	default:
		log.Printf("[SECURE] Validation alert: unmapped difficulty tier payload: %d", difficulty)
		http.Error(w, "Invalid difficulty level.", http.StatusBadRequest)
		return
	}

	isNonNegotiable := 0
	if r.FormValue("is_non_negotiable") == "1" {
		isNonNegotiable = 1
	}

	// Parse custom interval parameters
	var interval sql.NullInt64
	if questType == "Repeating" && intervalStr != "" {
		val, err := strconv.Atoi(intervalStr)
		if err == nil && val > 0 {
			interval = sql.NullInt64{Int64: int64(val), Valid: true}
		}
	}

	// Parse day-of-week targets
	resetDayOfWeek := 0
	if questType == "Weekly" && resetDayStr != "" {
		val, err := strconv.Atoi(resetDayStr)
		if err == nil && val >= 0 && val <= 6 {
			resetDayOfWeek = val
		}
	}

	// Defensive DAO Abstraction: Relocate direct DB actions completely
	err = repository.CreateQuest(ctx, database.DB, cleanTitle, categoryID, difficulty, calculatedXP, isNonNegotiable, ownerID, questType, interval, resetDayOfWeek)
	if err != nil {
		log.Printf("[ERROR] Ingestion breakdown inserting quest to ledger: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Successfully minted quest contract '%s' into active ledger pool", cleanTitle)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleCompleteQuest handles marking a quest as resolved.
func HandleCompleteQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	questID, err := strconv.Atoi(r.FormValue("quest_id"))
	if err != nil {
		http.Error(w, "Invalid Quest Pointer Reference", http.StatusBadRequest)
		return
	}

	// Complete via transaction wrapper inside the DAO layer
	if err := repository.CompleteQuest(ctx, database.DB, questID, 1); err != nil {
		log.Printf("[ERROR] Relational database breakdown finalizing transaction state: %v", err)
		http.Error(w, "Could not finalize quest completion status", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] State transition committed: quest ID %d resolved and rewards disbursed", questID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ====================================================================
// -- ADMINISTRATIVE & TAXONOMY MAINTENANCE HANDLERS --
// ====================================================================

// HandleSettings renders the admin dashboard view.
func HandleSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, err := repository.GetCategories(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Settings layer failed loading active category metadata: %v", err)
	}

	quests, err := repository.GetActiveQuests(ctx, database.DB, 1, false)
	if err != nil {
		log.Printf("[ERROR] Settings layer failed loading quest configuration registry: %v", err)
	}

	data := repository.SettingsPageData{
		Categories: categories,
		Quests:     quests,
	}

	log.Printf("[REALTIME] Rendering Administrative Settings Panel")
	RenderTemplate(w, "settings", data)
}

// HandleCreateCategory processes requests to establish new taxonomies.
func HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := strings.TrimSpace(r.FormValue("name"))
	color := r.FormValue("color")

	if name == "" {
		http.Error(w, "Category name cannot be blank.", http.StatusBadRequest)
		return
	}

	if err := repository.InsertCategory(ctx, database.DB, name, color); err != nil {
		log.Printf("[ERROR] Data persistence failure minting taxonomy label: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Successfully established new taxonomy category: %s", name)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleDeleteCategory drops a category if no dependency checks fail.
func HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.FormValue("category_id")

	count, err := repository.GetQuestCountByCategory(ctx, database.DB, id)
	if err != nil {
		log.Printf("[ERROR] Relational integrity check failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		log.Printf("[ERROR] Deletion blocked: category ID %s contains %d active references", id, count)
		http.Error(w, "Conflict: Cannot delete category with active dependencies.", http.StatusConflict)
		return
	}

	if err := repository.DeleteCategoryByID(ctx, database.DB, id); err != nil {
		log.Printf("[ERROR] Data persistence failure removing category: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Successfully dropped taxonomy node record ID: %s", id)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleChronicleQuests processes bulk archival transitions for finished tasks.
func HandleChronicleQuests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	count, err := repository.ChronicleCompletedQuests(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Scribe engine failed compiling Chronicle tasks: %v", err)
		http.Error(w, "Failed to compile the chronicle ledger", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Chronicles updated: %d finished quests committed to historical ledger archives.", count)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ArchiveQuestHandler executes soft-delete sequences.
func ArchiveQuestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	id, _ := strconv.Atoi(r.FormValue("id"))

	if err := repository.SoftDeleteQuest(ctx, database.DB, id); err != nil {
		log.Printf("[ERROR] Failed to execute soft-delete routine on quest ID %d: %v", id, err)
		http.Error(w, "Failed to archive quest", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Quest ID %d successfully moved to non-destructive cold state storage", id)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// DowngradeQuestHandler reduces a repeating task parameter down to single lifecycle execution rules.
func DowngradeQuestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	id, _ := strconv.Atoi(r.FormValue("id"))

	if err := repository.DowngradeToOneTime(ctx, database.DB, id); err != nil {
		log.Printf("[ERROR] Failed to execute down-tier modification on quest ID %d: %v", id, err)
		http.Error(w, "Failed to downgrade quest specifications", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Target cadence configuration reduced down to one-time parameters for ID %d", id)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
