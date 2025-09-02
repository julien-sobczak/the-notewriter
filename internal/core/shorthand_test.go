package core

import (
	"testing"
)

func TestShorthandExtraction(t *testing.T) {
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
			PreserveShorthand: BoolPointer(false),
		},
		"rating": &ConfigAttribute{
			Name: "rating",
			Type: "string",
			Shorthands: map[string]string{
				"★":   "★",
				"★★":  "★★",
				"★★★": "★★★",
			},
			PreserveShorthand: BoolPointer(true),
		},
	}

	// Test extraction
	tests := []struct {
		title           string
		expectedAttrs   map[string]interface{}
		expectedTitle   string
	}{
		{
			title: "Add Zen Mode 🕒",
			expectedAttrs: map[string]interface{}{
				"status": "in-progress",
			},
			expectedTitle: "Add Zen Mode",
		},
		{
			title: "Thinking Fast & Slow ★★★",
			expectedAttrs: map[string]interface{}{
				"rating": "★★★",
			},
			expectedTitle: "Thinking Fast & Slow ★★★", // preserved
		},
		{
			title: "Complete Project ✅",
			expectedAttrs: map[string]interface{}{
				"status": "done",
			},
			expectedTitle: "Complete Project",
		},
		{
			title: "Simple Book",
			expectedAttrs: map[string]interface{}{},
			expectedTitle: "Simple Book",
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			// Test extraction
			extracted := ExtractShorthandsFromTitle(test.title, attributes)
			if len(extracted) != len(test.expectedAttrs) {
				t.Errorf("Expected %d attributes, got %d", len(test.expectedAttrs), len(extracted))
			}
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

func TestDefaultValueApplication(t *testing.T) {
	// Create test config
	attributes := ConfigAttributes{
		"rating": &ConfigAttribute{
			Name:         "rating",
			Type:         "string",
			DefaultValue: "★★",
		},
	}

	noteType := &ConfigType{
		Name: "BookReview",
		Attributes: []ConfigTypeAttribute{
			{
				Name:     "rating",
				Optional: BoolPointer(false), // required
			},
		},
	}

	// Test with missing attribute
	attrs := make(AttributeSet)
	ApplyDefaultAttributeValues(noteType, attrs, attributes)

	if rating, exists := attrs["rating"]; !exists || rating != "★★" {
		t.Errorf("Expected default rating ★★, got %v", rating)
	}

	// Test with existing attribute (should not override)
	attrs2 := AttributeSet{"rating": "★★★"}
	ApplyDefaultAttributeValues(noteType, attrs2, attributes)

	if rating, exists := attrs2["rating"]; !exists || rating != "★★★" {
		t.Errorf("Expected existing rating ★★★, got %v", rating)
	}
	
	// Additional test: check that the Find function works
	foundAttr, found := attributes.Find("rating")
	if !found {
		t.Errorf("Could not find rating attribute")
	}
	if foundAttr.DefaultValue != "★★" {
		t.Errorf("Expected default value ★★, got %v", foundAttr.DefaultValue)
	}
}