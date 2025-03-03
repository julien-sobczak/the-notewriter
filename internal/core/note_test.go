package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNote(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)
	AssertNoNotes(t)

	createdAt := clock.Now()
	note := &Note{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		RelativePath: "go.md",
		Slug:         "go-reference-golang-history",
		NoteKind:     KindReference,
		Title:        "Reference: Golang History",
		LongTitle:    "Go / Golang History",
		ShortTitle:   "Golang History",
		Wikilink:     "go#Reference: Golang History",
		Attributes: AttributeSet(map[string]any{
			"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
			"tags":   []string{"go"},
			"title":  "Golang History",
		}),
		Tags: TagSet([]string{"go"}),
		Line: 8,
		Content: markdown.Document(text.UnescapeTestContent(`
## Reference: Golang History

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
	require.Equal(t, 1, MustCountNotes(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadNoteByOID(note.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, note.OID, actual.OID)
	assert.Equal(t, note.PackFileOID, actual.PackFileOID)
	assert.Equal(t, note.FileOID, actual.FileOID)
	assert.Equal(t, note.NoteKind, actual.NoteKind)
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
	require.Equal(t, 1, MustCountNotes(t))

	// ...and compare again
	actual, err = CurrentRepository().LoadNoteByOID(note.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, note.OID, actual.OID) // Must have found the previous one
	assert.Contains(t, actual.Comment, "Golang was created in 2007")

	// Delete
	require.NoError(t, note.Delete())
	AssertNoNotes(t)
}

func TestNoteFormats(t *testing.T) {
	FreezeOn(t, "2023-01-01 01:12:30")

	note := &Note{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		RelativePath: "go.md",
		Slug:         "go-reference-golang-history",
		NoteKind:     KindReference,
		Title:        "Reference: Golang History",
		LongTitle:    "Go / Golang History",
		ShortTitle:   "Golang History",
		Wikilink:     "go#Reference: Golang History",
		Attributes: AttributeSet(map[string]any{
			"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
			"tags":   []string{"go"},
			"title":  "Golang History",
		}),
		Tags: TagSet([]string{"go"}),
		Line: 8,
		Content: markdown.Document(text.UnescapeTestContent(`## Reference: Golang History

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
slug: go-reference-golang-history
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
file_oid: 3e8d915d4e524560ae8a2e5a45553f3034b391a2
kind: reference
title: 'Reference: Golang History'
long_title: Go / Golang History
short_title: Golang History
relative_path: go.md
wikilink: 'go#Reference: Golang History'
attributes:
  source: https://en.wikipedia.org/wiki/Go_(programming_language)
  tags:
    - go
  title: Golang History
tags:
  - go
line: 8
content: |-
  ## Reference: Golang History

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
  "slug": "go-reference-golang-history",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "file_oid": "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
  "kind": "reference",
  "title": "Reference: Golang History",
  "long_title": "Go / Golang History",
  "short_title": "Golang History",
  "relative_path": "go.md",
  "wikilink": "go#Reference: Golang History",
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
  "content": "## Reference: Golang History\n\n‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛\n\nGolang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.",
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
# Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}

func TestSearchNotes(t *testing.T) {
	SetUpRepositoryFromGoldenDirNamed(t, "TestNoteFTS")

	CurrentLogger().SetVerboseLevel(VerboseTrace)

	// Insert the note
	parsedFile := ParseFileFromRelativePath(t, "note.md")

	dummyPackFile := DummyPackFile()

	file, err := NewFile(dummyPackFile, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: FTS5")
	require.True(t, ok)
	note, err := NewNote(dummyPackFile, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Save())

	// Search the note using a full-text query
	notes, err := CurrentRepository().SearchNotes("kind:reference fts5")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Update the note content
	note.Content = "full-text"
	require.NoError(t, note.Save())

	// Search the note using a full-text query
	notes, err = CurrentRepository().SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Delete the note
	require.NoError(t, note.Delete())
	require.NoError(t, err)

	// Check the note is no longer present
	notes, err = CurrentRepository().SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}
