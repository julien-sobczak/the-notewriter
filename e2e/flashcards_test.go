package e2e_test

import (
	"testing"
	"time"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashcards(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("learning.md", `
# Learning

## Flashcard: Rote Memorization

[[c1::Rote memorization]] is a technique for learning information through repetition and recall.

## Flashcard: Spaced Repetition

[[c1::Spaced repetition]] is a technique that involves reviewing information at increasing intervals to enhance retention.`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit()
	require.NoError(t, err)

	// Check the flashcard
	flashcardRote, err := CurrentRepository().FindFlashcardByShortTitle("Rote Memorization")
	require.NoError(t, err)
	require.NotNil(t, flashcardRote)
	flashcardSRS, err := CurrentRepository().FindFlashcardByShortTitle("Spaced Repetition")
	require.NoError(t, err)
	require.NotNil(t, flashcardSRS)
	// None has been reviewed yet
	require.Zero(t, flashcardRote.DueAt)
	require.Zero(t, flashcardSRS.DueAt)

	// Review a flashcard
	dueAt := clock.Now().Add(24 * time.Hour)
	packFile, err := NewPackFileFromOperations([]*Operation{
		NewOperationReviewFlashcard(flashcardRote.OID, FlashcardReview{
			Feedback: FeedbackGood,
			Duration: 50 * time.Millisecond,
			DueAt:    dueAt,
			Settings: map[string]any{
				"interval":    1,
				"ease_factor": 2500,
			},
		}),
	})
	require.NoError(t, err)
	assert.FileExists(t, packFile.ObjectPath())
	CurrentDB().UpsertPackFiles(packFile)

	// Reread the flashcard
	flashcardRote, err = CurrentRepository().FindFlashcardByShortTitle("Rote Memorization")
	require.NoError(t, err)
	require.NotNil(t, flashcardRote)
	assert.Equal(t, clock.Now().UTC().Add(24*time.Hour), flashcardRote.DueAt.UTC())
	// The second flashcard is still not reviewed
	flashcardSRS, err = CurrentRepository().FindFlashcardByShortTitle("Spaced Repetition")
	require.NoError(t, err)
	require.NotNil(t, flashcardSRS)
	assert.Zero(t, flashcardSRS.DueAt)

	// Edit the flashcard text
	tr.FastForward(1 * time.Hour) // Force a new timestamp
	tr.WriteFile("learning.md", `
# Learning

## Flashcard: Rote Memorization

[[c1::Rote memorization]] is a **learning technique** using repetition and recall.

## Flashcard: Spaced Repetition

[[c1::Spaced repetition]] is a **learning technique** using increasing intervals to enhance retention.
		`)
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit()
	require.NoError(t, err)

	// Check the flashcard text has been updated and prior reviews still preserved
	flashcardRote, err = CurrentRepository().FindFlashcardByShortTitle("Rote Memorization")
	require.NoError(t, err)
	require.NotNil(t, flashcardRote)
	assert.Equal(t, "[...] is a **learning technique** using repetition and recall.", flashcardRote.Front.String())
	assert.Equal(t, dueAt.UTC(), flashcardRote.DueAt.UTC())

	// Try to import a packfile containing an old study to ignore
	// Review a flashcard
	oldDueAt := clock.Now().Add(-24 * time.Hour)
	oldOperation := NewOperationReviewFlashcard(flashcardRote.OID, FlashcardReview{
		Feedback: FeedbackEasy,
		Duration: 150 * time.Millisecond,
		DueAt:    oldDueAt,
		Settings: map[string]any{
			"interval":    2,
			"ease_factor": 2200,
		},
	})
	oldOperation.Timestamp = clock.Now().Add(-48 * time.Hour) // Old review
	packFile, err = NewPackFileFromOperations([]*Operation{oldOperation})
	require.NoError(t, err)
	CurrentDB().UpsertPackFiles(packFile)

	// Reread the flashcard
	flashcardRote, err = CurrentRepository().FindFlashcardByShortTitle("Rote Memorization")
	require.NoError(t, err)
	require.NotNil(t, flashcardRote)
	assert.Equal(t, "[...] is a **learning technique** using repetition and recall.", flashcardRote.Front.String())
	assert.Equal(t, dueAt.UTC(), flashcardRote.DueAt.UTC()) // Not updated by the old review
	assert.Equal(t, map[string]any{
		"interval":    1,
		"ease_factor": 2500,
	}, flashcardRote.Settings)
}
