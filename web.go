package main

import (
    "database/sql"
    "net/http"
    "html/template"
    "log"
)

// ViewPastureHandler handles traffic to the homepage
func ViewPastureHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Get the data from your quests package
        activeQuests, err := GetActiveQuests(r.Context(), db, 1) // Assuming User ID 1 for now
        if err != nil {
            http.Error(w, "Failed to load quests", http.StatusInternalServerError)
            return
        }

        // 2. Parse the HTML templates
        tmpl, err := template.ParseGlob("templates/**/*.html")
        if err != nil {
            log.Fatal("Could not find templates: ", err)
        }
        // Also parse the top-level templates folder
        tmpl, err = tmpl.ParseGlob("templates/*.html")

        // 3. Inject the data into the HTML
        tmpl.ExecuteTemplate(w, "base", activeQuests)
    }
}