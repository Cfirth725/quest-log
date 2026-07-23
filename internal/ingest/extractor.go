// ====================================================================
// -- INGESTION DOMAIN: JSON EXTRACTOR & SCHEMA VALIDATOR --
// ====================================================================

// Package ingest coordinates external payload parsing, string sanitization,
// and schema verification before records hit the transactional DAO layer.
package ingest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ====================================================================
// -- DOMAIN STRUCTURES & DATA TRANSFER OBJECTS (DTOs) --
// ====================================================================

// RawQuestPayload represents the external JSON schema for bulk quest seeding.
type RawQuestPayload struct {
	Title              string      `json:"title"`
	Category           string      `json:"category"`
	Difficulty         interface{} `json:"difficulty"` // Accepts flexible inputs: string ("easy") or int (1)
	QuestType          string      `json:"quest_type"` // "One-Time", "Daily", "Repeating", "Weekly"
	IsNonNegotiable    bool        `json:"is_non_negotiable"`
	RepeatIntervalDays *int64      `json:"repeat_interval_days,omitempty"`
	ResetDayOfWeek     *int        `json:"reset_day_of_week,omitempty"`
}

// ExtractedQuest represents the fully validated, DAO-ready record payload.
type ExtractedQuest struct {
	Title              string
	CategoryName       string
	Difficulty         int
	BaseXP             int
	QuestType          string
	IsNonNegotiable    int
	RepeatIntervalDays sql.NullInt64
	ResetDayOfWeek     int
}

// ====================================================================
// -- PARSING & EXTRACTION PIPELINE --
// ====================================================================

// ParseJSONPayload parses a raw byte slice containing a single quest or a JSON array of quests.
func ParseJSONPayload(data []byte) ([]ExtractedQuest, error) {
	log.Printf("[INIT] Beginning JSON payload extraction sweep...")

	// Try parsing as an array first
	var rawArray []RawQuestPayload
	if err := json.Unmarshal(data, &rawArray); err != nil {
		log.Printf("[REALTIME] Array unmarshal attempt failed, attempting single object fallback parse...")

		// Fallback: try parsing as a single object
		var single RawQuestPayload
		if errSingle := json.Unmarshal(data, &single); errSingle != nil {
			log.Printf("[ERROR] Payload extraction fault: raw bytes match neither JSON array nor single quest object")
			return nil, fmt.Errorf("json parse error: payload is neither array nor single quest object: %w", err)
		}
		rawArray = append(rawArray, single)
	}

	log.Printf("[REALTIME] Unmarshaled %d raw quest record(s). Proceeding to validation pipeline...", len(rawArray))

	var results []ExtractedQuest
	for i, raw := range rawArray {
		extracted, err := ValidateAndConvert(raw)
		if err != nil {
			log.Printf("[ERROR] Validation pipeline blocked item at index %d ('%s'): %v", i, raw.Title, err)
			return nil, fmt.Errorf("validation error in item index %d ('%s'): %w", i, raw.Title, err)
		}
		results = append(results, extracted)
	}

	log.Printf("[OK] Extraction sweep complete: %d quest payload(s) validated and ready for DAO batch execution", len(results))
	return results, nil
}

// ====================================================================
// -- SCHEMA VALIDATION & TYPE TRANSFORMATION --
// ====================================================================

// ValidateAndConvert transforms a RawQuestPayload into a type-safe ExtractedQuest object.
func ValidateAndConvert(raw RawQuestPayload) (ExtractedQuest, error) {
	var result ExtractedQuest

	// 1. Sanitize & Validate Title via input hygiene engine
	title, err := SanitizeTitle(raw.Title)
	if err != nil {
		return result, err
	}
	result.Title = title

	// 2. Category fallback mapping
	result.CategoryName = strings.TrimSpace(raw.Category)
	if result.CategoryName == "" {
		result.CategoryName = "Uncategorized"
	}

	// 3. Map Difficulty to Tier (1, 2, 3) and Hard-Coded Economy XP (1, 5, 10)
	tier, xp, err := parseDifficulty(raw.Difficulty)
	if err != nil {
		return result, err
	}
	result.Difficulty = tier
	result.BaseXP = xp

	// 4. Validate Quest Type boundaries
	qType := strings.TrimSpace(raw.QuestType)
	switch strings.ToLower(qType) {
	case "one-time", "onetime", "single":
		result.QuestType = "One-Time"
	case "daily":
		result.QuestType = "Daily"
	case "repeating", "recurring":
		result.QuestType = "Repeating"
	case "weekly":
		result.QuestType = "Weekly"
	default:
		// Default to One-Time if unrecognized string is passed
		result.QuestType = "One-Time"
	}

	// 5. Non-Negotiable priority shield flag mapping
	if raw.IsNonNegotiable {
		result.IsNonNegotiable = 1
	} else {
		result.IsNonNegotiable = 0
	}

	// 6. Handle Repeating Interval bounds
	if result.QuestType == "Repeating" && raw.RepeatIntervalDays != nil && *raw.RepeatIntervalDays > 0 {
		result.RepeatIntervalDays = sql.NullInt64{Int64: *raw.RepeatIntervalDays, Valid: true}
	}

	// 7. Handle Weekly Reset Day index mapping (0 = Sunday ... 6 = Saturday)
	if result.QuestType == "Weekly" && raw.ResetDayOfWeek != nil {
		if *raw.ResetDayOfWeek >= 0 && *raw.ResetDayOfWeek <= 6 {
			result.ResetDayOfWeek = *raw.ResetDayOfWeek
		}
	}

	return result, nil
}

// parseDifficulty handles flexible difficulty inputs (int or string) and maps to Economy XP.
func parseDifficulty(diff interface{}) (tier int, xp int, err error) {
	switch v := diff.(type) {
	case float64: // JSON numbers unmarshal to float64 by default
		tier = int(v)
	case int:
		tier = v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "easy", "coin":
			tier = 1
		case "2", "medium", "standard", "moneybag":
			tier = 2
		case "3", "hard", "crown", "boss":
			tier = 3
		default:
			log.Printf("[ERROR] Schema evaluation fault: unrecognized difficulty string '%s'", v)
			return 0, 0, fmt.Errorf("unrecognized difficulty label: '%s'", v)
		}
	default:
		log.Printf("[ERROR] Schema evaluation fault: unsupported difficulty data type")
		return 0, 0, fmt.Errorf("invalid difficulty payload type")
	}

	// Map verified tier index to Hard-Coded Economy XP bounds
	switch tier {
	case 1:
		return 1, 1, nil // Tier 1: 🪙 Coin (1 XP)
	case 2:
		return 2, 5, nil // Tier 2: 💰 Moneybag (5 XP)
	case 3:
		return 3, 10, nil // Tier 3: 👑 Crown (10 XP)
	default:
		log.Printf("[ERROR] Economy evaluation fault: difficulty tier index %d outside valid [1-3] boundary", tier)
		return 0, 0, fmt.Errorf("invalid difficulty level: %d (must be 1, 2, or 3)", tier)
	}
}
