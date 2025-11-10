package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectPersistance(t *testing.T) {
	// The goal of this test is to populate the database to fill all columns
	// and check all values are correctly persisted.
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithSequenceOIDs(),

		// Create the minimal setup to have all columns populated in DB
		WithFile("programming/index.md", `
---
tags: programming
---
# Programming
---
`),
		WithFile("programming/python.md", `
---
tags: python
---
# Python

## Note: Logo

![Logo](./medias/logo-python.svg)

## Note: Conferences

* [PyCon France](https://www.pycon.fr/2025/ "PyCon #go/pycon-fr") ‛#reminder-2025-10-30‛
* [EuroPython](hhttps://ep2025.europython.eu/ "#go/europython") ‛#reminder-2025-07-14‛

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum

## Syntax

### Note: Special Methods

‛‛‛python
# Demonstrating (mis)use of special methods
class SillyClass:
    def __getitem__(self, key):
        """ Determines behavior of ‛self[key]‛ """
        return [True, False, True, False]
‛‛‛

> Make your type Python-friendly
`))

	// Add to persist objects in DB
	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)

	require.Equal(t, 2, tr.CountFiles())
	require.Equal(t, 1, tr.CountMedias())
	require.Equal(t, 4, tr.CountNotes()) // including the flashcard note
	require.Equal(t, 1, tr.CountFlashcards())
	require.Equal(t, 1, tr.CountGotos())
	require.Equal(t, 2, tr.CountReminders())
	require.Equal(t, 0, tr.CountMemories())

	// Read a single object of each kind and check all fields

	// File
	file := MustFindFileByRelativePath(t, "programming/python.md")
	assert.NotEmpty(t, file.OID)
	assert.NotEmpty(t, file.PackFileOID)
	assert.Equal(t, "programming-python", file.Slug)
	assert.Equal(t, "programming/python", file.Wikilink)
	assert.Equal(t, markdown.FrontMatter("tags: python\n"), file.FrontMatter)
	assert.Equal(t, AttributeSet(map[string]any{
		"tags": []string{"programming", "python"},
	}), file.Attributes)
	assert.Equal(t, "Python", file.Title.String())
	assert.Equal(t, "Python", file.ShortTitle.String())
	assert.True(t, strings.HasPrefix(file.Body.String(), "# Python\n\n"))
	assert.Equal(t, 5, file.BodyLine)
	assert.Greater(t, file.Size, int64(0))
	assert.NotEmpty(t, file.Hash)
	assert.NotNil(t, file.MTime.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), file.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), file.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), file.IndexedAt.Truncate(time.Second))

	// Media
	media := MustFindMediaByRelativePath(t, "programming/medias/logo-python.svg")
	assert.NotEmpty(t, media.OID)
	assert.NotEmpty(t, media.PackFileOID)                   // Different pack file than the Markdown file
	assert.NotEqual(t, file.PackFileOID, media.PackFileOID) // Different pack file than the Markdown file
	assert.Equal(t, "programming/medias/logo-python.svg", media.RelativePath)
	assert.Equal(t, KindPicture, media.MediaKind)
	assert.True(t, media.Dangling)
	assert.Equal(t, ".svg", media.Extension)
	assert.Zero(t, media.MTime)
	assert.Zero(t, media.Hash)
	assert.Zero(t, media.Size)
	assert.Empty(t, media.BlobRefs)
	assert.Equal(t, clock.Now().Truncate(time.Second), media.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), media.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), media.IndexedAt.Truncate(time.Second))

	// Note
	note := MustFindNoteByTitle(t, "Note: Special Methods")
	// A unique identifier among all files
	assert.NotEmpty(t, note.OID, "")
	assert.Equal(t, file.PackFileOID, note.PackFileOID)
	assert.Equal(t, file.OID, note.FileOID)
	assert.Equal(t, "programming-python-note-special-methods", note.Slug)
	assert.Equal(t, "Note", note.Type)
	assert.Equal(t, markdown.Document("Note: Special Methods"), note.Title)
	assert.Equal(t, markdown.Document("Python / Syntax / Special Methods"), note.LongTitle)
	assert.Equal(t, markdown.Document("Special Methods"), note.ShortTitle)
	assert.Equal(t, "programming/python.md", note.RelativePath)
	assert.Equal(t, "programming/python#Note: Special Methods", note.Wikilink)
	assert.Equal(t, AttributeSet(map[string]any{
		"tags":  []string{"programming", "python"},
		"title": "Special Methods",
	}), note.Attributes)
	assert.Equal(t, TagSet([]string{"programming", "python"}), note.Tags)
	assert.Greater(t, note.Line, 0)
	assert.True(t, strings.HasPrefix(note.Content.String(), "### Note: Special Methods"))
	assert.NotEmpty(t, note.Hash)
	assert.True(t, strings.HasPrefix(note.Body.String(), "```python"))
	assert.Equal(t, markdown.Document("Make your type Python-friendly"), note.Comment)
	assert.Equal(t, clock.Now().Truncate(time.Second), note.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), note.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), note.IndexedAt.Truncate(time.Second))

	// Flashcard
	note = MustFindNoteByTitle(t, "Flashcard: Python's creator")
	flashcard := tr.FindFlashcardByShortTitle("Python's creator")
	assert.NotEmpty(t, flashcard.OID)
	assert.Equal(t, file.PackFileOID, flashcard.PackFileOID)
	assert.Equal(t, file.OID, flashcard.FileOID)
	assert.Equal(t, note.OID, flashcard.NoteOID)
	assert.Equal(t, "programming/python.md", flashcard.RelativePath)
	assert.Equal(t, "programming-python-flashcard-pythons-creator", flashcard.Slug)
	assert.Equal(t, markdown.Document("Python's creator"), flashcard.ShortTitle)
	assert.Equal(t, TagSet([]string{"programming", "python"}), flashcard.Tags)
	assert.Equal(t, markdown.Document("Who invented Python?"), flashcard.Front)
	assert.Equal(t, markdown.Document("Guido van Rossum"), flashcard.Back)
	assert.Equal(t, clock.Now().Truncate(time.Second), flashcard.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), flashcard.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), flashcard.IndexedAt.Truncate(time.Second))

	// Reminder
	note = MustFindNoteByTitle(t, "Note: Conferences")
	reminder := MustFindReminderByDescription(t, `Conferences / [PyCon France](https://www.pycon.fr/2025/ "PyCon #go/pycon-fr")`)
	assert.NotEmpty(t, reminder.OID)
	assert.Equal(t, file.PackFileOID, reminder.PackFileOID)
	assert.Equal(t, file.OID, reminder.FileOID)
	assert.Equal(t, note.OID, reminder.NoteOID)
	assert.Equal(t, "programming/python.md", reminder.RelativePath)
	assert.Equal(t, markdown.Document(`Conferences / [PyCon France](https://www.pycon.fr/2025/ "PyCon #go/pycon-fr")`), reminder.Description)
	assert.Equal(t, "#reminder-2025-10-30", reminder.Tag)
	assert.Equal(t, clock.Now().Truncate(time.Second), reminder.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), reminder.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), reminder.IndexedAt.Truncate(time.Second))

	// Goto
	gotoLink := MustFindGotoByName(t, "pycon-fr")
	assert.NotEmpty(t, gotoLink.OID)
	assert.Equal(t, file.PackFileOID, gotoLink.PackFileOID)
	assert.Equal(t, note.OID, gotoLink.NoteOID)
	assert.Equal(t, "programming/python.md", gotoLink.RelativePath)
	assert.Equal(t, markdown.Document("PyCon France"), gotoLink.Text)
	assert.Equal(t, "https://www.pycon.fr/2025/", string(gotoLink.URL))
	assert.Equal(t, "PyCon", gotoLink.Title)
	assert.Equal(t, "pycon-fr", gotoLink.Name)
	assert.Equal(t, clock.Now().Truncate(time.Second), gotoLink.CreatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), gotoLink.UpdatedAt.Truncate(time.Second))
	assert.Equal(t, clock.Now().Truncate(time.Second), gotoLink.IndexedAt.Truncate(time.Second))
}

func TestStatsOnDisk(t *testing.T) {
	NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

	stats, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)
	assert.Equal(t, map[string]int{
		"file":      0,
		"note":      0,
		"flashcard": 0,
		"media":     0,
		"link":      0,
		"reminder":  0,
		"memory":    0,
	}, stats.Objects)
	require.Equal(t, 0, stats.IndexObjects)
	require.Equal(t, int64(0), stats.TotalSizeKB)

	// Add
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	statsAdd, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	// Objects are already written before a commit
	assert.Greater(t, statsAdd.Objects["file"], 0)
	assert.Greater(t, statsAdd.Objects["note"], 0)
	assert.Greater(t, statsAdd.Objects["flashcard"], 0)
	assert.Greater(t, statsAdd.Objects["media"], 0)
	assert.Greater(t, statsAdd.Objects["link"], 0)
	assert.Greater(t, statsAdd.Objects["reminder"], 0)
	assert.Greater(t, statsAdd.Objects["memory"], 0)
	assert.Greater(t, statsAdd.Blobs, 0)
	assert.Greater(t, statsAdd.ObjectFiles, 0)
	assert.Greater(t, statsAdd.IndexObjects, 0)
	assert.Greater(t, statsAdd.TotalSizeKB, int64(0))

	// Commit
	err = CurrentRepository().Commit()
	require.NoError(t, err)
	statsCommit, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	assert.Equal(t, statsAdd.Objects["file"], statsCommit.Objects["file"])
	assert.Equal(t, statsAdd.Objects["note"], statsCommit.Objects["note"])
	assert.Equal(t, statsAdd.Objects["flashcard"], statsCommit.Objects["flashcard"])
	assert.Equal(t, statsAdd.Objects["media"], statsCommit.Objects["media"])
	assert.Equal(t, statsAdd.Objects["link"], statsCommit.Objects["link"])
	assert.Equal(t, statsAdd.Objects["memory"], statsCommit.Objects["memory"])
	assert.Equal(t, statsAdd.Objects["memory"], statsCommit.Objects["memory"])
	assert.Equal(t, statsAdd.Blobs, statsCommit.Blobs)
	assert.Equal(t, statsAdd.ObjectFiles, statsCommit.ObjectFiles)
	assert.Equal(t, statsAdd.IndexObjects, statsCommit.IndexObjects)
	assert.Equal(t, statsAdd.TotalSizeKB, statsCommit.TotalSizeKB)
}
