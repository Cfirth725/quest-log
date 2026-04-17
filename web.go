package main

import (
    "database/sql"
    "net/http"
    "html/template"
    "log"
    "fmt"
)

// RenderTemplate is your new 'Standard' for showing pages
func RenderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
    // 1. Point to the EXACT subfolder for the base layout
    basePath := "templates/layouts/base.html" 
    
    // 2. Point to the root templates folder for the page (pasture or new_quest)
    pagePath := fmt.Sprintf("templates/%s.html", tmplName)

    // 3. Parse those specific files
    tmpl, err := template.ParseFiles(basePath, pagePath)
    if err != nil {
        log.Printf("❌ Template Load Error: %v", err)
        http.Error(w, "Check your folder paths in RenderTemplate", 500)
        return
    }

    // 4. Also grab the partials (the quest cards)
    _, err = tmpl.ParseGlob("templates/partials/*.html")
    if err != nil {
        log.Printf("⚠️ Partial Warning: %v", err)
        // We don't necessarily want to kill the whole page if a partial is missing
    }

    // 5. Execute the 'base' layout
    err = tmpl.ExecuteTemplate(w, "base", data)
    if err != nil {
        log.Printf("❌ Execution Error: %v", err)
    }
}


// ViewPastureHandler handles traffic to the homepage
func ViewPastureHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        activeQuests, err := GetActiveQuests(r.Context(), db, 1)
        if err != nil {
            http.Error(w, "Failed to load quests", http.StatusInternalServerError)
            return
        }
        
        RenderTemplate(w, "pasture", activeQuests) 
    }
}


// ----- New Quest Logic -----
func handleCreateQuest(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/newquest", http.StatusSeeOther)
        return
    }

    // 1. Parse the Form Data
    err := r.ParseForm()
    if err != nil {
        log.Printf("Error parsing form: %v", err)
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // 2. Extract values (matching the 'name' attributes in your HTML)
    title := r.FormValue("title")
    categoryID := r.FormValue("category_id")
    difficulty := r.FormValue("difficulty")
    baseXP := r.FormValue("base_xp")

    // 3. Insert into SQLite
    // For now, we "hard-code" the ID to 1 (Carolyn)
    const CurrentUserID = 1 

    query := `INSERT INTO quests (title, category_id, difficulty, base_xp, status, owner_id, quest_type)
        VALUES (?, ?, ?, ?, 'active', ?, 'One-Time')`

    _, err = DB.Exec(query, title, categoryID, difficulty, baseXP, CurrentUserID)
    if err != nil {
        log.Printf("Database Error: %v", err)
        http.Error(w, "Could not save quest", http.StatusInternalServerError)
        return
    }

    // 4. Success! Send them back to the pasture to see the new animal
    http.Redirect(w, r, "/", http.StatusSeeOther)
}


// ----- Quest Complete Logic -----
func handleCompleteQuest(w http.ResponseWriter, r *http.Request) {
    // 1. Ensure it's a POST request
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    // 2. Grab the quest_id from the HIDDEN input field
    questID := r.FormValue("quest_id")
    if questID == "" {
        log.Println("❌ Error: No Quest ID received in form")
        http.Error(w, "Missing Quest ID", http.StatusBadRequest)
        return
    }

    // 3. Update the Database
    query := `UPDATE quests SET status = 'Completed' WHERE id = ?`
    _, err := DB.Exec(query, questID)
    if err != nil {
        log.Printf("❌ Database Error: %v", err)
        http.Error(w, "Could not update quest", http.StatusInternalServerError)
        return
    }

    // 4. Send them back to the pasture to see the card vanish/update
    http.Redirect(w, r, "/", http.StatusSeeOther)
}