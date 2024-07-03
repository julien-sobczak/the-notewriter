package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashcard(t *testing.T) {
	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
# Go

## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./medias/go.svg)
`))

	UseSequenceOID(t)
	AssertNoFlashcards(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Flashcard: Golang Logo")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Save())

	// Create
	flashcard, err := NewFlashcard(file, note, parsedNote.Flashcard)
	require.NoError(t, err)

	// Check all fields
	assert.NotNil(t, flashcard.OID)
	assert.Equal(t, file.OID, flashcard.FileOID)
	assert.Equal(t, note.OID, flashcard.NoteOID)
	assert.Equal(t, note.RelativePath, flashcard.RelativePath)
	assert.Equal(t, note.Slug, flashcard.Slug)
	assert.Equal(t, note.ShortTitle, flashcard.ShortTitle)
	assert.Equal(t, note.Tags, flashcard.Tags)
	assert.Equal(t, "What does the **Golang logo** represent?", flashcard.Front.String())
	assert.Equal(t, "A **gopher**.\n\n![Logo](./medias/go.svg)", flashcard.Back.String())
	assert.Equal(t, clock.Now(), flashcard.CreatedAt)
	assert.Equal(t, clock.Now(), flashcard.UpdatedAt)
	assert.Empty(t, flashcard.DeletedAt)
	assert.Empty(t, flashcard.LastCheckedAt)

	// Save
	require.NoError(t, flashcard.Save())
	require.Equal(t, 1, MustCountFlashcards(t))

	// Reread and check the flashcard
	actual, err := CurrentRepository().LoadFlashcardByOID(flashcard.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, flashcard.OID, actual.OID)
	assert.Equal(t, flashcard.Slug, actual.Slug)
	assert.Equal(t, flashcard.FileOID, actual.FileOID)
	assert.Equal(t, flashcard.NoteOID, actual.NoteOID)
	assert.Equal(t, flashcard.RelativePath, actual.RelativePath)
	assert.Equal(t, flashcard.Slug, actual.Slug)
	assert.Equal(t, flashcard.ShortTitle, actual.ShortTitle)
	assert.Equal(t, flashcard.Tags, actual.Tags)
	assert.Equal(t, flashcard.Front, actual.Front)
	assert.Equal(t, flashcard.Back, actual.Back)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)
	assert.Empty(t, flashcard.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	ReplaceLine(t, filepath.Join(root, "go.md"), 6,
		"What does the **Golang logo** represent?",
		"What is the **Golang logo**?")
	parsedFile, err = ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedNote, ok = parsedFile.FindNoteByShortTitle("Golang Logo")
	require.True(t, ok)
	newNote, err := NewOrExistingNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NoError(t, newNote.Save())
	newFlashcard, err := NewOrExistingFlashcard(file, newNote, parsedNote.Flashcard)
	require.NoError(t, err)
	require.NoError(t, newFlashcard.Save())
	// ... and compare
	assert.Equal(t, flashcard.OID, newFlashcard.OID) // Must have found the previous file
	assert.Contains(t, newFlashcard.Front, "What is the **Golang logo**?")

	// Retrieve
	updatedFlashcard, err := CurrentRepository().LoadFlashcardByOID(newFlashcard.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedFlashcard.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedFlashcard.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedFlashcard.LastCheckedAt, 1*time.Second)

	// Delete
	require.NoError(t, flashcard.Delete())
	assert.Equal(t, clock.Now(), flashcard.DeletedAt)

	AssertNoFlashcards(t)
}

func TestFlashcardFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
# Go

## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.
`))

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)

	// Init the flashcard
	parsedNote, ok := parsedFile.FindNoteByTitle("Flashcard: Golang Logo")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NotNil(t, parsedNote.Flashcard)
	flashcard, err := NewFlashcard(file, note, parsedNote.Flashcard)
	require.NoError(t, err)

	t.Run("ToYAML", func(t *testing.T) {
		actual := flashcard.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: go.md
slug: go-flashcard-golang-logo
short_title: Golang Logo
front: What does the **Golang logo** represent?
back: A **gopher**.
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := flashcard.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "file_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "note_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "relative_path": "go.md",
  "slug": "go-flashcard-golang-logo",
  "short_title": "Golang Logo",
  "front": "What does the **Golang logo** represent?",
  "back": "A **gopher**.",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "deleted_at": "0001-01-01T00:00:00Z",
  "due_at": "0001-01-01T00:00:00Z",
  "studied_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := flashcard.ToMarkdown()
		expected := UnescapeTestContent(`
What does the **Golang logo** represent?

---

A **gopher**.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}
