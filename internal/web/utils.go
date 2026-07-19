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

	tmpl, err := template.ParseFiles(basePath, pagePath)
	if err != nil {
		log.Printf("[ERROR] Failed to parse template layout assets: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = tmpl.ParseGlob("templates/partials/*.html")
	if err != nil {
		log.Printf("[INIT] Partial mapping warning or bypass: %v", err)
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("[ERROR] Layout execution failure while rendering page: %v", err)
	}
}
