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
	"time"

	"quest-log/internal/database"
	"quest-log/internal/ingest"
	"quest-log/internal/middleware"
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
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	momentumMode := r.URL.Query().Get("momentum") == "true"

	// Fetch personal + household quests for the authenticated user
	activeQuests, err := repository.GetActiveQuests(ctx, database.DB, user.ID, momentumMode)
	if err != nil {
		log.Printf("[ERROR] Database transaction failure loading active workload layout for user %d: %v", user.ID, err)
		http.Error(w, "Failed to load quests from the vault", http.StatusInternalServerError)
		return
	}

	data := struct {
		User         *repository.User
		Quests       []repository.QuestResponse
		MomentumMode bool
	}{
		User:         user,
		Quests:       activeQuests,
		MomentumMode: momentumMode,
	}

	log.Printf("[REALTIME] Compiling active contracts matrix for User %d (%s) on Bounty Board", user.ID, user.Name)
	RenderTemplate(w, "bounty_board", data)
}

// ====================================================================
// -- 2. QUEST FORGE MUTATION HANDLERS --
// ====================================================================

// HandleNewQuest serves the 'Quest Forge' creation interface.
func HandleNewQuest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	categories, err := repository.GetCategories(ctx, database.DB, user.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch taxonomy categories for The Forge: %v", err)
	}

	users, err := repository.GetUsers(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch user records for The Forge: %v", err)
	}

	data := repository.ForgeData{
		User:       user,
		Categories: categories,
		Users:      users,
	}

	log.Printf("[REALTIME] Rendering Quest Forge template wrapper for User %d (%s)", user.ID, user.Name)
	RenderTemplate(w, "new_quest", data)
}

// HandleCreateQuest processes incoming submissions from the Quest Forge form.
func HandleCreateQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/newquest", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

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

	// Default unassigned owner selections to the active session user
	if ownerID == 0 && r.FormValue("owner_id") == "" {
		ownerID = user.ID
	}

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

	log.Printf("[OK] Successfully minted quest contract '%s' into active ledger pool for Owner %d", cleanTitle, ownerID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleCompleteQuest handles marking a quest as resolved.
func HandleCompleteQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	questID, err := strconv.Atoi(r.FormValue("quest_id"))
	if err != nil {
		http.Error(w, "Invalid Quest Pointer Reference", http.StatusBadRequest)
		return
	}

	// Attribute quest resolution and streak gains directly to the session user
	if err := repository.CompleteQuest(ctx, database.DB, questID, user.ID); err != nil {
		log.Printf("[ERROR] Relational database breakdown finalizing transaction state for user %d: %v", user.ID, err)
		http.Error(w, "Could not finalize quest completion status", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] State transition committed: quest ID %d resolved and rewards disbursed to User %d (%s)", questID, user.ID, user.Name)
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
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	summary, err := repository.GetWeeklySummary(ctx, database.DB, user.ID)
	if err != nil {
		log.Printf("[ERROR] Scribe engine summary parser failure for user %d: %v", user.ID, err)
		http.Error(w, "Failed to load historical archives from The Chronicle", http.StatusInternalServerError)
		return
	}

	summary.User = user

	report, err := repository.GenerateWeeklyChronicleReport(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Chronicle metrics evaluation execution block: %v", err)
	} else {
		summary.Report = report
	}

	log.Printf("[REALTIME] Fetching historic ledger archives for weekly review window (User %d)", user.ID)
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

	user, _ := middleware.GetUserFromContext(r.Context())

	data := struct {
		User  *repository.User
		Title string
	}{
		User:  user,
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

	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	userID := 1
	if ok {
		userID = user.ID
	}

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

	// 2. Execute transactional batch import bound to session user
	result, err := ingest.ExecuteBatchIngestion(ctx, database.DB, userID, extractedQuests)
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

	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	userID := 1
	if ok {
		userID = user.ID
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

	// Fetch current database categories for the session user + household
	existingCategories, err := repository.GetCategories(ctx, database.DB, userID)
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
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	categories, err := repository.GetCategories(ctx, database.DB, user.ID)
	if err != nil {
		log.Printf("[ERROR] Settings layer failed loading active category metadata for user %d: %v", user.ID, err)
	}

	quests, err := repository.GetActiveQuests(ctx, database.DB, user.ID, false)
	if err != nil {
		log.Printf("[ERROR] Settings layer failed loading quest configuration registry for user %d: %v", user.ID, err)
	}

	data := repository.SettingsPageData{
		User:       user,
		Categories: categories,
		Quests:     quests,
	}

	log.Printf("[REALTIME] Rendering Administrative Settings Panel for User %d (%s)", user.ID, user.Name)
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

// ====================================================================
// -- 6. HEADLESS TELEMETRY & OBSERVABILITY HANDLERS --
// ====================================================================

// TelemetryAPIHandler executes real-time system metrics evaluation against the active
// workload database, returning structured JSON for headless observability consumption.
func TelemetryAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[REALTIME] Compiling headless system telemetry and workload metrics matrix")

	ctx := r.Context()
	todayStr := time.Now().Format("2006-01-02")

	telemetry := repository.TelemetryPayload{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		CategoryBreakdown: []repository.CategoryDistribution{},
	}

	// 1. Evaluate active workload contracts & priority non-negotiables
	err := database.DB.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN is_non_negotiable = 1 AND status != 'completed' THEN 1 ELSE 0 END), 0)
		FROM quests WHERE status != 'completed' AND deleted_at IS NULL
	`).Scan(&telemetry.TotalActiveQuests, &telemetry.NonNegotiablesOpen)
	if err != nil {
		log.Printf("[ERROR] Telemetry engine metrics evaluation failure on active pool: %v", err)
	}

	// 2. Evaluate daily resolution velocity & XP disbursements from the immutable ledger
	err = database.DB.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(SUM(xp_awarded), 0)
		FROM quest_completions 
		WHERE DATE(completed_at) = ?
	`, todayStr).Scan(&telemetry.CompletedToday, &telemetry.XPEarnedToday)
	if err != nil {
		log.Printf("[ERROR] Telemetry engine metrics evaluation failure on daily ledger: %v", err)
	}

	// 3. Compile active taxonomy category distribution matrix
	rows, err := database.DB.QueryContext(ctx, `
		SELECT c.name, COALESCE(c.color_hex, '#808080'), COUNT(q.id) 
		FROM quests q
		JOIN categories c ON q.category_id = c.id
		WHERE q.status != 'completed' AND q.deleted_at IS NULL
		GROUP BY c.id
	`)
	if err != nil {
		log.Printf("[ERROR] Telemetry engine failure querying taxonomy distribution: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var cat repository.CategoryDistribution
			if err := rows.Scan(&cat.Name, &cat.Color, &cat.Count); err == nil {
				telemetry.CategoryBreakdown = append(telemetry.CategoryBreakdown, cat)
			}
		}
	}

	log.Printf("[OK] Telemetry payload assembled successfully: %d active quests, %d XP disbursed today",
		telemetry.TotalActiveQuests, telemetry.XPEarnedToday)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(telemetry)
}
