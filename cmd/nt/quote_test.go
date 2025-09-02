package main

import (
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
)

func TestMarkdownDocumentToANSI(t *testing.T) {
	tests := []struct {
		name                  string
		input                 string
		shouldContainMarkdown bool
		shouldContainANSI     bool
	}{
		{
			name:                  "plain text",
			input:                 "This is plain text",
			shouldContainMarkdown: false,
			shouldContainANSI:     false,
		},
		{
			name:                  "italic text with asterisks",
			input:                 "This has *italic* text",
			shouldContainMarkdown: false, // Should remove the asterisks
			shouldContainANSI:     true,  // Should contain ANSI escape codes
		},
		{
			name:                  "italic text with underscores",
			input:                 "This has _italic_ text",
			shouldContainMarkdown: false, // Should remove the underscores
			shouldContainANSI:     true,  // Should contain ANSI escape codes
		},
		{
			name:                  "bold text with double asterisks",
			input:                 "This has **bold** text",
			shouldContainMarkdown: false, // Should remove the asterisks
			shouldContainANSI:     true,  // Should contain ANSI escape codes
		},
		{
			name:                  "bold text with double underscores",
			input:                 "This has __bold__ text",
			shouldContainMarkdown: false, // Should remove the underscores
			shouldContainANSI:     true,  // Should contain ANSI escape codes
		},
		{
			name:                  "mixed formatting",
			input:                 "This has *italic* and **bold** text",
			shouldContainMarkdown: false, // Should remove all markdown
			shouldContainANSI:     true,  // Should contain ANSI escape codes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := markdown.Document(tt.input)
			result := doc.ToANSI()

			// Check that markdown characters are removed
			if !tt.shouldContainMarkdown {
				if strings.Contains(result, "*") {
					t.Errorf("ToANSI(%q) still contains asterisks: %q", tt.input, result)
				}
				if strings.Contains(result, "_") {
					t.Errorf("ToANSI(%q) still contains underscores: %q", tt.input, result)
				}
			}

			// Check for ANSI escape codes where expected
			if tt.shouldContainANSI {
				if !strings.Contains(result, "\x1b[") {
					t.Errorf("ToANSI(%q) should contain ANSI escape codes but doesn't: %q", tt.input, result)
				}
			} else {
				if strings.Contains(result, "\x1b[") {
					t.Errorf("ToANSI(%q) should not contain ANSI escape codes but does: %q", tt.input, result)
				}
			}

			// Check that content is preserved (by checking word count is similar)
			inputWords := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(tt.input, "*", ""), "_", ""))
			resultWords := strings.Fields(stripColorCodes(result))

			if len(inputWords) != len(resultWords) {
				t.Errorf("ToANSI(%q) changed word count: input %d words, result %d words",
					tt.input, len(inputWords), len(resultWords))
			}
		})
	}
}

// stripColorCodes removes ANSI color codes from a string for testing
func stripColorCodes(s string) string {
	// Simple regex replacement might be complex, so let's use a basic approach
	// Remove common ANSI escape sequences
	result := s
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}