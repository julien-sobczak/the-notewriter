package markdown_test

import (
	"testing"

	"github.com/fatih/color"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestMarkdownDocument(t *testing.T) {

	t.Run("ToANSI", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "plain text",
				input:    "This is plain text",
				expected: "This is plain text",
			},
			{
				name:     "italic text with asterisks",
				input:    "This has *italic* text",
				expected: "This has \x1b[3mitalic\x1b[0m text",
			},
			{
				name:     "italic text with underscores",
				input:    "This has _italic_ text",
				expected: "This has \x1b[3mitalic\x1b[0m text",
			},
			{
				name:     "bold text with double asterisks",
				input:    "This has **bold** text",
				expected: "This has \x1b[1mbold\x1b[0m text",
			},
			{
				name:     "bold text with double underscores",
				input:    "This has __bold__ text",
				expected: "This has \x1b[1mbold\x1b[0m text",
			},
			{
				name:     "mixed formatting",
				input:    "This has *italic* and **bold** text",
				expected: "This has \x1b[3mitalic\x1b[0m and \x1b[1mbold\x1b[0m text",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// fatih/color checks for non-tty output like test environment and does nothing
				initialNoColor := color.NoColor
				color.NoColor = false
				t.Cleanup(func() {
					color.NoColor = initialNoColor
				})

				doc := markdown.Document(tt.input)
				result := doc.ToANSI()
				assert.Equal(t, tt.expected, result, "ToANSI(%q) = %q; want %q", tt.input, result, tt.expected)
			})
		}
	})

}
