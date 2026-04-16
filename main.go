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
    log.Println("✅ Phase 1 Complete: Database connected.")

    log.Println("🛰️ Phase 2: Setting up Router...")
    mux := http.NewServeMux()
    mux.HandleFunc("GET /", ViewPastureHandler(db))
    log.Println("✅ Phase 2 Complete: Router ready.")

    log.Println("🚀 Phase 3: Launching Server on http://localhost:8081")
    log.Fatal(http.ListenAndServe(":8081", mux))
}