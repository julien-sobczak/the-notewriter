package core

import (
	"testing"
)

func TestTitleModification(t *testing.T) {
	// Test that when preserveShorthand is false, the emoji gets removed from the title
	// But when preserveShorthand is true (or unset), the emoji stays
	
	// Create test attributes 
	attributes := ConfigAttributes{
		"status": &ConfigAttribute{
			Name: "status",
			Type: "string",
			Shorthands: map[string]string{
				"📋": "todo",
				"🕒": "in-progress", 
				"⛔": "blocked",
				"✅": "done",
			},
			PreserveShorthand: BoolPointer(false), // Remove from title
		},
		"rating": &ConfigAttribute{
			Name: "rating",
			Type: "string",
			Shorthands: map[string]string{
				"★":   "★",
				"★★":  "★★",
				"★★★": "★★★",
			},
			PreserveShorthand: BoolPointer(true), // Keep in title
		},
	}

	tests := []struct {
		name           string
		title          string
		expectedTitle  string
		expectedAttrs  map[string]interface{}
	}{
		{
			name:  "Status shorthand removed",
			title: "Add Zen Mode 🕒",
			expectedTitle: "Add Zen Mode",
			expectedAttrs: map[string]interface{}{"status": "in-progress"},
		},
		{
			name:  "Rating shorthand preserved",
			title: "Great Book ★★★",
			expectedTitle: "Great Book ★★★",
			expectedAttrs: map[string]interface{}{"rating": "★★★"},
		},
		{
			name:  "Status shorthand at beginning",
			title: "🕒 Add Zen Mode",
			expectedTitle: "Add Zen Mode",
			expectedAttrs: map[string]interface{}{"status": "in-progress"},
		},
		{
			name:  "Multiple spaces cleaned up",
			title: "Add    Zen    Mode   🕒   ",
			expectedTitle: "Add Zen Mode",
			expectedAttrs: map[string]interface{}{"status": "in-progress"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test extraction
			extracted := ExtractShorthandsFromTitle(test.title, attributes)
			for key, expected := range test.expectedAttrs {
				if actual, exists := extracted[key]; !exists || actual != expected {
					t.Errorf("Expected %s=%v, got %v", key, expected, actual)
				}
			}

			// Test title modification  
			modified := RemoveShorthandsFromTitle(test.title, attributes)
			if modified != test.expectedTitle {
				t.Errorf("Expected title %q, got %q", test.expectedTitle, modified)
			}
		})
	}
}