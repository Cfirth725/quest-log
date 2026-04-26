package main

import (
	"log"
	"net/http"
)

func main() {
	// Initialize database connection
	log.Println("Initializing system: Connecting to SQLite database...")
	db, err := Connect()
	if err != nil {
		log.Fatalf("Critical Failure: Could not establish database connection: %v", err)
	}

	// Assign to global package-level variable for cross-handler access
	DB = db
	log.Println("Initialization: Database connection established.")

	// Configure application router
	log.Println("Initialization: Configuring HTTP router and routes...")
	mux := http.NewServeMux()

	// Serve static files (CSS, Images)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// --- Dashboard Routes (The Pasture) ---
	mux.HandleFunc("GET /", ViewPastureHandler(db))

	// --- Quest Management Routes (The Forge) ---
	mux.HandleFunc("GET /newquest", handleNewQuest)
	mux.HandleFunc("POST /quests/create", handleCreateQuest)
	mux.HandleFunc("POST /quests/complete", handleCompleteQuest)
	log.Println("Initialization: Router configured successfully.")

	// --- Settings & Category Management ---
	mux.HandleFunc("GET /settings", handleSettings)
	mux.HandleFunc("POST /categories/create", handleCreateCategory)
	mux.HandleFunc("POST /categories/delete", handleDeleteCategory)
	log.Println("Initialization: Settings configured successfully.")

	// Launch server
	// Port 8081 is the designated entry point for the Milford Node Quest Log service
	log.Println("Service Status: Launching Quest Log at http://localhost:8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Critical Failure: Server failed to start: %v", err)
	}
}
