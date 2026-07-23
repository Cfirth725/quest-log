// ====================================================================
// -- INGESTION DOMAIN: SANITIZER & INPUT HYGIENE ENGINE --
// ====================================================================

// Package ingest coordinates external payload parsing, string sanitization,
// and schema verification before records hit the transactional DAO layer.
package ingest

import (
	"fmt"
	"html"
	"log"
	"strings"
	"unicode"
)

// ====================================================================
// -- SYSTEM CONSTANTS & DOMAIN CONSTRAINTS --
// ====================================================================

const (
	// MaxTitleLength defines the strict boundary length for database string fields.
	MaxTitleLength = 255
)

// ====================================================================
// -- INPUT HYGIENE & VALIDATION LOGIC --
// ====================================================================

// SanitizeTitle strips unwanted control characters, trims whitespace,
// escapes HTML, and enforces length constraint boundaries.
func SanitizeTitle(raw string) (string, error) {
	log.Printf("[REALTIME] Executing input hygiene pass on raw title payload...")

	// 1. Remove non-printable control characters while preserving standard space characters
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, raw)

	// 2. Collapse leading and trailing whitespace
	trimmed := strings.TrimSpace(cleaned)

	if trimmed == "" {
		log.Printf("[ERROR] Validation block: quest title payload is empty or whitespace-only")
		return "", fmt.Errorf("validation fault: quest title cannot be empty")
	}

	if len(trimmed) > MaxTitleLength {
		log.Printf("[ERROR] Validation block: title length (%d) exceeds max threshold (%d)", len(trimmed), MaxTitleLength)
		return "", fmt.Errorf("validation fault: quest title exceeds maximum length of %d characters", MaxTitleLength)
	}

	// 3. Escape HTML entities to neutralize potential XSS vectors in external text payloads
	escaped := html.EscapeString(trimmed)

	log.Printf("[OK] Input hygiene pass completed successfully: title sanitized")
	return escaped, nil
}
