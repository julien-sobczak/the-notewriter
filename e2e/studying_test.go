package e2e_test

import (
	"testing"
	"time"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditingFlashcards(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("learning.md", text.UnescapeTestContent(`
# Learning

## Flashcard: Rote Memorization

‛@slug: rote-memorization‛

[{c1::Rote memorization}] is a technique for learning information through repetition and recall.

## Flashcard: Spaced Repetition

‛@slug: spaced-repetition‛

[{c1::Spaced repetition}] is a technique that involves reviewing information at increasing intervals to enhance retention.`)))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
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
			Algorithm:  DefaultSRSAlgorithm,
			Confidence: 60, // Good = moderate-high confidence
			Duration:   50 * time.Millisecond,
			DueAt:      dueAt,
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
	tr.WriteFile("learning.md", text.UnescapeTestContent(`
# Learning

## Flashcard: Rote Memorization

‛@slug: rote-memorization‛

[{c1::Rote memorization}] is a **learning technique** using repetition and recall.

## Flashcard: Spaced Repetition

‛@slug: spaced-repetition‛

[{c1::Spaced repetition}] is a **learning technique** using increasing intervals to enhance retention.
		`))
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
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
		Algorithm:  DefaultSRSAlgorithm,
		Confidence: 80, // Easy = high confidence
		Duration:   150 * time.Millisecond,
		DueAt:      oldDueAt,
		Settings: map[string]any{
			"interval":    2,
			"ease_factor": 2200,
		},
	})
	oldOperation.Timestamp = clock.Now().Add(-48 * time.Hour) // Old review
	packFile, err = NewPackFileFromOperations([]*Operation{oldOperation})
	require.NoError(t, err)
	err = CurrentDB().UpsertPackFiles(packFile)
	require.NoError(t, err)

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

	// Try to delete the flashcards and replay the operations (must fail silently)
	tr.DeleteFile("learning.md")
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	// The flashcards must no longer exist
	flashcardRote, err = CurrentRepository().FindFlashcardByShortTitle("Rote Memorization")
	require.NoError(t, err)
	require.Nil(t, flashcardRote)
	flashcardSRS, err = CurrentRepository().FindFlashcardByShortTitle("Spaced Repetition")
	require.NoError(t, err)
	require.Nil(t, flashcardSRS)

	err = CurrentDB().UpsertPackFiles(packFile)
	require.NoError(t, err)
}

func TestAutomatingFlashcards(t *testing.T) {
	tr := NewTestRepository(t,
		WithFile("english.md", `
## Generator: Expressions

‛@interpreter: python3‛

‛‛‛python
import re

expressions = {
    "processus en place" : "process in place",
    "flux de travail" : "workflow",
    "retour sur investissement" : "return on investment",
}

def generate_slug(text):
    return re.sub(r'[^a-z0-9]+', '-', text.lower()).strip('-')

def print_flashcards(expressions):
    for key, value in expressions.items():
        slug = generate_slug(value)
        flashcard = f"""
# Flashcard: Translate _"{value}"_

‛@slug: {slug}‛

**Translate** _{key}_

---

**{value}**

"""
        print(flashcard)

print_flashcards(expressions)
‛‛‛
		`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// Check the generated flashcards
	require.Equal(t, 3, tr.CountFlashcards())
	flashcard, err := CurrentRepository().FindFlashcardByShortTitle("Translate _\"process in place\"_")
	require.NoError(t, err)
	require.NotNil(t, flashcard)
	assert.Equal(t, "**Translate** _processus en place_", flashcard.Front.String())
	assert.Equal(t, "**process in place**", flashcard.Back.String())

	tr.WriteFile("english.md", `
## Generator: Expressions

‛@interpreter: python3‛

‛‛‛python
import re

expressions = {
    "processus en place" : "process in place",
    "flux de travail" : "workflow",
    "retour sur investissement" : "return on investment",
	"période d'essai" : "probationary period",
}

def print_flashcards(expressions):
    for key, value in expressions.items():
        slug = re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-')
        flashcard = f"""
# Flashcard: Translate _"{value}"_

‛@slug: {slug}‛

**Translate** _{key}_

---

**{value}**

"""
        print(flashcard)

print_flashcards(expressions)
‛‛‛
		`)
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// Check the new flashcard has been added and prior ones preserved
	require.Equal(t, 4, tr.CountFlashcards())
	flashcard, err = CurrentRepository().FindFlashcardBySlug("probationary-period")
	require.NoError(t, err)
	require.NotNil(t, flashcard)

	// Try to change the slugs
	tr.ReplaceLine("english.md",
		18,
		"        slug = re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-')",
		"        slug = 'english-' + re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-')",
	)
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// We still must have the same flashcards (identified by their new slug)
	// and old ones must have been deleted
	require.Equal(t, 4, tr.CountFlashcards())
	oldFlashcard, err := CurrentRepository().FindFlashcardBySlug("probationary-period")
	require.NoError(t, err)
	require.Nil(t, oldFlashcard) // Must no longer exist
	newFlashcard, err := CurrentRepository().FindFlashcardBySlug("english-probationary-period")
	require.NoError(t, err)
	require.NotNil(t, newFlashcard) // Must exist with the new slug
}
