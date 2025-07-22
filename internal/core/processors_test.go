package core_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDateFromTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    markdown.Document // input
		expected string            // expected date
		found    bool              // whether a date was found
	}{
		{
			name:     "Valid full date",
			input:    "Journal: 2023-10-15.",
			expected: "2023-10-15",
			found:    true,
		},
		{
			name:     "Valid year and month",
			input:    "Journal: The event happened in 2023-10",
			expected: "2023-10",
			found:    true,
		},
		{
			name:     "Valid year only",
			input:    "Journal: We Are in 2023",
			expected: "2023",
			found:    true,
		},
		{
			name:     "Multiple dates, pick first",
			input:    "Journal: 2023-10-15 and 2022-05-01.",
			expected: "2023-10-15",
			found:    true,
		},
		{
			name:     "Invalid date format but valid year",
			input:    "Journal: The date is 15-10-2023.",
			expected: "2023",
			found:    true,
		},
		{
			name:     "No date present",
			input:    "Journal: No date",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, found := core.ExtractDateFromTitle(tt.input)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.found, found)
		})
	}
}

func TestQuoteRewriterPreprocessor(t *testing.T) {
	tests := []struct {
		name         string
		inputBody    string
		expectedBody string
	}{
		{
			name:         "Regular text is prefixed with quotes",
			inputBody:    "This is a regular line\nAnother regular line",
			expectedBody: "> This is a regular line\n> Another regular line",
		},
		{
			name:         "Empty lines remain unchanged",
			inputBody:    "First line\n\nThird line",
			expectedBody: "> First line\n\n> Third line",
		},
		{
			name:         "Already quoted lines remain unchanged",
			inputBody:    "> Already quoted\nNot quoted",
			expectedBody: "> Already quoted\n> Not quoted",
		},
		{
			name:         "Tags/attributes lines are not quoted",
			inputBody:    "Regular text\n`#tag1` `#tag2` `@attribute1:value1`",
			expectedBody: "> Regular text\n`#tag1` `#tag2` `@attribute1:value1`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			// IMPROVEMENT: Use builder or factory pattern for creating test data
			note := &core.ParsedNote{
				Body: markdown.Document(tt.inputBody),
			}
			file := &core.ParsedFile{}

			// Call the function
			resultNotes, err := core.QuoteRewriterPreprocessor(file, note)

			// Check results
			assert.NoError(t, err)
			assert.Len(t, resultNotes, 1)
			assert.Equal(t, tt.expectedBody, resultNotes[0].Body.String())
		})
	}
}

func TestFlashcardExtractorPreprocessor(t *testing.T) {

	t.Run("Basic syntax", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Basic"),
			ShortTitle: markdown.Document("Basic"),
			Slug:       "flashcard-basic",
			Body: markdown.Document(`
Front

---

Back
`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// With a flashcard extracted
		require.Len(t, note.Flashcards, 1)
		flashcard := note.Flashcards[0]
		assert.Equal(t, "Basic", flashcard.ShortTitle.String())
		assert.Equal(t, "flashcard-basic", flashcard.Slug)
		assert.Equal(t, "Front", flashcard.Front.String())
		assert.Equal(t, "Back", flashcard.Back.String())
	})

	t.Run("Basic with reversed syntax", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Basic"),
			ShortTitle: markdown.Document("Basic"),
			Slug:       "flashcard-basic",
			NoteTags:   []string{"reversed"},
			Body: markdown.Document(`
Front

---

Back
`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// With two flashcards extracted
		require.Len(t, note.Flashcards, 2)
		flashcard := note.Flashcards[0]
		flashcardReversed := note.Flashcards[1]
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Basic",
			Slug:       "flashcard-basic",
			Front:      "Front",
			Back:       "Back",
		}, flashcard)
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Basic",
			Slug:       "flashcard-basic-reversed",
			Front:      "Back",
			Back:       "Front",
		}, flashcardReversed)
	})

	t.Run("Cloze deletion syntax", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Cloze Deletion"),
			ShortTitle: markdown.Document("Cloze Deletion"),
			Slug:       "flashcard-cloze-deletion",
			Body:       markdown.Document(`Canberra was founded in **[[c1::1913]]**.`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// A flashcard must have been generated from the cloze deletion
		require.Len(t, note.Flashcards, 1)
		flashcard := note.Flashcards[0]
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Cloze Deletion",
			Slug:       "flashcard-cloze-deletion",
			Front:      "Canberra was founded in **[...]**.",
			Back:       "Canberra was founded in **1913**.",
		}, flashcard)
	})

	t.Run("Cloze deletion with hint syntax", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Cloze Deletion"),
			ShortTitle: markdown.Document("Cloze Deletion"),
			Slug:       "flashcard-cloze-deletion",
			Body:       markdown.Document(`Canberra was founded in [[c1::1913::year]].`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// A flashcard must have been generated from the cloze deletion
		require.Len(t, note.Flashcards, 1)
		flashcard := note.Flashcards[0]
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Cloze Deletion",
			Slug:       "flashcard-cloze-deletion",
			Front:      "Canberra was founded in [year].",
			Back:       "Canberra was founded in 1913.",
		}, flashcard)
	})

	t.Run("Multiple cloze deletions in same group", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Cloze Deletion"),
			ShortTitle: markdown.Document("Cloze Deletion"),
			Slug:       "flashcard-cloze-deletion",
			Body:       markdown.Document(`[[c1::Canberra::city]] was founded in [[c1::1913]].`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// A flashcard must have been generated from the cloze deletion
		require.Len(t, note.Flashcards, 1)
		flashcard := note.Flashcards[0]
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Cloze Deletion",
			Slug:       "flashcard-cloze-deletion",
			Front:      "[city] was founded in [...].",
			Back:       "Canberra was founded in 1913.",
		}, flashcard)
	})

	t.Run("Multiple cloze deletions in different groups", func(t *testing.T) {
		file := &core.ParsedFile{}
		note := &core.ParsedNote{
			Title:      markdown.Document("Flashcard: Cloze Deletion"),
			ShortTitle: markdown.Document("Cloze Deletion"),
			Slug:       "flashcard-cloze-deletion",
			Body:       markdown.Document(`[[c1::Canberra::city]] was founded in [[c2::1913]].`),
		}
		notes, err := core.FlashcardExtractorPreprocessor(file, note)
		require.NoError(t, err)

		// We still have the original note
		require.Len(t, notes, 1)
		note = notes[0]

		// Several flashcards must have been generated from the cloze deletions
		require.Len(t, note.Flashcards, 2)
		flashcard1 := note.Flashcards[0]
		flashcard2 := note.Flashcards[1]
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Cloze Deletion",
			Slug:       "flashcard-cloze-deletion",
			Front:      "[city] was founded in 1913.",
			Back:       "Canberra was founded in 1913.",
		}, flashcard1)
		assert.Equal(t, &core.ParsedFlashcard{
			ShortTitle: "Cloze Deletion",
			Slug:       "flashcard-cloze-deletion",
			Front:      "Canberra was founded in [...].",
			Back:       "Canberra was founded in 1913.",
		}, flashcard2)
	})

}
