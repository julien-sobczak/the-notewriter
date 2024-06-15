package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportedFormats(t *testing.T) {
	var tests = []struct {
		name string // name
	}{
		{"BasicMarkdown"},
		{"Headings"},
		{"Emphasis"},
		{"Blockquotes"},
		{"Lists"},
		{"CodeBlocks"},
		{"Medias"},
		{"Links"},
		{"Tables"},
		{"Tasks"},
	}
	for _, tt := range tests {
		input, err := os.ReadFile(filepath.Join("testdata/TestConversion", tt.name+".md"))
		require.NoError(t, err)
		outputMarkdown, err := os.ReadFile(filepath.Join("testdata/TestConversion", tt.name+".md.markdown"))
		require.NoError(t, err)
		outputHTML, err := os.ReadFile(filepath.Join("testdata/TestConversion", tt.name+".md.html"))
		require.NoError(t, err)
		outputText, err := os.ReadFile(filepath.Join("testdata/TestConversion", tt.name+".md.txt"))
		require.NoError(t, err)

		document := markdown.Document(input)

		t.Run(tt.name+"ToCleanMarkdown", func(t *testing.T) {
			actualMarkdown := document.ToCleanMarkdown()
			assert.Equal(t, strings.TrimSpace(string(outputMarkdown)), string(actualMarkdown))
		})

		t.Run(tt.name+"ToHTML", func(t *testing.T) {
			actualHTML := document.ToHTML()
			assert.Equal(t, strings.TrimSpace(string(outputHTML)), actualHTML)

		})

		t.Run(tt.name+"ToText", func(t *testing.T) {
			actualText := document.ToText()
			assert.Equal(t, strings.TrimSpace(string(outputText)), actualText)
		})
	}
}
