// ====================================================================
// -- MIDDLEWARE DOMAIN: SECURITY & ACCESS CONTROL GATES --
// ====================================================================

// Package middleware provides HTTP interceptors, token verification gates,
// and zero-trust authentication wrappers for the Quest Log service.
package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

// ====================================================================
// -- SECURITY RESPONSE ENTITIES --
// ====================================================================

// ErrorResponse defines the standard JSON payload structure returned
// when access control gates reject an incoming request.
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ====================================================================
// -- 1. HEADLESS API KEY AUTHENTICATION MIDDLEWARE --
// ====================================================================

// APIKeyAuth validates inbound requests against the configured QUESTLOG_API_KEY
// secret. It accepts token tokens via 'X-API-Key' or 'Authorization: Bearer <key>' headers.
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedKey := os.Getenv("QUESTLOG_API_KEY")
		if expectedKey == "" {
			expectedKey = "ql_dev_default_key" // Fallback secret for local development
		}

		clientKey := r.Header.Get("X-API-Key")
		if clientKey == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				clientKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if clientKey == "" || clientKey != expectedKey {
			log.Printf("[SECURE] Access block on %s: invalid or missing API key token from %s", r.URL.Path, r.RemoteAddr)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Status:  "error",
				Message: "unauthorized: invalid or missing API key",
			})
			return
		}

		next.ServeHTTP(w, r)
	}
}
