package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashcard(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)

	AssertNoFlashcards(t)

	createdAt := clock.Now()
	flashcard := &Flashcard{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "project.md",
		Slug:         "go-flashcard-golang-logo",
		ShortTitle:   "Golang Logo",
		Tags:         []string{"go"},
		Front:        "What does the **Golang logo** represent?",
		Back:         "A **gopher**.\n\n![Logo](./medias/go.svg)",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		IndexedAt:    createdAt,
	}

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
	assert.WithinDuration(t, clock.Now(), actual.IndexedAt, 1*time.Second)

	// Force update
	actual.Front = "What is the **Golang logo**?"
	require.NoError(t, actual.Save())
	require.Equal(t, 1, MustCountFlashcards(t))

	// ... and compare again
	actual, err = CurrentRepository().LoadFlashcardByOID(flashcard.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, flashcard.OID, actual.OID) // Must have found the previous file
	assert.Contains(t, actual.Front, "What is the **Golang logo**?")

	// Delete
	require.NoError(t, flashcard.Delete())
	AssertNoFlashcards(t)
}

func TestFlashcardFormats(t *testing.T) {
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	flashcard := &Flashcard{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:      "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "go.md",
		Slug:         "go-flashcard-golang-logo",
		ShortTitle:   "Golang Logo",
		Tags:         []string{"go"},
		Front:        "What does the **Golang logo** represent?",
		Back:         "A **gopher**.",
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		IndexedAt:    clock.Now(),
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := flashcard.ToYAML()

		expected := text.UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
file_oid: 3e8d915d4e524560ae8a2e5a45553f3034b391a2
note_oid: 52d02a28a961471db62c6d40d30639dafe4aba00
relative_path: go.md
slug: go-flashcard-golang-logo
short_title: Golang Logo
tags:
  - go
front: What does the **Golang logo** represent?
back: A **gopher**.
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := flashcard.ToJSON()
		expected := text.UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "file_oid": "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
  "note_oid": "52d02a28a961471db62c6d40d30639dafe4aba00",
  "relative_path": "go.md",
  "slug": "go-flashcard-golang-logo",
  "short_title": "Golang Logo",
  "tags": [
    "go"
  ],
  "front": "What does the **Golang logo** represent?",
  "back": "A **gopher**.",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z",
  "due_at": "0001-01-01T00:00:00Z",
  "studied_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := flashcard.ToMarkdown()
		expected := text.UnescapeTestContent(`
What does the **Golang logo** represent?

---

A **gopher**.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}
