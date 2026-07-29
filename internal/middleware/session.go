// Package middleware provides HTTP interceptors for request logging,
// zero-trust API security, and session-based user authentication.
package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"quest-log/internal/repository"
)

// ====================================================================
// -- CONTEXT BINDING KEYS & SECURITY CONSTANTS --
// ====================================================================

type contextKey string

// UserContextKey serves as the immutable key for retrieving the active
// authenticated user profile from the request context.
const UserContextKey contextKey = "authenticatedUser"

// SessionCookieName defines the standard cookie identifier across all web clients.
const SessionCookieName = "ql_session"

// ====================================================================
// -- SESSION AUTHENTICATION INTERCEPTOR --
// ====================================================================

// SessionAuth creates a middleware interceptor that validates the ql_session cookie.
// On valid sessions, it injects the resolved User object into r.Context().
// Unauthenticated requests to protected web routes are redirected to the login gateway.
func SessionAuth(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// 1. Bypass authentication checks for static assets and public endpoints
			if strings.HasPrefix(path, "/static/") || path == "/favicon.ico" || path == "/login" || path == "/api/v1/telemetry" {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Extract session cookie from request
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				log.Printf("[SECURE] Unauthenticated access attempt to %s -> Redirecting to /login", path)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := r.Context()

			// 3. Resolve active session from database
			session, err := repository.GetSessionByToken(ctx, db, cookie.Value)
			if err != nil {
				log.Printf("[SECURE] Invalid or expired session token presented -> Redirecting to /login")
				// Clear invalid cookie from browser
				http.SetCookie(w, &http.Cookie{
					Name:   SessionCookieName,
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// 4. Resolve associated User profile
			user, err := repository.GetUserByID(ctx, db, session.UserID)
			if err != nil {
				log.Printf("[ERROR] Failed to resolve user identity for session ID %d: %v", session.UserID, err)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// 5. Inject authenticated user profile into request context
			newCtx := context.WithValue(ctx, UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

// GetUserFromContext is a helper function that safely retrieves the active User
// struct from the request context inside handlers.
func GetUserFromContext(ctx context.Context) (*repository.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*repository.User)
	return user, ok
}
