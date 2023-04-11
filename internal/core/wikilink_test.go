package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikilink(t *testing.T) {
	w := Wikilink{
		Link: "path/to/file#A section",
		Text: "",
		Line: 1,
	}
	assert.Equal(t, "path/to/file", w.Path())
	assert.Equal(t, "A section", w.Section())
	assert.False(t, w.Anchored())
	assert.False(t, w.Piped())

	w = Wikilink{
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
		actual, err := NewWikilink(tt.wikilink)
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

func TestParseWikilinks(t *testing.T) {
	text := `
[[1234abcdABCD-_]]
[[file.md]]
[[path/to/file.md]]
[[path/to/file]]
[[a|b]]
[[a.md|A long text]]
[[a.md#B]]
	`

	actual := ParseWikilinks(text)
	require.Len(t, actual, 7)

	expected := []*Wikilink{
		{
			Link: "1234abcdABCD-_",
			Text: "",
			Line: 2,
		},
		{
			Link: "file.md",
			Text: "",
			Line: 3,
		},
		{
			Link: "path/to/file.md",
			Text: "",
			Line: 4,
		},
		{
			Link: "path/to/file",
			Text: "",
			Line: 5,
		},
		{
			Link: "a",
			Text: "b",
			Line: 6,
		},
		{
			Link: "a.md",
			Text: "A long text",
			Line: 7,
		},
		{
			Link: "a.md#B",
			Text: "",
			Line: 8,
		},
	}
	assert.Equal(t, expected, actual)
}
