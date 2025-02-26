package text_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
)

func TestUnescapeTestContent(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Replace special backtick character ”",
			input:    "”@slug: toto”",
			expected: "`@slug: toto`",
		},
		{
			name:     "Replace special backtick character ‛",
			input:    "‛@slug: toto‛",
			expected: "`@slug: toto`",
		},
		{
			name:     "Replace both special backtick characters",
			input:    "”@slug: toto‛",
			expected: "`@slug: toto`",
		},
		{
			name:     "No special characters",
			input:    "@slug: toto",
			expected: "@slug: toto",
		},
		{
			name:     "Mixed content",
			input:    "Hello ”@slug: toto‛ World",
			expected: "Hello `@slug: toto` World",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, text.UnescapeTestContent(tt.input))
		})
	}
}
