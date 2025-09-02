package main

import (
	"strings"
	"testing"
)

func TestFormatMarkdownText(t *testing.T) {
	tests := []struct {
		name                  string
		input                 string
		shouldContainMarkdown bool
	}{
		{
			name:                  "plain text",
			input:                 "This is plain text",
			shouldContainMarkdown: false,
		},
		{
			name:                  "italic text with asterisks",
			input:                 "This has *italic* text",
			shouldContainMarkdown: false, // Should remove the asterisks
		},
		{
			name:                  "italic text with underscores",
			input:                 "This has _italic_ text",
			shouldContainMarkdown: false, // Should remove the underscores
		},
		{
			name:                  "bold text with double asterisks",
			input:                 "This has **bold** text",
			shouldContainMarkdown: false, // Should remove the asterisks
		},
		{
			name:                  "bold text with double underscores",
			input:                 "This has __bold__ text",
			shouldContainMarkdown: false, // Should remove the underscores
		},
		{
			name:                  "mixed formatting",
			input:                 "This has *italic* and **bold** text",
			shouldContainMarkdown: false, // Should remove all markdown
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMarkdownText(tt.input)

			// Check that markdown characters are removed
			if !tt.shouldContainMarkdown {
				if strings.Contains(result, "*") {
					t.Errorf("formatMarkdownText(%q) still contains asterisks: %q", tt.input, result)
				}
				if strings.Contains(result, "_") {
					t.Errorf("formatMarkdownText(%q) still contains underscores: %q", tt.input, result)
				}
			}

			// Check that content is preserved (by checking word count is similar)
			inputWords := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(tt.input, "*", ""), "_", ""))
			resultWords := strings.Fields(stripColorCodes(result))

			if len(inputWords) != len(resultWords) {
				t.Errorf("formatMarkdownText(%q) changed word count: input %d words, result %d words",
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

func TestFormatMarkdownTextRemovesMarkdown(t *testing.T) {
	// Test that markdown emphasis markers are removed
	tests := []struct {
		input          string
		hasAsterisks   bool
		hasUnderscores bool
	}{
		{"This has *italic* text", false, false},
		{"This has **bold** text", false, false},
		{"This has _italic_ text", false, false},
		{"This has __bold__ text", false, false},
		{"This has *italic* and **bold** text", false, false},
		{"Plain text without markdown", false, false},
	}

	for _, tt := range tests {
		result := formatMarkdownText(tt.input)

		if strings.Contains(result, "*") && !tt.hasAsterisks {
			t.Errorf("formatMarkdownText(%q) still contains asterisks: %q", tt.input, result)
		}

		if strings.Contains(result, "_") && !tt.hasUnderscores {
			t.Errorf("formatMarkdownText(%q) still contains underscores: %q", tt.input, result)
		}
	}
}