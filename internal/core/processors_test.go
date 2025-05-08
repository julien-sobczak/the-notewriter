package core_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestExtractDateFromTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    markdown.Document // input
		expected string            // expected date
		found    bool              // whether a date was found
	}{
		{
			name:     "Valid full date",
			input:    "Journal: 2023-10-15.",
			expected: "2023-10-15",
			found:    true,
		},
		{
			name:     "Valid year and month",
			input:    "Journal: The event happened in 2023-10",
			expected: "2023-10",
			found:    true,
		},
		{
			name:     "Valid year only",
			input:    "Journal: We Are in 2023",
			expected: "2023",
			found:    true,
		},
		{
			name:     "Multiple dates, pick first",
			input:    "Journal: 2023-10-15 and 2022-05-01.",
			expected: "2023-10-15",
			found:    true,
		},
		{
			name:     "Invalid date format but valid year",
			input:    "Journal: The date is 15-10-2023.",
			expected: "2023",
			found:    true,
		},
		{
			name:     "No date present",
			input:    "Journal: No date",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, found := core.ExtractDateFromTitle(tt.input)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.found, found)
		})
	}
}

func TestQuoteRewriterPreprocessor(t *testing.T) {
	tests := []struct {
		name         string
		inputBody    string
		expectedBody string
	}{
		{
			name:         "Regular text is prefixed with quotes",
			inputBody:    "This is a regular line\nAnother regular line",
			expectedBody: "> This is a regular line\n> Another regular line",
		},
		{
			name:         "Empty lines remain unchanged",
			inputBody:    "First line\n\nThird line",
			expectedBody: "> First line\n\n> Third line",
		},
		{
			name:         "Already quoted lines remain unchanged",
			inputBody:    "> Already quoted\nNot quoted",
			expectedBody: "> Already quoted\n> Not quoted",
		},
		{
			name:         "Tags/attributes lines are not quoted",
			inputBody:    "Regular text\n`#tag1` `#tag2` `@attribute1:value1`",
			expectedBody: "> Regular text\n`#tag1` `#tag2` `@attribute1:value1`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			// IMPROVEMENT: Use builder or factory pattern for creating test data
			note := &core.ParsedNote{
				Body: markdown.Document(tt.inputBody),
			}
			file := &core.ParsedFile{}

			// Call the function
			resultNotes, err := core.QuoteRewriterPreprocessor(file, note)

			// Check results
			assert.NoError(t, err)
			assert.Len(t, resultNotes, 1)
			assert.Equal(t, tt.expectedBody, resultNotes[0].Body.String())
		})
	}
}
