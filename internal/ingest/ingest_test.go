// ====================================================================
// -- INGESTION DOMAIN: EXTRACTION & SANITIZER UNIT TESTS --
// ====================================================================

package ingest

import (
	"testing"
)

// ====================================================================
// -- TABLE-DRIVEN PARSER & EDGE-CASE VALIDATION TESTS --
// ====================================================================

func TestParseJSONPayload_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectError   bool
		expectedCount int
		validateFirst func(t *testing.T, q ExtractedQuest)
	}{
		{
			name: "Happy Path: Standard Array Payload",
			jsonInput: `[
				{
					"title": "  Refactor DNS Config  ",
					"category": "Infrastructure",
					"difficulty": "medium",
					"quest_type": "One-Time",
					"is_non_negotiable": true
				}
			]`,
			expectError:   false,
			expectedCount: 1,
			validateFirst: func(t *testing.T, q ExtractedQuest) {
				if q.Title != "Refactor DNS Config" {
					t.Errorf("expected trimmed title, got '%s'", q.Title)
				}
				if q.Difficulty != 2 || q.BaseXP != 5 {
					t.Errorf("expected difficulty tier 2 and 5 XP, got tier %d with %d XP", q.Difficulty, q.BaseXP)
				}
			},
		},
		{
			name: "Fallback: Single Object (Not Array)",
			jsonInput: `{
				"title": "Single Quest Payload",
				"category": "General",
				"difficulty": "easy",
				"quest_type": "Daily"
			}`,
			expectError:   false,
			expectedCount: 1,
			validateFirst: func(t *testing.T, q ExtractedQuest) {
				if q.Title != "Single Quest Payload" {
					t.Errorf("expected title 'Single Quest Payload', got '%s'", q.Title)
				}
			},
		},
		{
			name: "Security: HTML Injection Sanitization",
			jsonInput: `[{
				"title": "<script>alert('xss')</script> Clean Quest",
				"category": "Security",
				"difficulty": 1
			}]`,
			expectError:   false,
			expectedCount: 1,
			validateFirst: func(t *testing.T, q ExtractedQuest) {
				expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; Clean Quest"
				if q.Title != expected {
					t.Errorf("expected HTML escaped string, got '%s'", q.Title)
				}
			},
		},
		{
			name: "Edge Case: Repeating Quest Interval Mapping",
			jsonInput: `[{
				"title": "Water Household Plants",
				"category": "Home",
				"difficulty": "easy",
				"quest_type": "Repeating",
				"repeat_interval_days": 3
			}]`,
			expectError:   false,
			expectedCount: 1,
			validateFirst: func(t *testing.T, q ExtractedQuest) {
				if !q.RepeatIntervalDays.Valid || q.RepeatIntervalDays.Int64 != 3 {
					t.Errorf("expected RepeatIntervalDays = 3, got %v", q.RepeatIntervalDays)
				}
			},
		},
		{
			name:        "Failure: Empty Title Payload",
			jsonInput:   `[{"title": "   ", "category": "Test", "difficulty": 1}]`,
			expectError: true,
		},
		{
			name:        "Failure: Invalid Difficulty Label",
			jsonInput:   `[{"title": "Valid Title", "difficulty": "ultra-hard"}]`,
			expectError: true,
		},
		{
			name:        "Failure: Malformed JSON Syntax",
			jsonInput:   `[{"title": "Broken JSON", "difficulty": 1,}]`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseJSONPayload([]byte(tt.jsonInput))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for case '%s', but got nil", tt.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for case '%s': %v", tt.name, err)
			}

			if len(results) != tt.expectedCount {
				t.Fatalf("expected %d extracted quest(s), got %d", tt.expectedCount, len(results))
			}

			if tt.validateFirst != nil && len(results) > 0 {
				tt.validateFirst(t, results[0])
			}
		})
	}
}
