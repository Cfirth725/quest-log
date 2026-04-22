package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("🛰️ Phase 1: Attempting to connect to Database...")
	db, err := Connect()
	if err != nil {
		log.Fatal("❌ Database connection failed: ", err)
	}
	// Assign the local 'db' to the global 'DB' if you set that up in SQLite.go
	DB = db
	log.Println("✅ Phase 1 Complete: Database connected.")

	log.Println("🛰️ Phase 2: Setting up Router...")
	mux := http.NewServeMux()

	// ----- Show the Home Page (The Pasture) -----
	mux.HandleFunc("GET /", ViewPastureHandler(db))

	// ----- Show the "Create" Form (The Forge) -----
	mux.HandleFunc("GET /newquest", func(w http.ResponseWriter, r *http.Request) {
		// 1. Fetch the data
		categories, err := GetCategories(DB)
		if err != nil {
			log.Printf("Error: %v", err)
		}
		// 2. Render the template
		RenderTemplate(w, "new_quest", categories)
	})

	// ----- Process the "Create" Form (The Blacksmith) -----
	mux.HandleFunc("POST /quests/create", handleCreateQuest)

	// ----- Process the "Complete" Form -----
	mux.HandleFunc("POST /quests/complete", handleCompleteQuest)
	log.Println("✅ Phase 2 Complete: Router ready.")

	log.Println("🚀 Phase 3: Launching Server on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
