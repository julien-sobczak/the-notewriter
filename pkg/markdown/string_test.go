package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
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
			actual := markdown.SquashBlankLines(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
