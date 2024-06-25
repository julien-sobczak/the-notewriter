package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikilink(t *testing.T) {
	w := markdown.Wikilink{
		Link: "path/to/file#A section",
		Text: "",
		Line: 1,
	}
	assert.Equal(t, "path/to/file", w.Path())
	assert.Equal(t, "A section", w.Section())
	assert.False(t, w.Anchored())
	assert.False(t, w.Piped())

	w = markdown.Wikilink{
		Link: "#A section",
		Text: "A Section",
		Line: 1,
	}
	assert.Equal(t, "", w.Path())
	assert.Equal(t, "A section", w.Section())
	assert.True(t, w.Anchored())
	assert.True(t, w.Piped())
}

func TestNewWikilink(t *testing.T) {
	tests := []struct {
		name     string
		wikilink string // input
		invalid  bool   // output
		link     string // output
		text     string // output
	}{
		{
			name:     "Invalid",
			wikilink: "not a wikilink",
			invalid:  true,
		},
		{
			name:     "No section",
			wikilink: "[[path/to/file]]",
			link:     "path/to/file",
			text:     "",
		},
		{
			name:     "No text",
			wikilink: "[[path/to/file#A section]]",
			link:     "path/to/file#A section",
			text:     "",
		},
		{
			name:     "Only Section",
			wikilink: "[[#Section]]",
			link:     "#Section",
			text:     "",
		},
		{
			name:     "Link & Text",
			wikilink: "[[file.md#Section|Text]]",
			link:     "file.md#Section",
			text:     "Text",
		},
	}
	for _, tt := range tests {
		actual, err := markdown.NewWikilink(tt.wikilink)
		if tt.invalid {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.link, actual.Link)
			assert.Equal(t, tt.text, actual.Text)
			assert.Equal(t, 0, actual.Line)
		}
	}
}
