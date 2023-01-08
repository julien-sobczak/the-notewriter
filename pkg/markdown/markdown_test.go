package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSupportedFormats(t *testing.T) {
	var tests = []struct {
		name string // name
	}{
		{"BasicMarkdown"},
		{"Headings"},
		{"Emphasis"},
		{"Blockquotes"},
		{"Lists"},
		{"CodeBlocks"},
		{"Images"},
		{"Links"},
		{"Tables"},
		{"Tasks"},
	}
	for _, tt := range tests {
		input, err := os.ReadFile(filepath.Join("testdata", tt.name+".md"))
		require.NoError(t, err)
		outputMarkdown, err := os.ReadFile(filepath.Join("testdata", tt.name+".md.markdown"))
		require.NoError(t, err)
		outputHTML, err := os.ReadFile(filepath.Join("testdata", tt.name+".md.html"))
		require.NoError(t, err)
		outputText, err := os.ReadFile(filepath.Join("testdata", tt.name+".md.txt"))
		require.NoError(t, err)

		t.Run(tt.name+"ToMarkdown", func(t *testing.T) {
			actualMarkdown := markdown.ToMarkdown(string(input))
			assert.Equal(t, strings.TrimSpace(string(outputMarkdown)), actualMarkdown)
		})

		t.Run(tt.name+"ToHTML", func(t *testing.T) {
			actualHTML := markdown.ToHTML(string(input))
			assert.Equal(t, strings.TrimSpace(string(outputHTML)), actualHTML)

		})

		t.Run(tt.name+"ToText", func(t *testing.T) {
			actualText := markdown.ToText(string(input))
			assert.Equal(t, strings.TrimSpace(string(outputText)), actualText)
		})
	}
}

func TestAlignHeadings(t *testing.T) {
	var tests = []struct {
		name     string // name
		input    string
		expected string
	}{
		{
			name: "No headings",
			input: `
blabla

blablabla

blablablabla
`,
			expected: `
blabla

blablabla

blablablabla
`,
		},

		{
			name: "Basic example",
			input: `
blabla
#### Blablabla
blablabla
##### Blablablabla
blablablabla
`,
			expected: `
blabla
## Blablabla
blablabla
### Blablablabla
blablablabla
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := markdown.AlignHeadings(tt.input)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}
