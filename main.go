package main

import (
	"context"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// STORAGE SWEEP: Fire database compaction asynchronously on boot.
	// This cleans up unallocated space before heavy daily operations begin.
	go OptimizeDatabase(db)

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
	mux.HandleFunc("POST /settings/archive", ArchiveQuestHandler)
	mux.HandleFunc("POST /settings/downgrade/", DowngradeQuestHandler)

	log.Println("Initialization: Service routes registered successfully.")

	// --- PHASE 4: Service Execution (Non-Blocking) ---
	// Start the HTTP server on the designated internal port.
	// In production, this service is typically proxied via Nginx or managed by systemd.
	addr := ":8081"
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Launch the server in a background goroutine
	go func() {
		log.Printf("Service Status: Launching Quest Log at http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("CRITICAL: Server failed: %v", err)
		}
	}()

	// --- PHASE 5: Graceful Shutdown Orchestration ---
	// Create a channel to listen for interrupt signals (Ctrl+C or Docker stop)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait here until we receive a signal
	<-stop
	log.Println("\nShutdown Initiated: Closing Quest Log...")

	// 1. Stop the cron scheduler
	c.Stop()

	// 2. Give the HTTP server 5 seconds to finish active requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Warning: Server forced to shutdown: %v", err)
	}

	// 3. THE CRITICAL STEP: Close the database to merge the WAL file
	if db != nil {
		log.Println("Database: Merging WAL and closing connection...")
		if err := db.Close(); err != nil {
			log.Printf("Error: Database closure failed: %v", err)
		} else {
			log.Println("Database: Successfully closed and persisted.")
		}
	}

	log.Println("System Offline: All resources released.")
}
