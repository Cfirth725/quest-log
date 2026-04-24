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
		log.Fatal("Critical Failure: Could not establish database connection: %v", err)
	}

	// Assign to global package-level variable for cross-handler access
	DB = db
	log.Println("Initialization: Database connection established.")

	// Configure application router
	log.Println("Initialization: Configuring HTTP router and routes...")
	mux := http.NewServeMux()

	// --- Dashboard Routes (The Pasture) ---
	mux.HandleFunc("GET /", ViewPastureHandler(db))

	// --- Quest Management Routes (The Forge) ---
	// Route: GET /newquest
	// Renders the quest creation interface with dynamic category data
	mux.HandleFunc("GET /newquest", func(w http.ResponseWriter, r *http.Request) {
		categories, err := GetCategories(DB)
		if err != nil {
			log.Printf("Internal Error: Failed to fetch categories for The Forge: %v", err)
		}

		users, err := GetUsers(DB)
		if err != nil {
			log.Printf("Internal Error: Failed to fetch users: %v", err)
		}

		data := ForgeData{
			Categories: categories,
			Users:      users,
		}

		RenderTemplate(w, "new_quest", data)
	})

	// Route: POST /quests/create
	// Processes incoming quest form data and persists to the database
	mux.HandleFunc("POST /quests/create", handleCreateQuest)

	// Route: POST /quests/complete
	// Handles quest status transitions and completion logic
	mux.HandleFunc("POST /quests/complete", handleCompleteQuest)
	log.Println("Initialization: Router configured successfully.")

	// Launch server
	// Port 8081 is the designated entry point for the Milford Node Quest Log service
	log.Println("Service Status: Launching Quest Log at http://localhost:8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Critical Failure: Server failed to start: %v", err)
	}
}
