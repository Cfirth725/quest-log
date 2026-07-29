// Package web coordinates HTTP request lifecycle routing, form decoding,
// template rendering, and user session authentication handlers.
package web

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"quest-log/internal/database"
	"quest-log/internal/middleware"
	"quest-log/internal/repository"
)

// ====================================================================
// -- SESSION & IDENTITY AUTHENTICATION HANDLERS --
// ====================================================================

// RenderLoginPageHandler renders the profile selection and PIN entry view.
func RenderLoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	users, err := repository.GetAllUsers(ctx, database.DB)
	if err != nil {
		log.Printf("[ERROR] Login gateway failure fetching users: %v", err)
		http.Error(w, "Failed to load system user accounts", http.StatusInternalServerError)
		return
	}

	data := struct {
		Users []repository.User
		Error string
	}{
		Users: users,
		Error: r.URL.Query().Get("error"),
	}

	RenderTemplate(w, "login", data)
}

// HandleLoginSubmitHandler validates the selected user profile and PIN credential,
// minting a 30-day session cookie on successful verification.
func HandleLoginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	userIDStr := r.FormValue("user_id")
	pinInput := r.FormValue("pin")

	if userIDStr == "" {
		http.Redirect(w, r, "/login?error=Please+select+a+user+profile", http.StatusSeeOther)
		return
	}

	var userID int
	if _, err := stringToInt(userIDStr, &userID); err != nil {
		http.Redirect(w, r, "/login?error=Invalid+user+identifier", http.StatusSeeOther)
		return
	}

	user, err := repository.GetUserByID(ctx, database.DB, userID)
	if err != nil {
		http.Redirect(w, r, "/login?error=User+profile+not+found", http.StatusSeeOther)
		return
	}

	// Verify PIN if set
	if user.PinHash != "" {
		hasher := sha256.New()
		hasher.Write([]byte(pinInput))
		enteredHash := hex.EncodeToString(hasher.Sum(nil))

		if enteredHash != user.PinHash {
			log.Printf("[SECURE] Failed PIN authentication attempt for user ID %d (%s)", user.ID, user.Name)
			http.Redirect(w, r, "/login?error=Incorrect+PIN+entered", http.StatusSeeOther)
			return
		}
	}

	// Create 30-day session
	session, err := repository.CreateSession(ctx, database.DB, user.ID, 30*24*time.Hour)
	if err != nil {
		log.Printf("[ERROR] Session creation transaction failure for user %d: %v", user.ID, err)
		http.Error(w, "Failed to establish authenticated session", http.StatusInternalServerError)
		return
	}

	// Set secure HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[OK] Authenticated session established for User %d (%s) -> Granted access", user.ID, user.Name)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleLogoutHandler revokes the active session token and clears the browser cookie.
func HandleLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = repository.DeleteSession(r.Context(), database.DB, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	log.Printf("[SECURE] User session revoked cleanly -> Redirecting to login gateway")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Helper utility for simple integer conversion
func stringToInt(s string, target *int) (int, error) {
	var val int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, http.ErrBodyNotAllowed
		}
		val = val*10 + int(ch-'0')
	}
	*target = val
	return val, nil
}
