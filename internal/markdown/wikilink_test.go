package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
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

func TestWikilinks(t *testing.T) {

	t.Run("Syntaxes", func(t *testing.T) {

		text := markdown.Document(`
[[1234abcdABCD-_]]
[[file.md]]
[[path/to/file.md]]
[[path/to/file]]
[[a|b]]
[[a.md|A long text]]
[[a.md#B]]
	`)

		actual := text.Wikilinks()
		require.Len(t, actual, 7)

		expected := []markdown.Wikilink{
			{
				Link: "1234abcdABCD-_",
				Text: "1234abcdABCD-_",
				Line: 2,
			},
			{
				Link: "file.md",
				Text: "file.md",
				Line: 3,
			},
			{
				Link: "path/to/file.md",
				Text: "path/to/file.md",
				Line: 4,
			},
			{
				Link: "path/to/file",
				Text: "path/to/file",
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
				Text: "a.md#B",
				Line: 8,
			},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("Example", func(t *testing.T) {
		filename := testutil.SetUpFromGoldenFileNamed(t, "TestMarkdown/links.md")
		md, err := markdown.ParseFile(filename)
		require.NoError(t, err)

		actual := md.Body.Wikilinks()
		require.Len(t, actual, 4)

		expected := []markdown.Wikilink{
			{
				Link: "links#Wikilinks",
				Text: "links#Wikilinks",
				Line: 5,
			},
			{
				Link: "testdata/TestMarkdown/links",
				Text: "testdata/TestMarkdown/links",
				Line: 13,
			},
			{
				Link: "TestMarkdown/links#Wikilinks",
				Text: "TestMarkdown/links#Wikilinks",
				Line: 13,
			},
			{
				Link: "TestMarkdown/links#Change Display Text",
				Text: "displayed text",
				Line: 17,
			},
		}
		assert.Equal(t, expected, actual)

	})
}
