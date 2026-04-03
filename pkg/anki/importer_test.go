package anki

import (
	"fmt"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
)

func TestHtmlToMarkdown(t *testing.T) {

	t.Run("Basic HTML", func(t *testing.T) {
		tests := []struct {
			name          string
			html          string
			expectedText  string
			expectedMedia []string
		}{
			{
				name:          "empty string",
				html:          "",
				expectedText:  "",
				expectedMedia: []string{},
			},
			{
				name:          "plain text",
				html:          "plain text",
				expectedText:  "plain text",
				expectedMedia: []string{},
			},
			{
				name:          "bold with b tag",
				html:          "<b>bold text</b>",
				expectedText:  "**bold text**",
				expectedMedia: []string{},
			},
			{
				name:          "bold with strong tag",
				html:          "<strong>bold text</strong>",
				expectedText:  "**bold text**",
				expectedMedia: []string{},
			},
			{
				name:          "italic with i tag",
				html:          "<i>italic text</i>",
				expectedText:  "_italic text_",
				expectedMedia: []string{},
			},
			{
				name:          "italic with em tag",
				html:          "<em>italic text</em>",
				expectedText:  "_italic text_",
				expectedMedia: []string{},
			},
			{
				name:          "mixed formatting",
				html:          "<b>bold</b> and <i>italic</i>",
				expectedText:  "**bold** and _italic_",
				expectedMedia: []string{},
			},
			{
				name:          "img tag with src",
				html:          `<img src="image.jpg">`,
				expectedText:  `![image.jpg](./image.jpg)`,
				expectedMedia: []string{"image.jpg"},
			},
			{
				name:          "img tag with quotes",
				html:          `<img src='image.png'>`,
				expectedText:  `![image.png](./image.png)`,
				expectedMedia: []string{"image.png"},
			},
			{
				name:          "sound tag",
				html:          `[sound:audio.mp3]`,
				expectedText:  `![audio.mp3](./audio.mp3)`,
				expectedMedia: []string{"audio.mp3"},
			},
			{
				name:          "multiple sounds",
				html:          `[sound:file1.mp3] and [sound:file2.mp3]`,
				expectedText:  `![file1.mp3](./file1.mp3) and ![file2.mp3](./file2.mp3)`,
				expectedMedia: []string{"file1.mp3", "file2.mp3"},
			},
			{
				name:          "combined formatting and media",
				html:          `<b>Bold</b> with <img src="pic.jpg"> and [sound:note.mp3]`,
				expectedText:  `**Bold** with ![pic.jpg](./pic.jpg) and ![note.mp3](./note.mp3)`,
				expectedMedia: []string{"pic.jpg", "note.mp3"},
			},
			{
				name:          "nested tags",
				html:          `<b><i>bold italic</i></b>`,
				expectedText:  `**_bold italic_**`,
				expectedMedia: []string{},
			},
			{
				name:          "img with extra attributes",
				html:          `<img class="image" src="test.gif" alt="test">`,
				expectedText:  `![test.gif](./test.gif)`,
				expectedMedia: []string{"test.gif"},
			},
		}

		importer := &Importer{MediaDir: ""}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				text, media := importer.htmlToMarkdown(tt.html)
				assert.Equal(t, tt.expectedText, text)
				assert.Equal(t, tt.expectedMedia, media)
			})
		}
	})

	t.Run("With Media Dir", func(t *testing.T) {
		importer := &Importer{MediaDir: "assets"}
		text, media := importer.htmlToMarkdown(`<img src="photo.jpg">`)
		expectedText := `![photo.jpg](./assets/photo.jpg)`
		expectedMedia := []string{"photo.jpg"}

		assert.Equal(t, expectedText, text)
		assert.Equal(t, expectedMedia, media)
	})

	t.Run("Real-World HTML", func(t *testing.T) {
		tests := []struct {
			name     string
			html     string
			markdown string
		}{
			{
				name: "Markdown Emphasis",
				html: "(System Design) What is the <b>latency</b>? What is the <b>throughput</b>?<div><br></div><div><i>An assembly line is manufacturing cars. It takes eight hours to manufacture a car and that the factory produces one hundred and twenty cars per day.</i></div>",
				markdown: `(System Design) What is the **latency**? What is the **throughput**?

_An assembly line is manufacturing cars. It takes eight hours to manufacture a car and that the factory produces one hundred and twenty cars per day._`,
			},

			{
				name: "Markdown Codeblocks",
				html: `(Go)&nbsp;<b>Compile</b>?<pre><code>func (p Point) Distance(q Point) float64 { ... }

p := Point{1, 2}
q := Point{4, 6}
fmt.Println(Distance(p, q))
fmt.Println(p.Distance(q))</code></pre>`,
				markdown: text.UnescapeTestContent(`(Go) **Compile**?
‛‛‛
func (p Point) Distance(q Point) float64 { ... }

p := Point{1, 2}
q := Point{4, 6}
fmt.Println(Distance(p, q))
fmt.Println(p.Distance(q))
‛‛‛`),
			},
		}

		importer := &Importer{MediaDir: ""}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				text, _ := importer.htmlToMarkdown(tt.html)
				assert.Equal(t, tt.markdown, text)
			})
		}
	})
}

func TestAnkiEaseToConfidence(t *testing.T) {
	tests := []struct {
		name     string
		ease     int
		expected int
	}{
		{
			name:     "Again",
			ease:     1,
			expected: 0,
		},
		{
			name:     "Hard",
			ease:     2,
			expected: 30,
		},
		{
			name:     "Good",
			ease:     3,
			expected: 60,
		},
		{
			name:     "Easy",
			ease:     4,
			expected: 90,
		},
		{
			name:     "Unknown ease value",
			ease:     5,
			expected: 0,
		},
		{
			name:     "Negative ease value",
			ease:     -1,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnkiEaseToConfidence(tt.ease)
			if got != tt.expected {
				t.Errorf("AnkiEaseToConfidence(%d) = %d, want %d", tt.ease, got, tt.expected)
			}
		})
	}
}

// TestAnkiDueToDueAt tests the AnkiDueToDueAt function with various due values
func TestAnkiDueToDueAt(t *testing.T) {
	baseTime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		ankiDue           int
		collectionTime    time.Time
		expectedTimestamp int64
	}{
		{
			name:              "Unix timestamp (large due value)",
			ankiDue:           1704067200, // 2024-01-01 00:00:00 UTC
			collectionTime:    baseTime,
			expectedTimestamp: 1704067200,
		},
		{
			name:              "Days-based due (0 days from collection)",
			ankiDue:           0,
			collectionTime:    baseTime,
			expectedTimestamp: baseTime.Unix(),
		},
		{
			name:              "Days-based due (1 day from collection)",
			ankiDue:           1,
			collectionTime:    baseTime,
			expectedTimestamp: baseTime.Add(24 * time.Hour).Unix(),
		},
		{
			name:              "Days-based due (7 days from collection)",
			ankiDue:           7,
			collectionTime:    baseTime,
			expectedTimestamp: baseTime.Add(7 * 24 * time.Hour).Unix(),
		},
		{
			name:              "Days-based due (negative days)",
			ankiDue:           -1,
			collectionTime:    baseTime,
			expectedTimestamp: baseTime.Add(-24 * time.Hour).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnkiDueToDueAt(tt.ankiDue, tt.collectionTime)
			if got.Unix() != tt.expectedTimestamp {
				t.Errorf("AnkiDueToDueAt(%d, %v) = %d, want %d", tt.ankiDue, tt.collectionTime, got.Unix(), tt.expectedTimestamp)
			}
		})
	}
}

func TestAnkiReviewsToOperations(t *testing.T) {
	oid.UseFixed(t, oid.New()) // Use a fixed OID to ignore operation OID in test assertions

	// Set up a base time and a card for testing
	collectionTime := testutil.HumanTime(t, "2024-01-01 00:00")
	cardID := int64(123)
	cardDue := 24 // 24 days after collection time

	tests := []struct {
		name       string
		reviews    []*Review
		operations []*core.Operation
	}{
		{
			name:       "No reviews",
			reviews:    []*Review{},
			operations: nil,
		},
		{
			name: "Single review, learning queue",
			reviews: []*Review{
				{
					ID:      testutil.HumanTime(t, "2024-01-01 10:15").Unix(),
					CID:     cardID,
					Ease:    3,
					Time:    1500,
					Ivl:     0,
					LastIvl: 0,
					Factor:  0,
					Type:    0, // learning
				},
			},
			operations: []*core.Operation{
				// The review must have been preserved
				core.NewOperationReviewFlashcardAt(
					oid.NewFromString("anki-"+fmt.Sprint(cardID)),
					core.FlashcardReview{
						Algorithm:  "anki-sm-2",
						Confidence: AnkiEaseToConfidence(3),
						Duration:   1500 * time.Millisecond,
						DueAt:      AnkiDueToDueAt(cardDue, collectionTime),
						Settings: map[string]any{
							"cid":     cardID,
							"ease":    3,
							"time":    1500,
							"ivl":     0,
							"lastIvl": 0,
							"factor":  0,
							"type":    0,
						},
					},
					testutil.HumanTime(t, "2024-01-01 10:15:00").UTC(),
				),
				// A new review must have been created to continue using TheNoteWriter SRS algorithm
				core.NewOperationReviewFlashcardAt(
					oid.NewFromString("anki-"+fmt.Sprint(cardID)),
					core.FlashcardReview{
						Algorithm:  "nt-boring",
						Confidence: AnkiEaseToConfidence(3),
						Duration:   1500 * time.Millisecond,
						DueAt:      AnkiDueToDueAt(cardDue, collectionTime),
						Settings: map[string]any{
							"queue":       "learning",
							"step":        0,
							"interval":    "1m",
							"easeFactor":  2.5,
							"repetitions": 1,
						},
					},
					testutil.HumanTime(t, "2024-01-01 10:15:01").UTC(), // One second later
				),
			},
		},
		{
			name: "Multiple reviews, reviewing queue",
			reviews: []*Review{
				{
					ID:      testutil.HumanTime(t, "2024-01-01 10:15").Unix(),
					CID:     cardID,
					Ease:    3,
					Time:    1000,
					Ivl:     1,
					LastIvl: 0,
					Factor:  2000,
					Type:    1, // review
				},
				{
					ID:      testutil.HumanTime(t, "2024-01-02 10:15").Unix(),
					CID:     cardID,
					Ease:    1,
					Time:    2000,
					Ivl:     5,
					LastIvl: 1,
					Factor:  2800,
					Type:    1, // review
				},
			},
			operations: []*core.Operation{
				// The reviews must have been preserved
				core.NewOperationReviewFlashcardAt(
					oid.NewFromString("anki-"+fmt.Sprint(cardID)),
					core.FlashcardReview{
						Algorithm:  "anki-sm-2",
						Confidence: AnkiEaseToConfidence(3),
						Duration:   1000 * time.Millisecond,
						DueAt:      AnkiDueToDueAt(cardDue, collectionTime),
						Settings: map[string]any{
							"cid":     cardID,
							"ease":    3,
							"time":    1000,
							"ivl":     1,
							"lastIvl": 0,
							"factor":  2000,
							"type":    1,
						},
					},
					testutil.HumanTime(t, "2024-01-01 10:15:00").UTC(),
				),
				core.NewOperationReviewFlashcardAt(
					oid.NewFromString("anki-"+fmt.Sprint(cardID)),
					core.FlashcardReview{
						Algorithm:  "anki-sm-2",
						Confidence: AnkiEaseToConfidence(1),
						Duration:   2000 * time.Millisecond,
						DueAt:      AnkiDueToDueAt(cardDue, collectionTime),
						Settings: map[string]any{
							"cid":     cardID,
							"ease":    1,
							"time":    2000,
							"ivl":     5,
							"lastIvl": 1,
							"factor":  2800,
							"type":    1,
						},
					},
					testutil.HumanTime(t, "2024-01-02 10:15:00").UTC(),
				),
				// A new review must have been created to continue using TheNoteWriter SRS algorithm
				core.NewOperationReviewFlashcardAt(
					oid.NewFromString("anki-"+fmt.Sprint(cardID)),
					core.FlashcardReview{
						Algorithm:  "nt-boring",
						Confidence: AnkiEaseToConfidence(1),
						Duration:   2000 * time.Millisecond,
						DueAt:      AnkiDueToDueAt(cardDue, collectionTime),
						Settings: map[string]any{
							"queue":       "reviewing",
							"interval":    "5d",
							"easeFactor":  2.8,
							"repetitions": 2,
						},
					},
					testutil.HumanTime(t, "2024-01-02 10:15:01").UTC(), // One second later
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &Card{
				ID:  cardID,
				Due: cardDue,
			}
			actual := AnkiReviewsToOperations(collectionTime, card, tt.reviews)
			assert.Len(t, actual, len(tt.operations), "Expected %d operations, got %d", len(tt.operations), len(actual))
			for i, op := range actual {
				actualOperation := tt.operations[i]
				assert.Equal(t, actualOperation, op, "Operation %d does not match expected", i)
			}
		})
	}
}
