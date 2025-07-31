package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNote(t *testing.T) {
	tr := NewTestRepository(t, WithFreezeNow())

	tr.AssertNoNotes()

	createdAt := clock.Now()
	note := &Note{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		RelativePath: "go.md",
		Slug:         "go-note-golang-history",
		Type:         "Note",
		Title:        "Note: Golang History",
		LongTitle:    "Go / Golang History",
		ShortTitle:   "Golang History",
		Wikilink:     "go#Note: Golang History",
		Attributes: AttributeSet(map[string]any{
			"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
			"tags":   []string{"go"},
			"title":  "Golang History",
		}),
		Tags: TagSet([]string{"go"}),
		Line: 8,
		Content: markdown.Document(text.UnescapeTestContent(`
## Note: Golang History

‛#history‛

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

> Go was created in 2007
`)),
		Hash: "40411b52dcd5eccdb5845ef8e8fc18bbff3c3411",
		Body: markdown.Document(text.UnescapeTestContent(`‛#history‛

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.`)),
		Comment:   "Go was created in 2007",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		IndexedAt: createdAt,
	}

	// Save
	require.NoError(t, note.Save())
	require.Equal(t, 1, tr.CountNotes())

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadNoteByOID(note.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, note.OID, actual.OID)
	assert.Equal(t, note.PackFileOID, actual.PackFileOID)
	assert.Equal(t, note.FileOID, actual.FileOID)
	assert.Equal(t, note.Type, actual.Type)
	assert.Equal(t, note.Slug, actual.Slug)
	assert.Equal(t, note.Title, actual.Title)
	assert.Equal(t, note.ShortTitle, actual.ShortTitle)
	assert.Equal(t, note.LongTitle, actual.LongTitle)
	assert.Equal(t, note.RelativePath, actual.RelativePath)
	assert.Equal(t, note.Wikilink, actual.Wikilink)
	assert.Equal(t, note.Attributes, actual.Attributes)
	assert.Equal(t, note.Tags, actual.Tags)
	assert.Equal(t, note.Line, actual.Line)
	assert.Equal(t, note.Content, actual.Content)
	assert.Equal(t, note.Hash, actual.Hash)
	assert.Equal(t, note.Body, actual.Body)
	assert.Equal(t, note.Comment, actual.Comment)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.IndexedAt, 1*time.Second)

	// Update
	actual.Comment = "Golang was created in 2007"
	require.NoError(t, actual.Save())
	require.Equal(t, 1, tr.CountNotes())

	// ...and compare again
	actual, err = CurrentRepository().LoadNoteByOID(note.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, note.OID, actual.OID) // Must have found the previous one
	assert.Contains(t, actual.Comment, "Golang was created in 2007")

	// Delete
	require.NoError(t, note.Delete())
	tr.AssertNoNotes()
}

func TestNoteHooks(t *testing.T) {

	t.Run("HasHooks", func(t *testing.T) {
		tr := NewTestRepository(t)

		// Insert the note
		tr.WriteFile("go.md", `# Go

## Note: Golang History

‛@hook: gist‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
		`)

		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit()
		require.NoError(t, err)

		// Check in database
		note := MustFindNoteByTitle(t, "Note: Golang History")
		assert.True(t, note.HasHooks())
		assert.Equal(t, []string{"gist"}, note.GetHooks())

		// Check in objects
		packFile, err := CurrentIndex().ReadLastPackFile("go.md")
		require.NoError(t, err)
		require.NotNil(t, packFile)

		packObjects := packFile.PackObjects
		require.Len(t, packObjects, 2) // File + Note
		note, ok := packObjects[1].Read().(*Note)
		require.True(t, ok)
		assert.True(t, note.HasHooks())
		assert.Equal(t, []string{"gist"}, note.GetHooks())
	})

}

func TestNoteFormats(t *testing.T) {
	testutil.FreezeOn(t, "2023-01-01 01:12:30")

	note := &Note{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		RelativePath: "go.md",
		Slug:         "go-note-golang-history",
		Type:         "Note",
		Title:        "Note: Golang History",
		LongTitle:    "Go / Golang History",
		ShortTitle:   "Golang History",
		Wikilink:     "go#Note: Golang History",
		Attributes: AttributeSet(map[string]any{
			"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
			"tags":   []string{"go"},
			"title":  "Golang History",
		}),
		Tags: TagSet([]string{"go"}),
		Line: 8,
		Content: markdown.Document(text.UnescapeTestContent(`## Note: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.`)),
		Hash: "40411b52dcd5eccdb5845ef8e8fc18bbff3c3411",
		Body: markdown.Document(text.UnescapeTestContent(`‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.`)),
		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),
		IndexedAt: clock.Now(),
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := note.ToYAML()

		expected := text.UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
slug: go-note-golang-history
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
file_oid: 3e8d915d4e524560ae8a2e5a45553f3034b391a2
type: Note
title: 'Note: Golang History'
long_title: Go / Golang History
short_title: Golang History
relative_path: go.md
wikilink: 'go#Note: Golang History'
attributes:
  source: https://en.wikipedia.org/wiki/Go_(programming_language)
  tags:
    - go
  title: Golang History
tags:
  - go
line: 8
content: |-
  ## Note: Golang History

  ‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

  Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
content_hash: 40411b52dcd5eccdb5845ef8e8fc18bbff3c3411
body: |-
  ‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

  Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := note.ToJSON()
		expected := text.UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "slug": "go-note-golang-history",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "file_oid": "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
  "type": "Note",
  "title": "Note: Golang History",
  "long_title": "Go / Golang History",
  "short_title": "Golang History",
  "relative_path": "go.md",
  "wikilink": "go#Note: Golang History",
  "attributes": {
    "source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
    "tags": [
      "go"
    ],
    "title": "Golang History"
  },
  "tags": [
    "go"
  ],
  "line": 8,
  "content": "## Note: Golang History\n\n‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛\n\nGolang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.",
  "content_hash": "40411b52dcd5eccdb5845ef8e8fc18bbff3c3411",
  "body": "‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛\n\nGolang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := note.ToMarkdown()
		expected := text.UnescapeTestContent(`
# Note: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}

func TestSearchNotes(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestNoteFTS"))

	CurrentLogger().SetVerboseLevel(VerboseTrace)

	// Insert the note
	parsedFile := tr.ParseFile("note.md")

	dummyPackFile := DummyPackFile()

	file, err := NewFile(dummyPackFile, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Note: FTS5")
	require.True(t, ok)
	note, err := NewNote(dummyPackFile, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Save())

	// Search the note using a full-text query
	notes, err := CurrentRepository().SearchNotes("type:Note fts5")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Update the note content
	note.Content = "full-text"
	require.NoError(t, note.Save())

	// Search the note using a full-text query
	notes, err = CurrentRepository().SearchNotes("type:Note full")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Delete the note
	require.NoError(t, note.Delete())
	require.NoError(t, err)

	// Check the note is no longer present
	notes, err = CurrentRepository().SearchNotes("type:note full")
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}

func TestNoteOperations(t *testing.T) {

	t.Run("Mark and Unmark", func(t *testing.T) {
		tr := NewTestRepository(t)

		// Insert the note
		tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)

		_, err := CurrentRepository().Add(PathSpecs{"python.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit()
		require.NoError(t, err)

		// Check the note is present
		note := MustFindNoteByTitle(t, "Flashcard: Python's creator")

		// Mark the note
		note.Mark(clock.Now())

		// Save the note
		require.NoError(t, note.SaveMetadata())

		// Reread the note and check the marked status
		note, err = CurrentRepository().LoadNoteByOID(note.OID)
		require.NoError(t, err)
		assert.True(t, note.Marked)

		// Unmark the note, save it, and check the status
		note.Unmark(clock.Now())
		require.NoError(t, note.SaveMetadata())
		note, err = CurrentRepository().LoadNoteByOID(note.OID)
		require.NoError(t, err)
		assert.False(t, note.Marked)
	})

	t.Run("Annotate", func(t *testing.T) {
		tr := NewTestRepository(t, WithFreezeNow())

		date1 := clock.Now()

		// Insert the note
		tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)

		_, err := CurrentRepository().Add(PathSpecs{"python.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit()
		require.NoError(t, err)

		// Check the note is present
		note := MustFindNoteByTitle(t, "Flashcard: Python's creator")

		// Mark the note
		note.AddAnnotation(clock.Now(), Annotation{
			OID:  "42d74d967d9b4e989502647ac510777ca1e22f4a",
			Text: "Use Markdown emphasis",
		})

		// Save the note
		require.NoError(t, note.SaveMetadata())

		// Reread the note and check the annotation has been saved
		note, err = CurrentRepository().LoadNoteByOID(note.OID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(note.Annotations))
		assert.Equal(t, "Use Markdown emphasis", note.Annotations[0].Text)
		assert.Equal(t, date1.UTC(), note.Annotations[0].CreatedAt.UTC())

		date2 := tr.FastForward(1 * time.Hour)

		// Add a second annotation
		note.AddAnnotation(clock.Now(), Annotation{
			OID:  "639c1b9964ad45c9b50cb79c2daa03b59f57a01d",
			Text: "Delete",
		})
		require.NoError(t, note.SaveMetadata())
		note, err = CurrentRepository().LoadNoteByOID(note.OID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(note.Annotations))
		assert.Equal(t, "Use Markdown emphasis", note.Annotations[0].Text)
		assert.Equal(t, date1.UTC(), note.Annotations[0].CreatedAt.UTC())
		assert.Equal(t, "Delete", note.Annotations[1].Text)
		assert.Equal(t, date2.UTC(), note.Annotations[1].CreatedAt.UTC())

		// Delete the first annotation
		date3 := tr.FastForward(1 * time.Hour)
		note.RemoveAnnotation(date3, note.Annotations[0])
		require.NoError(t, note.SaveMetadata())
		note, err = CurrentRepository().LoadNoteByOID(note.OID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(note.Annotations))
		assert.Equal(t, "Delete", note.Annotations[0].Text)
		assert.Equal(t, date2.UTC(), note.Annotations[0].CreatedAt.UTC())
	})

}

/* Helpers */

func MustFindNoteByTitle(t *testing.T, title string) *Note {
	note, err := CurrentRepository().FindNoteByTitle(title)
	require.NoError(t, err)
	require.NotNil(t, note)
	return note
}
