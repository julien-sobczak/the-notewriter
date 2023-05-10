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

func TestReplaceAsciidocCharacterSubstitutions(t *testing.T) {
	tests := []struct {
		name     string
		md       string // input
		expected string // output
	}{
		{
			name:     "Empty",
			md:       "",
			expected: "",
		},
		{
			name:     "Basic",
			md:       "(C) (R) (TM) -- ... -> => <- <=",
			expected: "© ® ™ — … → ⇒ ← ⇐",
		},
		{
			name:     "Inline Code",
			md:       "i-- is different from `i--`",
			expected: "i— is different from `i--`",
		},
		{
			name: "Code Blocks",
			md: "" +
				"i--\n" +
				"\n" +
				"    i--\n" +
				"\n" +
				"```c\n" +
				"i--\n" +
				"```\n",
			expected: "" +
				"i—\n" +
				"\n" +
				"    i--\n" +
				"\n" +
				"```c\n" +
				"i--\n" +
				"```\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := markdown.ReplaceAsciidocCharacterSubstitutions(tt.md)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestStripComment(t *testing.T) {
	tests := []struct {
		name    string
		body    string // input
		content string // output
		comment string // output
	}{
		{
			name: "A simple quote without comment",
			body: `
> This is just a quote
`,
			content: "> This is just a quote",
			comment: "",
		},
		{
			name: "A simple quote with a comment",
			body: `
> This is just a quote

> A personal comment about this quote
`,
			content: "> This is just a quote",
			comment: "A personal comment about this quote",
		},
		{
			name: "A note with a comment",
			body: `
This is just a note

> A personal comment about this note
`,
			content: "This is just a note",
			comment: "A personal comment about this note",
		},
		{
			name: "A multiline comment",
			body: `
This is just a note

> A personal comment
> about this note
`,
			content: "This is just a note",
			comment: "A personal comment\nabout this note",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, comment := markdown.StripComment(tt.body)
			assert.Equal(t, tt.content, content)
			assert.Equal(t, tt.comment, comment)
		})
	}
}

func TestCleanCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		md       string // input
		expected string // output
	}{
		{
			name: "No code blocks",
			md: "# Hello\n\nWorld\n",
			expected: "# Hello\n\nWorld\n",
		},
		{
			name: "Syntax with backticks",
			md: "# Hello\n\nWorld\n\n```md\n# Hello\nWorld\n```\n",
			expected: "# Hello\n\nWorld\n\n\n\n\n\n",
		},
		{
			name: "Syntax with spaces",
			md: "# Hello\n\nWorld\n\n    # Hello\n    World\n",
			expected: "# Hello\n\nWorld\n\n\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := markdown.CleanCodeBlocks(tt.md)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExtractQuote(t *testing.T) {
	tests := []struct {
		name        string
		body        string // input
		quote       string // output
		attribution string // output
	}{
		{
			name:        "No attribution",
			body:        "> A basic quote",
			quote:       "A basic quote\n",
			attribution: "",
		},
		{
			name:        "Multiline quote",
			body:        "> A basic quote\n> on two lines.",
			quote:       "A basic quote\non two lines.\n",
			attribution: "",
		},
		{
			name:        "With attribution using en-dash",
			body:        "> A basic quote\n> — Me",
			quote:       "A basic quote\n",
			attribution: "Me",
		},
		{
			name:        "With attribution using --",
			body:        "> A basic quote\n> -- Me",
			quote:       "A basic quote\n",
			attribution: "Me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote, attribution := markdown.ExtractQuote(tt.body)
			assert.Equal(t, tt.quote, quote)
			assert.Equal(t, tt.attribution, attribution)
		})
	}

}

func TestStripTopHeading(t *testing.T) {
	tests := []struct {
		name     string
		md       string // input
		expected string // output
	}{
		{
			name:     "Basic",
			md:       "# Heading\n\nText\n",
			expected: "Text\n",
		},
		{
			name:     "No heading",
			md:       "Text1\n\nText2",
			expected: "Text1\n\nText2",
		},
		{
			name:     "Leading Blank Lines",
			md:       "\n\n#Heading\n\nText",
			expected: "Text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := markdown.StripTopHeading(tt.md)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
