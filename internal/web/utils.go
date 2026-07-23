// Package web coordinates HTTP routing multiplexers, input sanitization gates,
// and server-side reward economy verification middleware.
package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// RenderTemplate orchestrates the compilation and execution of modular Go layouts.
func RenderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	basePath := "templates/layouts/base.html"
	pagePath := fmt.Sprintf("templates/%s.html", tmplName)

	// Create root template explicitly bound to "base"
	tmpl := template.New("base")

	// Parse base layout and requested page
	tmpl, err := tmpl.ParseFiles(basePath, pagePath)
	if err != nil {
		log.Printf("[ERROR] Failed to parse template layout assets for '%s': %v", tmplName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Safely attach partials to the template set
	_, err = tmpl.ParseGlob("templates/partials/*.html")
	if err != nil {
		log.Printf("[INIT] Partial mapping warning or bypass for '%s': %v", tmplName, err)
	}

	// Explicitly set UTF-8 HTML content type for browsers/mobile
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute layout
	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("[ERROR] Layout execution failure while rendering page '%s': %v", tmplName, err)
	}
}
