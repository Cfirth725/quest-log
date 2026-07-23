// ====================================================================
// -- ROUTER MATRIX & PATH MAPPING ENGINE --
// ====================================================================

package web

import (
	"database/sql"
	"net/http"
)

// RegisterRoutes maps all system web views and API endpoints to the HTTP multiplexer.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// Asset Pipeline: Serve static dependencies (CSS, JS, Images).
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 1. Bounty Board & Main Dashboards
	mux.HandleFunc("GET /", ViewBountyBoardHandler)

	// 2. The Arcane Scriptorium (Bulk Ingestion Bridge)
	mux.HandleFunc("GET /scriptorium", RenderScriptoriumHandler)
	mux.HandleFunc("POST /api/v1/quests/import", ImportQuestsAPIHandler)

	// 3. The Chronicle (Weekly Review)
	mux.HandleFunc("GET /chronicle", HandleViewChronicle)
	mux.HandleFunc("POST /chronicle/archive", HandleChronicleQuests)

	// 4. The Forge (Quest Creation & Completion)
	mux.HandleFunc("GET /newquest", HandleNewQuest)
	mux.HandleFunc("POST /quests/create", HandleCreateQuest)
	mux.HandleFunc("POST /quests/complete", HandleCompleteQuest)

	// 5. Administrative & Taxonomy Management
	mux.HandleFunc("GET /settings", HandleSettings)
	mux.HandleFunc("POST /categories/create", HandleCreateCategory)
	mux.HandleFunc("POST /categories/delete", HandleDeleteCategory)
	mux.HandleFunc("POST /settings/archive", ArchiveQuestHandler)
	mux.HandleFunc("POST /settings/downgrade", DowngradeQuestHandler)

	// Suppress favicon.ico from triggering the catch-all Bounty Board route
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
    	w.WriteHeader(http.StatusNoContent)
	})
}
