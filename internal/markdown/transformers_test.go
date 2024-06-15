package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceCharacters(t *testing.T) {

	t.Run("AsciidocCharacterSubstitutions", func(t *testing.T) {
		tests := []struct {
			name     string
			md       markdown.Document // input
			expected markdown.Document // output
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
				actual, err := tt.md.Transform(markdown.ReplaceCharacters(markdown.AsciidocCharacterSubstitutions))
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}

func TestStripHTMLComments(t *testing.T) {
	var tests = []struct {
		name     string // name
		input    markdown.Document
		expected markdown.Document
	}{
		{
			name:     "No comment",
			input:    "This document doesn't contain an HTML comment",
			expected: "This document doesn't contain an HTML comment",
		},
		{
			name: "Single line",
			input: `
This document contains an HTML comment

<!-- here -->
`,
			expected: "This document contains an HTML comment",
		},
		{
			name: "Multi-lines",
			input: `
This document contains an HTML comment.
<!--
 A comment
 -->
A final text.
`,
			expected: "This document contains an HTML comment.\n\nA final text.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.input.Transform(markdown.StripHTMLComments())
			require.NoError(t, err)
			assert.Equal(t, tt.expected.TrimSpace(), actual.TrimSpace())
		})
	}
}

func StripMarkdownUnofficialComments(t *testing.T) {
	var tests = []struct {
		name     string // name
		input    markdown.Document
		expected markdown.Document
	}{
		{
			name:     "No comment",
			input:    "This document doesn't contain an HTML comment",
			expected: "This document doesn't contain an HTML comment",
		},
		{
			name: "HTML comment", // not supported by this transformer
			input: `
This document contains an HTML comment <!-- here -->
`,
			expected: "This document contains an HTML comment <!-- here -->",
		},
		{
			name: "Single line",
			input: `
This document contains a Markdown comment

<!--- here --->
`,
			expected: "This document contains a Markdown comment",
		},
		{
			name: "Multi-lines",
			input: `
This document contains a Markdown comment.
<!---
 A comment
 --->
A final text.
`,
			expected: "This document contains a Markdown comment.\n\nA final text.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.input.Transform(markdown.StripMarkdownUnofficialComments())
			require.NoError(t, err)
			assert.Equal(t, tt.expected.TrimSpace(), actual.TrimSpace())
		})
	}
}

func TestAlignHeadings(t *testing.T) {
	var tests = []struct {
		name     string // name
		input    markdown.Document
		expected markdown.Document
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
			actual, err := tt.input.Transform(markdown.AlignHeadings())
			require.NoError(t, err)
			assert.Equal(t, tt.expected.TrimSpace(), actual.TrimSpace())
		})
	}
}

func TestStripCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		md       markdown.Document // input
		expected markdown.Document // output
	}{
		{
			name:     "No code blocks",
			md:       "# Hello\n\nWorld\n",
			expected: "# Hello\n\nWorld\n",
		},
		{
			name:     "Syntax with backticks",
			md:       "# Hello\n\nWorld\n\n```md\n# Hello\nWorld\n```\n",
			expected: "# Hello\n\nWorld\n\n\n\n\n\n",
		},
		{
			name:     "Syntax with spaces",
			md:       "# Hello\n\nWorld\n\n    # Hello\n    World\n",
			expected: "# Hello\n\nWorld\n\n\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.md.Transform(markdown.StripCodeBlocks())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestStripTopHeading(t *testing.T) {
	tests := []struct {
		name     string
		md       markdown.Document // input
		expected markdown.Document // output
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
			actual, err := tt.md.Transform(markdown.StripTopHeading())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSquashBlankLines(t *testing.T) {
	var tests = []struct {
		name     string
		input    markdown.Document
		expected markdown.Document
	}{
		{
			name:     "No blank lines",
			input:    "A\nB\nC\nD",
			expected: "A\nB\nC\nD",
		},

		{
			name:     "With blank lines", // but not enough to squash
			input:    "A\n\nB\n\nC\n\nD",
			expected: "A\n\nB\n\nC\n\nD",
		},

		{
			name:     "With many blank lines",
			input:    "A\n\n\nB\n\n\nC\n\n\nD",
			expected: "A\n\nB\n\nC\n\nD",
		},

		{
			name:     "With many many blank lines",
			input:    "A\n\n\nB\n\n\n\nC\n\n\n\n\nD",
			expected: "A\n\nB\n\nC\n\nD",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.input.Transform(markdown.SquashBlankLines())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestStripEmphasis(t *testing.T) {
	var tests = []struct {
		name     string
		input    markdown.Document
		expected markdown.Document
	}{
		{
			"raw text",
			"no special Markdown character",
			"no special Markdown character",
		},
		// Bold
		{
			"bold with asterisks",
			"I just love **bold text**.",
			"I just love bold text.",
		},
		{
			"bold with underscores",
			"I just love __bold text__.",
			"I just love bold text.",
		},
		{
			"bold with double asterisks",
			"Love**is**bold",
			"Loveisbold",
		},
		// Italic
		{
			"italic with asterisks",
			"Italicized text is the *cat's meow*.",
			"Italicized text is the cat's meow.",
		},
		{
			"italic with underscores",
			"Italicized text is the _cat's meow_.",
			"Italicized text is the cat's meow.",
		},
		{
			"italic with double asterisks",
			"A*cat*meow",
			"Acatmeow",
		},

		// Bold+Italic
		{
			"bold+italic 1",
			"This text is ***really important***.",
			"This text is really important.",
		},
		{
			"bold+italic 2",
			"This text is ___really important___.",
			"This text is really important.",
		},
		{
			"bold+italic 3",
			"This text is __*really important*__.",
			"This text is really important.",
		},
		{
			"bold+italic 4",
			"This text is **_really important_**.",
			"This text is really important.",
		},
		{
			"bold+italic 5",
			"This is really***very***important text.",
			"This is reallyveryimportant text.",
		},
		{
			"code",
			"This is some `code`.",
			"This is some code.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.input.Transform(markdown.StripEmphasis())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
