// ====================================================================
// -- WEB DOMAIN: HTTP HANDLERS & CONTROLLER ENGINE --
// ====================================================================

// Package web coordinates HTTP routing multiplexers, input sanitization gates,
// and server-side reward economy verification middleware.
package web

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"quest-log/internal/database"
	"quest-log/internal/ingest"
	"quest-log/internal/repository"
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
// -- 1. BOUNTY BOARD & MAIN DASHBOARD HANDLERS --
// ====================================================================

// ViewBountyBoardHandler coordinates the retrieval of active tasks and manages the
// dashboard's display state. It supports a 'Momentum Mode' filter.
func ViewBountyBoardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	momentumMode := r.URL.Query().Get("momentum") == "true"

	// Defensive DAO Abstraction: Parameterized context lookup for User ID 1
	activeQuests, err := repository.GetActiveQuests(ctx, database.DB, 1, momentumMode)
	if err != nil {
		log.Printf("[ERROR] Database transaction failure loading active workload layout: %v", err)
		http.Error(w, "Failed to load quests from the vault", http.StatusInternalServerError)
		return
	}

	data := struct {
		Quests       []repository.QuestResponse
		MomentumMode bool
	}{
		Quests:       activeQuests,
		MomentumMode: momentumMode,
	}

	log.Printf("[REALTIME] Compiling active contracts matrix for Bounty Board display")
	RenderTemplate(w, "bounty_board", data)
}

// ====================================================================
// -- 2. QUEST FORGE MUTATION HANDLERS --
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
	difficultyStr := r.FormValue("difficulty")
	ownerID, _ := strconv.Atoi(r.FormValue("owner_id"))
	questType := r.FormValue("quest_type")
	intervalStr := r.FormValue("repeat_interval_days")
	resetDayStr := r.FormValue("reset_day_of_week")

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

	// Default to User ID 1 until multi-user UI interface is implemented
	userID := 1

	if err := repository.CompleteQuest(ctx, database.DB, questID, userID); err != nil {
		log.Printf("[ERROR] Relational database breakdown finalizing transaction state: %v", err)
		http.Error(w, "Could not finalize quest completion status", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] State transition committed: quest ID %d resolved and rewards disbursed", questID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ====================================================================
// -- 3. THE CHRONICLE (HISTORICAL REVIEW) HANDLERS --
// ====================================================================

// HandleViewChronicle renders the historical reporting dashboard and weekly summaries.
func HandleViewChronicle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	summary, err := repository.GetWeeklySummary(ctx, database.DB, 1)
	if err != nil {
		log.Printf("[ERROR] Scribe engine summary parser failure: %v", err)
		http.Error(w, "Failed to load historical archives from The Chronicle", http.StatusInternalServerError)
		return
	}

	report, err := repository.GenerateWeeklyChronicleReport(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Chronicle metrics evaluation execution block: %v", err)
	} else {
		summary.Report = report
	}

	log.Printf("[REALTIME] Fetching historic ledger archives for weekly review window")
	RenderTemplate(w, "chronicle", summary)
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

// ====================================================================
// -- 4. THE ARCANE SCRIPTORIUM (BULK INGESTION) HANDLERS --
// ====================================================================

// RenderScriptoriumHandler serves the bulk ingestion page interface.
func RenderScriptoriumHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[REALTIME] Serving Arcane Scriptorium interface")

	data := struct {
		Title string
	}{
		Title: "The Arcane Scriptorium",
	}

	RenderTemplate(w, "scriptorium", data)
}

// ImportQuestsAPIHandler receives raw JSON manifest payloads, executes sanitization
// and schema validation, and commits records down to the database inside a batch transaction.
func ImportQuestsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[REALTIME] Inbound JSON manifest received")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Payload read fault on /api/v1/quests/import: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body payload"})
		return
	}
	defer r.Body.Close()

	// 1. Pipeline extraction & schema validation pass
	extractedQuests, err := ingest.ParseJSONPayload(bodyBytes)
	if err != nil {
		log.Printf("[ERROR] Validation/Parsing fault on import: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 2. Execute transactional batch import
	userID := 1 // Default primary user scope
	result, err := ingest.ExecuteBatchIngestion(r.Context(), database.DB, userID, extractedQuests)
	if err != nil {
		log.Printf("[ERROR] Batch transaction execution failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 3. Return telemetry payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "success",
		"quests_minted":      result.QuestsMinted,
		"categories_created": result.CategoriesCreated,
		"total_processed":    result.TotalProcessed,
	})
}

// AnalyzeImportAPIHandler inspects the JSON payload against existing database categories
// without committing records, flagging any unmapped categories for UI resolution.
func AnalyzeImportAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read body"})
		return
	}
	defer r.Body.Close()

	// Parse payload schema
	extractedQuests, err := ingest.ParseJSONPayload(bodyBytes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Fetch current database categories
	existingCategories, err := repository.GetCategories(r.Context(), database.DB)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to load database categories"})
		return
	}

	// Build lookup set (case-insensitive)
	existingMap := make(map[string]bool)
	for _, c := range existingCategories {
		existingMap[strings.ToLower(c.Name)] = true
	}

	// Detect missing categories in payload
	unmatchedSet := make(map[string]bool)
	var unmatchedList []string

	for _, q := range extractedQuests {
		catName := strings.TrimSpace(q.CategoryName)
		if catName == "" {
			catName = "Uncategorized"
		}

		lookupKey := strings.ToLower(catName)
		if !existingMap[lookupKey] && !unmatchedSet[lookupKey] {
			unmatchedSet[lookupKey] = true
			unmatchedList = append(unmatchedList, catName)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":               "success",
		"total_quests":         len(extractedQuests),
		"unmatched_categories": unmatchedList,
		"existing_categories":  existingCategories,
	})
}

// ====================================================================
// -- 5. ADMINISTRATIVE & TAXONOMY HANDLERS --
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
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

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
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

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
