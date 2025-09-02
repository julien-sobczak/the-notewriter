package markdown_test

import (
	"testing"

	"github.com/fatih/color"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestIsHeading(t *testing.T) {
	ok, _, _ := markdown.IsHeading("Some text")
	assert.False(t, ok)

	ok, _, _ = markdown.IsHeading("")
	assert.False(t, ok)

	ok, title, level := markdown.IsHeading("# Heading 1")
	assert.True(t, ok)
	assert.Equal(t, "Heading 1", title)
	assert.Equal(t, 1, level)

	ok, title, level = markdown.IsHeading("## Heading 2")
	assert.True(t, ok)
	assert.Equal(t, "Heading 2", title)
	assert.Equal(t, 2, level)

	ok, title, level = markdown.IsHeading("### Heading 3")
	assert.True(t, ok)
	assert.Equal(t, "Heading 3", title)
	assert.Equal(t, 3, level)

	ok, title, level = markdown.IsHeading("#### Heading 4")
	assert.True(t, ok)
	assert.Equal(t, "Heading 4", title)
	assert.Equal(t, 4, level)

	ok, title, level = markdown.IsHeading("##### Heading 5")
	assert.True(t, ok)
	assert.Equal(t, "Heading 5", title)
	assert.Equal(t, 5, level)

	ok, title, level = markdown.IsHeading("###### Heading 6")
	assert.True(t, ok)
	assert.Equal(t, "Heading 6", title)
	assert.Equal(t, 6, level)

	// Sub levels are not currently supported
	ok, _, _ = markdown.IsHeading("####### Heading 7")
	assert.False(t, ok)
}

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
