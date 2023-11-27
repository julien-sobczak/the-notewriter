package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	. "time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const Day = 24 * Hour

func TestFlashcard(t *testing.T) {

	t.Run("YAML", func(t *testing.T) {
		SetUpCollectionFromTempDir(t)

		// Make tests reproductible
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

		fileSrc := NewEmptyFile("example.md")
		parsedNoteSrc := MustParseNote("## Flashcard: Syntax\n\nQuestion\n---\nAnswer", "")
		noteSrc := NewNote(fileSrc, nil, parsedNoteSrc)
		flashcardSrc := NewFlashcard(fileSrc, noteSrc)

		// Marshall
		buf := new(bytes.Buffer)
		err := flashcardSrc.Write(buf)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
short_title: Syntax
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: example.md
front_markdown: Question
back_markdown: Answer
front_html: <p>Question</p>
back_html: <p>Answer</p>
front_text: Question
back_text: Answer
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
		`), strings.TrimSpace(buf.String()))

		// Unmarshall
		flashcardDest := new(Flashcard)
		err = flashcardDest.Read(buf)
		require.NoError(t, err)

		// Compare ignore certain attributes
		flashcardSrc.File = nil
		flashcardSrc.Note = nil
		flashcardSrc.new = false
		flashcardSrc.stale = false
		assert.EqualValues(t, flashcardSrc, flashcardDest)

		// Now, try to record a review
		flashcardDest.DueAt = clock.Now().Add(24 * Hour)
		flashcardDest.StudiedAt = clock.Now().Add(-24 * Hour)
		flashcardDest.Settings = map[string]any{
			"interval":    3,
			"ease_factor": 130,
			"lapses":      2,
		}

		// Marshall
		buf.Reset()
		err = flashcardDest.Write(buf)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
short_title: Syntax
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: example.md
front_markdown: Question
back_markdown: Answer
front_html: <p>Question</p>
back_html: <p>Answer</p>
front_text: Question
back_text: Answer
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
due_at: 2023-01-02T01:12:30Z
studied_at: 2022-12-31T01:12:30Z
settings:
    ease_factor: 130
    interval: 3
    lapses: 2
`), strings.TrimSpace(buf.String()))

		// Unmarshall
		flashcardDest = new(Flashcard)
		err = flashcardDest.Read(buf)
		require.NoError(t, err)

		// Check SRS-specific fields
	})

}

func TestFlashcardWithStudy(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		now := HumanTime(t, "2023-02-03 12:00")

		// Make test reproductible
		UseSequenceOID(t)
		FreezeAt(t, now)

		// We will work with just a single file containing three basic flashcards
		root := SetUpCollectionFromTempDir(t)

		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Write the test file
		err := os.WriteFile(filepath.Join(root, "english.md"), []byte(`
# English Vocabulary

## Flashcard: Car

Translate _Voiture_

---

**Car**


## Flashcard: Airplane

Translate _Voiture_

---

**Airplane**

## Flashcard: Motorbike

Translate _Moto_

---

**Motorbike**

		`), 0644)
		require.NoError(t, err)

		// Commit and push
		err = CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)

		// Flashcards must not have SRS fields specified for now
		flashcardCar := MustFindFlashcardByShortTitle(t, "Car")
		flashcardAirplane := MustFindFlashcardByShortTitle(t, "Airplane")
		flashcardMotorbike := MustFindFlashcardByShortTitle(t, "Motorbike")
		assert.Zero(t, flashcardCar.DueAt)
		assert.Zero(t, flashcardAirplane.DueAt)
		assert.Zero(t, flashcardMotorbike.DueAt)

		// Now, we will similate a study session by pushing a new commit in the remote.
		// We will have a first study where we review the car and airplane flashcards
		// and a second study where only the car card is reviewed.
		studyOneTime := now.Add(4 * Hour)
		studyOne := &Study{
			OID:       NewOID(),
			StartedAt: studyOneTime,
			EndedAt:   studyOneTime.Add(1 * Minute),
			Reviews: []*Review{
				{
					FlashcardOID: flashcardCar.OID,
					Feedback:     "easy",
					DurationInMs: 3000,
					CompletedAt:  studyOneTime.Add(30 * Second),
					DueAt:        studyOneTime.Add(1 * Day),
					Settings: map[string]any{
						"interval":   1,
						"easeFactor": 130,
					},
				},
				{
					FlashcardOID: flashcardAirplane.OID,
					Feedback:     "hard",
					DurationInMs: 220,
					CompletedAt:  studyOneTime.Add(1 * Minute),
					DueAt:        studyOneTime.Add(1 * Hour),
					Settings: map[string]any{
						"interval":   0.1,
						"easeFactor": 130,
					},
				},
			},
		}
		studyTwoTime := now.Add(24 * Hour)
		studyTwo := &Study{
			OID:       NewOID(),
			StartedAt: studyTwoTime,
			EndedAt:   studyTwoTime.Add(5 * Second),
			Reviews: []*Review{
				{
					FlashcardOID: flashcardCar.OID,
					Feedback:     "easy",
					DurationInMs: 50,
					CompletedAt:  studyTwoTime.Add(5 * Second),
					DueAt:        studyTwoTime.Add(4 * Day),
					Settings: map[string]any{
						"interval":   4,
						"easeFactor": 150,
					},
				},
			},
		}

		// Write a new pack file with the studies
		packFile := NewPackFile()
		packFile.AppendObject(studyOne)
		packFile.AppendObject(studyTwo)
		packFilePath := filepath.Join(origin, OIDToPath(packFile.OID))
		err = packFile.SaveTo(packFilePath)
		require.NoError(t, err)

		// Append a new commit with this pack file
		originGCPath := filepath.Join(origin, "info/commit-graph")
		originCG, err := NewCommitGraphFromPath(originGCPath)
		require.NoError(t, err)
		commitWithStudies := NewCommitFromPackFiles(packFile)
		originCG.AppendCommit(commitWithStudies)
		err = originCG.SaveTo(originGCPath)
		require.NoError(t, err)

		// Pull the new commit locally
		err = CurrentDB().Pull()
		require.NoError(t, err)

		// Flashcards must have been updated
		flashcardCar = MustFindFlashcardByShortTitle(t, "Car")
		flashcardAirplane = MustFindFlashcardByShortTitle(t, "Airplane")
		flashcardMotorbike = MustFindFlashcardByShortTitle(t, "Motorbike")
		// Check SRS attributes
		assert.Equal(t, studyTwoTime.Add(4*Day), flashcardCar.DueAt)       // Second review wins
		assert.Equal(t, studyOneTime.Add(1*Hour), flashcardAirplane.DueAt) // First unique review "wins"
		assert.Zero(t, flashcardMotorbike.DueAt)                           // Still not reviewed
		assert.Equal(t, studyTwoTime.Add(5*Second), flashcardCar.StudiedAt)
		assert.Equal(t, studyOneTime.Add(1*Minute), flashcardAirplane.StudiedAt)
		assert.Zero(t, flashcardMotorbike.StudiedAt)
		assert.NotEmpty(t, flashcardCar.Settings)
		assert.NotEmpty(t, flashcardAirplane.Settings)
		assert.Empty(t, flashcardMotorbike.Settings)

		// We will redo by injecting an old study that must be ignored
		studyOldTime := now.Add(-1 * 100 * Day) // OLD
		studyOld := &Study{
			OID:       NewOID(),
			StartedAt: studyOldTime,
			EndedAt:   studyOldTime.Add(10 * Second),
			Reviews: []*Review{
				{
					FlashcardOID: flashcardCar.OID,
					Feedback:     "easy",
					DurationInMs: 50,
					CompletedAt:  studyOldTime.Add(10 * Second),
					DueAt:        studyOldTime.Add(10 * Minute),
					Settings: map[string]any{
						"interval":   10,
						"easeFactor": 140,
					},
				},
			},
		}

		// Write a new pack file with the studies
		packFile = NewPackFile()
		packFile.AppendObject(studyOld)
		err = packFile.SaveTo(filepath.Join(origin, OIDToPath(packFile.OID)))
		require.NoError(t, err)

		// Append a new commit with this pack file
		originCG, err = NewCommitGraphFromPath(originGCPath)
		require.NoError(t, err)
		originCG.AppendCommit(NewCommitFromPackFiles(packFile))
		err = originCG.SaveTo(originGCPath)
		require.NoError(t, err)

		// Pull the new commit locally
		err = CurrentDB().Pull()
		require.NoError(t, err)

		// The flashcard reviewed in the past must not have been updated
		flashcardCar = MustFindFlashcardByShortTitle(t, "Car")
		// Check SRS attributes
		assert.Equal(t, studyTwoTime.Add(4*Day), flashcardCar.DueAt) // Second review wins
		assert.Equal(t, studyTwoTime.Add(5*Second), flashcardCar.StudiedAt)
		assert.Equal(t, map[string]any{
			"interval":   4,
			"easeFactor": 150,
		}, flashcardCar.Settings)

		// Run GC to ensure Study objects are not reclaimed even if not really persisted in database directly
		err = CurrentDB().GC()
		require.NoError(t, err)
		commitWithStudiesAfterGC, ok := CurrentDB().ReadCommit(commitWithStudies.OID)
		require.True(t, ok)
		packFiles, err := CurrentDB().ReadPackFilesFromCommit(commitWithStudiesAfterGC)
		require.NoError(t, err)
		assert.Len(t, packFiles, 1)                // We created a single pack file for this commit that must still exists...
		assert.Len(t, packFiles[0].PackObjects, 2) // ... and contains our two studies
	})
}
