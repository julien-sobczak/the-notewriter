package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentLines(t *testing.T) {

	t.Run("Simple Markdown", func(t *testing.T) {
		doc := markdown.Document(text.UnescapeTestContent(`# A heading

A paragraph.
`))
		actual := doc.Lines()

		expected := []markdown.Line{
			{Number: 1, Text: "# A heading", InsideCodeBlock: false},
			{Number: 2, Text: "", InsideCodeBlock: false},
			{Number: 3, Text: "A paragraph.", InsideCodeBlock: false},
			{Number: 4, Text: "", InsideCodeBlock: false},
		}
		require.Len(t, actual, len(expected))
		for i, line := range expected {
			assert.Equal(t, line.Text, actual[i].Text, "Line %d text mismatch", i+1)
			assert.Equal(t, line.Number, actual[i].Number, "Line %d number mismatch", i+1)
			assert.Equal(t, line.InsideCodeBlock, actual[i].InsideCodeBlock, "Line %d code block status mismatch", i+1)
		}
		// Check links...
		// for the first line
		assert.Nil(t, actual[0].Prev)
		assert.NotNil(t, actual[0].Next)
		assert.Equal(t, actual[0], actual[0].Next.Prev)
		// and the last line
		assert.NotNil(t, actual[len(actual)-1].Prev)
		assert.Nil(t, actual[len(actual)-1].Next)
	})

	t.Run("Markdown in Markdown", func(t *testing.T) {
		input := text.UnescapeTestContent(`# Programming Languages
Here is a Go snippet:

‛‛‛
fmt.Println("Hello, World!")
‛‛‛

and here is a Markdown snippet containing a Python script:

‛‛‛‛md
# Python Script
‛‛‛py
print("Hello from Python!")
‛‛‛
‛‛‛‛
`)
		doc := markdown.Document(input)
		actual := doc.Lines()

		expected := []markdown.Line{
			{Number: 1, Text: "# Programming Languages", InsideCodeBlock: false},
			{Number: 2, Text: "Here is a Go snippet:", InsideCodeBlock: false},
			{Number: 3, Text: "", InsideCodeBlock: false},
			{Number: 4, Text: "```", InsideCodeBlock: true},
			{Number: 5, Text: `fmt.Println("Hello, World!")`, InsideCodeBlock: true},
			{Number: 6, Text: "```", InsideCodeBlock: true},
			{Number: 7, Text: "", InsideCodeBlock: false},
			{Number: 8, Text: "and here is a Markdown snippet containing a Python script:", InsideCodeBlock: false},
			{Number: 9, Text: "", InsideCodeBlock: false},
			{Number: 10, Text: "````md", InsideCodeBlock: true},
			{Number: 11, Text: "# Python Script", InsideCodeBlock: true},
			{Number: 12, Text: "```py", InsideCodeBlock: true},
			{Number: 13, Text: `print("Hello from Python!")`, InsideCodeBlock: true},
			{Number: 14, Text: "```", InsideCodeBlock: true},
			{Number: 15, Text: "````", InsideCodeBlock: true},
			{Number: 16, Text: "", InsideCodeBlock: false},
		}
		require.Len(t, actual, len(expected))
		for i, line := range expected {
			assert.Equal(t, line.Text, actual[i].Text, "Line %d text mismatch", i+1)
			assert.Equal(t, line.InsideCodeBlock, actual[i].InsideCodeBlock, "Line %d code block status mismatch", i+1)
		}
	})
}

func TestLine(t *testing.T) {

	t.Run("IsHeading_Basic", func(t *testing.T) {
		tests := []struct {
			lines    markdown.Document
			expected bool
		}{
			{lines: "# Heading 1", expected: true},
			{lines: "## Heading 2", expected: true},
			{lines: "### Heading 3", expected: true},
			{lines: "#### Heading 4", expected: true},
			{lines: "##### Heading 5", expected: true},
			{lines: "###### Heading 6", expected: true},
			{lines: "####### Too many hashes", expected: false},
			{lines: "No heading here", expected: false},
			{lines: " # Leading space", expected: false},
		}
		for _, tt := range tests {
			doc := markdown.Document(text.UnescapeTestContent(tt.lines.String()))
			firstLine := doc.Lines()[0]
			isHeading, _, _ := firstLine.IsHeading()
			assert.Equal(t, tt.expected, isHeading)
		}
	})

	t.Run("IsHeading_Alternate", func(t *testing.T) {
		// heading 1
		heading1 := "Heading level 1\n================"
		doc := markdown.Document(heading1)
		firstLine := doc.Lines()[0]
		isHeading, title, level := firstLine.IsHeading()
		assert.True(t, isHeading)
		assert.Equal(t, "Heading level 1", title)
		assert.Equal(t, 1, level)

		// heading 2
		heading2 := "Heading level 2\n----------------"
		doc = markdown.Document(heading2)
		firstLine = doc.Lines()[0]
		isHeading, title, level = firstLine.IsHeading()
		assert.True(t, isHeading)
		assert.Equal(t, "Heading level 2", title)
		assert.Equal(t, 2, level)
	})

}

func TestLineIterator(t *testing.T) {

	t.Run("HasNext", func(t *testing.T) {
		input := text.UnescapeTestContent(`# Heading
Line 2`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		assert.True(t, it.HasNext())
		it.Next() // Heading
		assert.True(t, it.HasNext())
		it.Next() // Line 2
		assert.False(t, it.HasNext())
	})

	t.Run("SkipBlankLines", func(t *testing.T) {
		input := text.UnescapeTestContent(`
# Heading


Line after blanks.
`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		it.SkipBlankLines()
		line := it.Next()
		assert.Equal(t, "# Heading", line.Text)
		it.Next() // move to blank line
		it.SkipBlankLines()
		line = it.Next()
		assert.Equal(t, "Line after blanks.", line.Text)
	})

	t.Run("NextNonBlankLine", func(t *testing.T) {
		input := text.UnescapeTestContent(`

# Heading
`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		line := it.NextNonBlankLine()
		assert.Equal(t, "# Heading", line.Text)
		line = it.NextNonBlankLine()
		assert.Nil(t, line)
	})

	t.Run("SkipHeading_Basic", func(t *testing.T) {
		input := text.UnescapeTestContent(`
# Heading 1
Some text under heading 1.

## Heading 2
Some text under heading 2.
`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		it.SkipBlankLines()
		line := it.Next()
		assert.Equal(t, "# Heading 1", line.Text)
		it.SkipHeading()
		line = it.Next()
		assert.Equal(t, "Some text under heading 1.", line.Text)
		it.SkipBlankLines()
		line = it.Next()
		assert.Equal(t, "## Heading 2", line.Text)
		it.SkipHeading()
		line = it.Next()
		assert.Equal(t, "Some text under heading 2.", line.Text)
	})

	t.Run("SkipHeading_Alternate", func(t *testing.T) {
		input := text.UnescapeTestContent(`
Heading Level 1
===============

Some text under heading 1.

Heading Level 2
---------------

Some text under heading 2.
`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		it.SkipBlankLines()
		line := it.Next()
		assert.Equal(t, "Heading Level 1", line.Text)
		it.SkipHeading() // Must skip ===
		line = it.NextNonBlankLine()
		assert.Equal(t, "Some text under heading 1.", line.Text)
		it.SkipBlankLines()
		line = it.Next()
		assert.Equal(t, "Heading Level 2", line.Text)
		it.SkipHeading() // Must skip ---
		line = it.NextNonBlankLine()
		assert.Equal(t, "Some text under heading 2.", line.Text)
	})

	t.Run("Matches", func(t *testing.T) {
		input := text.UnescapeTestContent(`# Heading
Line 2
`)
		doc := markdown.Document(input)
		it := doc.Iterator()
		line := it.Next()
		assert.True(t, line.Matches("^#.*$"))
		line = it.Next()
		assert.True(t, line.Matches("^Line [0-9]+$"))
	})

	t.Run("IsHorizontalRule", func(t *testing.T) {
		tests := []struct {
			line     markdown.Line
			expected bool
		}{
			{line: markdown.Line{Text: "---"}, expected: true},
			{line: markdown.Line{Text: "***"}, expected: true},
			{line: markdown.Line{Text: "___"}, expected: true},
			{line: markdown.Line{Text: "- - -"}, expected: true},
			{line: markdown.Line{Text: "* * *"}, expected: true},
			{line: markdown.Line{Text: "_ _ _"}, expected: true},
			{line: markdown.Line{Text: "--"}, expected: false},
			{line: markdown.Line{Text: "**"}, expected: false},
			{line: markdown.Line{Text: "__"}, expected: false},
			{line: markdown.Line{Text: "Not a horizontal rule"}, expected: false},
		}
		for _, tt := range tests {
			actual := tt.line.IsHorizontalRule()
			assert.Equal(t, tt.expected, actual, "Line: %q", tt.line.Text)
		}
	})

}
