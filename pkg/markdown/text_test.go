package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/stretchr/testify/assert"
)

func TestStripEmphasis(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
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
			actual := markdown.StripEmphasis(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSlug(t *testing.T) {
	var tests = []struct {
		name     string   // name
		inputs   []string // input
		expected string   // output
	}{
		{
			"already acceptable value",
			[]string{"this-is-a-slug"},
			"this-is-a-slug",
		},
		{
			"multiple values",
			[]string{"go", "Flashcard", "What Are Goroutines?"},
			"go-flashcard-what-are-goroutines",
		},
		{
			"Markdown values",
			[]string{"**go**", "Flashcard", "What Are __Goroutines__?"},
			"go-flashcard-what-are-goroutines",
		},
		{
			"Uppercases & Lowercases",
			[]string{"Hello World"},
			"hello-world",
		},
		{
			"Accents",
			[]string{"àéôiîè"},
			"aeoiie",
		},
		{
			"Empty values",
			[]string{"hello", "", "  ", "\t", "world"},
			"hello-world",
		},
		{
			"Punctation",
			[]string{"Doing = Creating & Improving"},
			"doing-creating-and-improving",
		},
		{
			"Quote",
			[]string{`Answering "Why?"`},
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
