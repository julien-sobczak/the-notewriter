package text_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
)

func TestSquashBlankLines(t *testing.T) {
	var tests = []struct {
		name     string // name
		input    string // input
		expected string // expected result
	}{
		{
			"TwoLines",
			`
This is a paragrah.


This is a second paragraph.

This is a third paragraph.

`,
			`
This is a paragrah.

This is a second paragraph.

This is a third paragraph.

`,
		},
		{
			"NoEmptyLines",
			`
A
B
C
D
E
`,
			`
A
B
C
D
E
`,
		},
		{
			"SeveralEmptyLines",
			`
A




C
`,
			`
A

C
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.SquashBlankLines(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestPrefixLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string // input
		prefix   string // input
		expected string // output
	}{
		{
			name:     "Basic",
			input:    "Hello\nWorld",
			prefix:   "> ",
			expected: "> Hello\n> World\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.PrefixLines(tt.input, tt.prefix)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name     string
		text     string // input
		n        int    // input
		expected string // output
	}{
		{
			name:     "Empty",
			text:     "",
			n:        10,
			expected: "",
		},
		{
			name:     "Basic",
			text:     "-",
			n:        3,
			expected: "---",
		},
		{
			name:     "String",
			text:     "cou",
			n:        2,
			expected: "coucou",
		},
	}
	for _, tt := range tests {
		actual := text.Repeat(tt.text, tt.n)
		assert.Equal(t, tt.expected, actual)
	}
}

func TestTrimLinePrefix(t *testing.T) {
	tests := []struct {
		name     string
		text     string // input
		prefix   string // input
		expected string // output
	}{
		{
			name:     "Basic",
			text:     "> This\n> is\n> an \n>  example",
			prefix:   "> ",
			expected: "This\nis\nan \n example\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.TrimLinePrefix(tt.text, tt.prefix)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIsBlank(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		blank bool
	}{

		{
			name:  "Empty",
			input: "",
			blank: true,
		},

		{
			name:  "Only spaces",
			input: "   ",
			blank: true,
		},

		{
			name:  "Leading spaces",
			input: " Not blank",
			blank: false,
		},

		{
			name:  "EOL",
			input: "\n",
			blank: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.IsBlank(tt.input)
			assert.Equal(t, actual, tt.blank)
		})
	}
}

func TestTrimExtension(t *testing.T) {
	var tests = []struct {
		name     string // name
		path     string // input
		expected string // output
	}{
		{
			name:     "Basic filename",
			path:     "README.md",
			expected: "README",
		},
		{
			name:     "Basic directory",
			path:     "medias/",
			expected: "medias",
		},
		{
			name:     "File path",
			path:     "medias/pic.png",
			expected: "medias/pic",
		},
		{
			name:     "Several extensions",
			path:     "medias/pic.png.back",
			expected: "medias/pic.png",
		},
		{
			name:     "md file",
			path:     "note.md",
			expected: "note",
		},
		{
			name:     "markdown file",
			path:     "note.markdown",
			expected: "note",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.TrimExtension(tt.path)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExtractLines(t *testing.T) {
	input := `line1
line2
line3
line4`

	firstLine := text.ExtractLines(input, 1, 1)
	assert.Equal(t, "line1", firstLine)

	twoLines := text.ExtractLines(input, 2, 3)
	assert.Equal(t, "line2\nline3", twoLines)

	lastLines := text.ExtractLines(input, 3, 5)
	assert.Equal(t, "line3\nline4", lastLines)
}

func TestLineNumber(t *testing.T) {
	input := `1. Hello
2. Bonjour
3. Ola
`
	assert.Equal(t, 1, text.LineNumber(input, "Hello"))
	assert.Equal(t, 2, text.LineNumber(input, "Bonjour"))
	assert.Equal(t, 3, text.LineNumber(input, "Ola"))
}

func TestStripHTMLComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No comment",
			input:    "A text with\nno comment.",
			expected: "A text with\nno comment.",
		},
		{
			name:     "Single line",
			input:    `A text with an <!-- inline --> comment.`,
			expected: `A text with an  comment.`,
		},
		{
			name:     "Multiple line",
			input:    "A text\nwith an\n<!--\nlong\nlong\n-->\ncomment\n.",
			expected: "A text\nwith an\n\ncomment\n.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.StripHTMLComments(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBookTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "title already ok",
			input:    "Good Inside",
			expected: "Good Inside",
		},

		{
			name:     "title with subtitle already ok",
			input:    "Good Inside: A Practical Guide to Becoming the Parent You Want to Be",
			expected: "Good Inside: A Practical Guide to Becoming the Parent You Want to Be",
		},

		{
			name:     "lowercase",
			input:    "good inside: a practical guide to becoming the parent you want to be",
			expected: "Good Inside: A Practical Guide to Becoming the Parent You Want to Be",
		},

		{
			name:     "short words in uppercase",
			input:    "good inside: A practical guide To becoming The parent you want To be",
			expected: "Good Inside: A Practical Guide to Becoming the Parent You Want to Be",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := text.ToBookTitle(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
