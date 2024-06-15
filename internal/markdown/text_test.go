package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestSlug(t *testing.T) {
	var tests = []struct {
		name     string // name
		inputs   []any  // input
		expected string // output
	}{
		{
			"already acceptable value",
			[]any{markdown.Document("this-is-a-slug")},
			"this-is-a-slug",
		},
		{
			"multiple values",
			[]any{"go", "Flashcard", markdown.Document("What Are Goroutines?")},
			"go-flashcard-what-are-goroutines",
		},
		{
			"Markdown values",
			[]any{"**go**", "Flashcard", markdown.Document("What Are __Goroutines__?")},
			"go-flashcard-what-are-goroutines",
		},
		{
			"Uppercases & Lowercases",
			[]any{"Hello World"},
			"hello-world",
		},
		{
			"Accents",
			[]any{"àéôiîè"},
			"aeoiie",
		},
		{
			"Empty values",
			[]any{"hello", "", "  ", "\t", "world"},
			"hello-world",
		},
		{
			"Punctation",
			[]any{markdown.Document("Doing = Creating & Improving")},
			"doing-creating-and-improving",
		},
		{
			"Quote",
			[]any{`Answering "Why?"`},
			"answering-why",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := markdown.Slug(tt.inputs...)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
