package text_test

import (
	"testing"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
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
			name: "Basic filename",
			path: "README.md",
			expected: "README",
		},
		{
			name: "Basic directory",
			path: "medias/",
			expected: "medias",
		},
		{
			name: "File path",
			path: "medias/pic.png",
			expected: "medias/pic",
		},
		{
			name: "Several extensions",
			path: "medias/pic.png.back",
			expected: "medias/pic.png",
		},
		{
			name: "md file",
			path: "note.md",
			expected: "note",
		},
		{
			name: "markdown file",
			path: "note.markdown",
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