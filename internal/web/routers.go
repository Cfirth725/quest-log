// Package web coordinates HTTP routing multiplexers, input sanitization gates,
// and server-side reward economy verification middleware.
package web

import (
	"log"
	"net/http"
	"quest-log/internal/database"
	"quest-log/internal/repository"
)

// ====================================================================
// -- THE BOUNTY BOARD (MAIN DASHBOARD) ENGINE --
// ====================================================================

// ViewBountyBoardHandler coordinates the retrieval of active tasks and manages the
// dashboard's display state. It supports a 'Momentum Mode' filter to assist
// with cognitive load management during high-pressure barometric events.
func ViewBountyBoardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	momentumMode := r.URL.Query().Get("momentum") == "true"

	// Defensive DAO Abstraction: Fully parameterized context lookup
	// Defaulting to User ID 1 for the system owner profile.
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
// -- THE CHRONICLE (WEEKLY REVIEW WINDOW) ENGINE --
// ====================================================================

// HandleViewChronicle renders the historical reporting dashboard and weekly summaries.
func HandleViewChronicle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Gather the core itemized rolling 7-day victory log
	summary, err := repository.GetWeeklySummary(ctx, database.DB, 1)
	if err != nil {
		log.Printf("[ERROR] Scribe engine summary parser failure: %v", err)
		http.Error(w, "Failed to load historical archives from The Chronicle", http.StatusInternalServerError)
		return
	}

	// Generate the aggregate analytical metrics
	report, err := repository.GenerateWeeklyChronicleReport(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Chronicle metrics evaluation execution block: %v", err)
	} else {
		summary.Report = report
	}

	log.Printf("[REALTIME] Fetching historic ledger archives for weekly review window")
	RenderTemplate(w, "chronicle", summary)
}
