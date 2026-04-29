package main

import (
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
)

func main() {
	// --- PHASE 1: Data Layer Initialization ---
	// Establish a persistent connection to the SQLite backend.
	log.Println("Initializing system: Connecting to SQLite database...")
	db, err := Connect()
	if err != nil {
		log.Fatalf("CRITICAL FAILURE: Could not establish database connection: %v", err)
	}

	// Assign the active connection to the package-level global for handler access.
	DB = db
	log.Println("Initialization: Database connection established.")

	// --- PHASE 2: Background Task Orchestration ---
	// Initialize the cron scheduler to manage automated quest lifecycles.
	c := cron.New()

	// Schedule the Master Spawner to run daily at 04:03 AM.
	// This staggered timing avoids resource contention with midnight system backups.
	_, err = c.AddFunc("3 4 * * *", func() {
		RunMasterSpawner(db)
	})

	if err != nil {
		log.Fatal("Initialization: Failed to configure cron scheduler.", err)
	}

	// Start the cron scheduler in a separate background goroutine.
	c.Start()

	// COLD START EXECUTION: Immediately run the spawner on launch to ensure
	// data consistency if the service was offline during the scheduled window
	log.Println("Startup: Performing idempotent quest spawn check...")
	RunMasterSpawner(db)

	// --- PHASE 3: Route Configuration ---
	// Initialize the multiplexer and define service endpoints.
	log.Println("Initialization: Configuring HTTP router and routes...")
	mux := http.NewServeMux()

	// Asset Pipeline: Serve static dependencies (CSS, JS, Images).
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Dashboard & Telemetry: Primary views for active tasks and historical performance.
	mux.HandleFunc("GET /", ViewPastureHandler(db))
	mux.HandleFunc("GET /corral", handleViewCorral)

	// The Forge: Management endpoints for quest creation and state transitions.
	mux.HandleFunc("GET /newquest", handleNewQuest)
	mux.HandleFunc("POST /quests/create", handleCreateQuest)
	mux.HandleFunc("POST /quests/complete", handleCompleteQuest)

	// Administrative: System settings and category hierarchy management.
	mux.HandleFunc("GET /settings", handleSettings)
	mux.HandleFunc("POST /categories/create", handleCreateCategory)
	mux.HandleFunc("POST /categories/delete", handleDeleteCategory)

	log.Println("Initialization: Service routes registered successfully.")

	// --- PHASE 4: Service Execution ---
	// Start the HTTP server on the designated internal port.
	// In production, this service is typically proxied via Nginx or managed by systemd.
	addr := ":8081"
	log.Printf("Service Status: Launching Quest Log at http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("CRITICAL: Server failed to start: %v", err)
	}
}
