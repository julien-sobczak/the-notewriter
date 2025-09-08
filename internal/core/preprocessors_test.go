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
		name            string
		inputBody       string
		inputAttributes map[string]any
		expectedBody    string
	}{
		{
			name:         "Regular text is prefixed with quotes",
			inputBody:    "This is a regular line\nAnother regular line",
			expectedBody: "> This is a regular line\n> Another regular line",
		},
		{
			name: "Attribution is appended if attributes are present",
			inputAttributes: map[string]any{
				"name":        "Jean Doe",
				"occupation":  "writer",
				"nationality": "French",
			},
			inputBody:    "This is a regular line\nAnother regular line",
			expectedBody: "> This is a regular line\n> Another regular line\n> ― Jean Doe, French writer",
		},
		{
			name:         "Empty lines are included if present in the middle of a quote",
			inputBody:    "First line\n>\nThird line",
			expectedBody: "> First line\n>\n> Third line",
		},
		{
			name:         "Already quoted lines remain unchanged",
			inputBody:    "> Already quoted\nNot quoted",
			expectedBody: "> Already quoted\n> Not quoted",
		},
		{
			name:         "Tags/attributes lines are not quoted",
			inputBody:    "`#tag1` `#tag2`\nRegular text\n`@attribute1:value1`",
			expectedBody: "`#tag1` `#tag2`\n> Regular text\n`@attribute1:value1`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			// IMPROVEMENT: Use builder or factory pattern for creating test data
			note := &core.ParsedNote{
				Attributes: core.AttributeSet(tt.inputAttributes),
				Body:       markdown.Document(tt.inputBody),
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
			Body:       markdown.Document(`Canberra was founded in **[{c1::1913}]**.`),
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
			Body:       markdown.Document(`Canberra was founded in [{c1::1913::year}].`),
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
			Body:       markdown.Document(`[{c1::Canberra::city}] was founded in [{c1::1913}].`),
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
			Body:       markdown.Document(`[{c1::Canberra::city}] was founded in [{c2::1913}].`),
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

func TestListItemsPreprocessor(t *testing.T) {
	t.Run("Empty content", func(t *testing.T) {
		file := &core.ParsedFile{
			RelativePath: "test.md",
		}
		note := &core.ParsedNote{
			Body: "",
			Line: 1,
		}

		result, err := core.ListItemsPreprocessor(file, note)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, core.ListItems{Children: []*core.ListItem{}}, result[0].Items)
	})

	t.Run("Simple list items", func(t *testing.T) {
		file := &core.ParsedFile{
			RelativePath: "test.md",
		}
		note := &core.ParsedNote{
			Body: "* First item\n* Second item with emoji 🎉\n* Third item with `#tag`",
			Line: 5,
		}

		result, err := core.ListItemsPreprocessor(file, note)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[0].Items.Children, 3)

		// First item
		item1 := result[0].Items.Children[0]
		assert.Equal(t, 5, item1.Line)
		assert.Equal(t, "First item", item1.Text)
		assert.Empty(t, item1.Tags)
		assert.Empty(t, item1.Attributes)
		assert.Empty(t, item1.Emojis)
		assert.Empty(t, item1.Children)

		// Second item with emoji
		item2 := result[0].Items.Children[1]
		assert.Equal(t, 6, item2.Line)
		assert.Equal(t, "Second item with emoji 🎉", item2.Text)
		assert.Empty(t, item2.Tags)
		assert.Empty(t, item2.Attributes)
		assert.Contains(t, item2.Emojis, "🎉")

		// Third item with tag
		item3 := result[0].Items.Children[2]
		assert.Equal(t, 7, item3.Line)
		assert.Equal(t, "Third item with", item3.Text)
		assert.Contains(t, item3.Tags, "tag")
		assert.Empty(t, item3.Children)
	})

	t.Run("Nested list items", func(t *testing.T) {
		file := &core.ParsedFile{
			RelativePath: "books.md",
		}
		note := &core.ParsedNote{
			Body: "* _L'Enfant, la Taupe, le Renard et le Cheval_, by Charlie Mackesy 🇬🇧 🇫🇷 [👀](https://example.com) ★★★★★ `#life` `#philosophy`\n  * A masterpiece of children literature (see the sequel _Always Remember_ ★★★, 2025, 🇬🇧 🤞 ).\n* _Grand Panda et Petit Dragon_, by James Norbury 🇬🇧 🇫🇷 [👀](https://example.com) ★★★★★ `#mindfulness` `#life`\n  * A book filled with wisdom. For small kids and big adults.",
			Line: 1,
		}

		result, err := core.ListItemsPreprocessor(file, note)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[0].Items.Children, 2)

		// First top-level item
		item1 := result[0].Items.Children[0]
		assert.Equal(t, 1, item1.Line)
		assert.Contains(t, item1.Text, "_L'Enfant, la Taupe, le Renard et le Cheval_, by Charlie Mackesy")
		assert.Contains(t, item1.Tags, "life")
		assert.Contains(t, item1.Tags, "philosophy")
		assert.Equal(t, 10, item1.Attributes["rating"]) // 5 stars = 10 points
		assert.Contains(t, item1.Emojis, "🇬🇧")
		assert.Contains(t, item1.Emojis, "🇫🇷")
		require.Len(t, item1.Children, 1)

		// Child of first item
		child1 := item1.Children[0]
		assert.Equal(t, 2, child1.Line)
		assert.Contains(t, child1.Text, "A masterpiece of children literature")
		assert.Contains(t, child1.Emojis, "🇬🇧")
		assert.Contains(t, child1.Emojis, "🤞")

		// Second top-level item
		item2 := result[0].Items.Children[1]
		assert.Equal(t, 3, item2.Line)
		assert.Contains(t, item2.Text, "_Grand Panda et Petit Dragon_, by James Norbury")
		assert.Contains(t, item2.Tags, "mindfulness")
		assert.Contains(t, item2.Tags, "life")
		assert.Equal(t, 10, item2.Attributes["rating"]) // 5 stars = 10 points
		require.Len(t, item2.Children, 1)

		// Child of second item
		child2 := item2.Children[0]
		assert.Equal(t, 4, child2.Line)
		assert.Contains(t, child2.Text, "A book filled with wisdom")
		assert.Empty(t, child2.Tags)
		assert.Empty(t, child2.Children)
	})

	t.Run("Mixed list markers", func(t *testing.T) {
		file := &core.ParsedFile{
			RelativePath: "test.md",
		}
		note := &core.ParsedNote{
			Body: "* Bullet item\n- Dash item\n+ Plus item\n1. Numbered item\n2. Second numbered item",
			Line: 10,
		}

		result, err := core.ListItemsPreprocessor(file, note)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[0].Items.Children, 5)

		// Check all items are parsed correctly
		expectedTexts := []string{"Bullet item", "Dash item", "Plus item", "Numbered item", "Second numbered item"}
		for i, expected := range expectedTexts {
			assert.Equal(t, expected, result[0].Items.Children[i].Text)
			assert.Equal(t, 10+i, result[0].Items.Children[i].Line)
		}
	})
}
