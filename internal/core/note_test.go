package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNote(t *testing.T) {
	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`---
tags:
- go
---

# Go

## Reference: Golang History

‛#history‛

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

> Go was created in 2007
`))

	UseSequenceOID(t)
	AssertNoNotes(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)

	// Create
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	noteCopy, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NotEqual(t, note.OID, noteCopy.OID)

	// Check all fields
	assert.Equal(t, "0000000000000000000000000000000000000002", note.OID)
	assert.Equal(t, file.OID, note.FileOID)
	assert.Empty(t, note.ParentNoteOID)
	assert.Equal(t, KindReference, note.NoteKind)
	assert.Equal(t, "go-reference-golang-history", note.Slug)
	assert.Equal(t, markdown.Document("Reference: Golang History"), note.Title)
	assert.Equal(t, markdown.Document("Golang History"), note.ShortTitle)
	assert.Equal(t, markdown.Document("Go / Golang History"), note.LongTitle)
	assert.Equal(t, "go.md", note.RelativePath)
	assert.Equal(t, "go#Reference: Golang History", note.Wikilink)
	assert.Equal(t, AttributeSet(map[string]any{
		"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
		"tags":   []string{"go", "history"},
		"title":  "Golang History",
	}), note.Attributes)
	assert.Equal(t, TagSet([]string{"history", "go"}), note.Tags)
	assert.Equal(t, 8, note.Line)
	assert.Equal(t, markdown.Document("## Reference: Golang History\n\n`#history`\n\n`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n\n> Go was created in 2007"), note.Content)
	assert.Equal(t, "96ce446651b290347d7c1bd87d636da441c1b34a", note.Hash)
	assert.Equal(t, markdown.Document("`#history`\n\n`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007."), note.Body)
	assert.Equal(t, markdown.Document("Go was created in 2007"), note.Comment)
	assert.Equal(t, clock.Now(), note.CreatedAt)
	assert.Equal(t, clock.Now(), note.UpdatedAt)
	assert.Empty(t, note.DeletedAt)
	assert.Empty(t, note.LastCheckedAt)

	// Save
	require.NoError(t, note.Save())
	require.Equal(t, 1, MustCountNotes(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadNoteByOID(note.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, note.OID, actual.OID)
	assert.Equal(t, note.FileOID, actual.FileOID)
	assert.Equal(t, note.ParentNoteOID, actual.ParentNoteOID)
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
	assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)
	assert.Empty(t, actual.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	ReplaceLine(t, filepath.Join(root, "go.md"), 16,
		"> Go was created in 2007",
		"> Golang was created in 2007")

	// Recreate...
	parsedFile, err = ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedNote, ok = parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	newNote, err := NewOrExistingNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NoError(t, newNote.Save())
	// ...and compare
	assert.Equal(t, note.OID, newNote.OID) // Must have found the previous one
	assert.Contains(t, newNote.Comment, "Golang was created in 2007")

	// Retrieve
	updatedNote, err := CurrentRepository().LoadNoteByOID(newNote.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedNote.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedNote.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedNote.LastCheckedAt, 1*time.Second)

	// Delete
	require.NoError(t, note.Delete())
	assert.Equal(t, clock.Now(), note.DeletedAt)

	AssertNoNotes(t)
}

func TestNoteWithParent(t *testing.T) {
	content := UnescapeTestContent(`
# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛
‛@tags: go‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

### Flashcard: Golang History

‛#study‛

(Go) **When** was created Go?

---

2007
`)
	root := SetUpRepositoryFromTempDir(t)
	err := os.WriteFile(filepath.Join(root, "go.md"), []byte(UnescapeTestContent(content)), 0644)
	require.NoError(t, err)

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())

	// Init the notes
	parentParsedFile, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	parentNote, err := NewNote(file, nil, parentParsedFile)
	require.NoError(t, err)
	childParsedFile, ok := parsedFile.FindNoteByTitle("Flashcard: Golang History")
	require.True(t, ok)
	childNote, err := NewNote(file, parentNote, childParsedFile)
	require.NoError(t, err)

	// Check attributes
	assert.Equal(t, parentNote.OID, childNote.ParentNoteOID)
	assert.Equal(t, AttributeSet(map[string]any{
		"tags":  []string{"go", "study"},
		"title": "Golang History",
	}), childNote.Attributes)
	assert.ElementsMatch(t, []string{"go", "study"}, childNote.Tags.AsList())
}

func TestNoteOld(t *testing.T) {

	t.Run("NewNote", func(t *testing.T) {

		content := `---
tags:
- go
---

# Go

## Reference: Golang History

‛#history‛

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

> Go was created in 2007
`
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		note, err := NewNote(file, nil, parsedFile.Notes[0])
		require.NoError(t, err)

		// Check all fields
		assert.Equal(t, "42d74d967d9b4e989502647ac510777ca1e22f4a", note.OID)
		assert.Equal(t, "go-reference-golang-history", note.Slug)
		assert.Equal(t, file.OID, note.FileOID)
		assert.Empty(t, note.ParentNoteOID)
		assert.Equal(t, KindReference, note.NoteKind)
		assert.Equal(t, markdown.Document("Reference: Golang History"), note.Title)
		assert.Equal(t, markdown.Document("Golang History"), note.ShortTitle)
		assert.Equal(t, markdown.Document("Go / Golang History"), note.LongTitle)
		assert.Equal(t, "go.md", note.RelativePath)
		assert.Equal(t, "go#Reference: Golang History", note.Wikilink)
		assert.Equal(t, AttributeSet(map[string]any{
			"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
			"tags":   []string{"go", "history"},
			"title":  "Golang History",
		}), note.Attributes)
		assert.Equal(t, TagSet([]string{"history", "go"}), note.Tags)
		assert.Equal(t, 8, note.Line)
		assert.Equal(t, markdown.Document("## Reference: Golang History\n\n`#history`\n\n`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n\n> Go was created in 2007"), note.Content)
		assert.Equal(t, "96ce446651b290347d7c1bd87d636da441c1b34a", note.Hash)
		assert.Equal(t, markdown.Document("`#history`\n\n`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007."), note.Body)
		assert.Equal(t, markdown.Document("Go was created in 2007"), note.Comment)
		assert.Equal(t, clock.Now(), note.CreatedAt)
		assert.Equal(t, clock.Now(), note.UpdatedAt)
		assert.Empty(t, note.DeletedAt)
		assert.Empty(t, note.LastCheckedAt)
	})

	t.Run("NewNote with parent", func(t *testing.T) {

		content := UnescapeTestContent(`
# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛
‛@tags: go‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

### Flashcard: Golang History

‛#study‛

(Go) **When** was created Go?

---

2007
`)
		root := SetUpRepositoryFromTempDir(t)
		err := os.WriteFile(filepath.Join(root, "go.md"), []byte(UnescapeTestContent(content)), 0644)
		require.NoError(t, err)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())

		// Init the notes
		parentParsedFile, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
		require.True(t, ok)
		parentNote, err := NewNote(file, nil, parentParsedFile)
		require.NoError(t, err)
		childParsedFile, ok := parsedFile.FindNoteByTitle("Flashcard: Golang History")
		require.True(t, ok)
		childNote, err := NewNote(file, parentNote, childParsedFile)
		require.NoError(t, err)

		// Check attributes
		assert.Equal(t, parentNote.OID, childNote.ParentNoteOID)
		assert.Equal(t, AttributeSet(map[string]any{
			"tags":  []string{"go", "study"},
			"title": "Golang History",
		}), childNote.Attributes)
		assert.ElementsMatch(t, []string{"go", "study"}, childNote.Tags.AsList())
	})

	t.Run("Save", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		FreezeNow(t)
		AssertNoNotes(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())

		// Save the note
		note, err := NewNote(file, nil, parsedFile.Notes[0])
		require.NoError(t, err)
		require.NoError(t, note.Save())

		require.Equal(t, 1, MustCountNotes(t))

		// Reread and check the note
		actual, err := CurrentRepository().FindNoteByWikilink("go#Reference: Golang History")
		require.NoError(t, err)
		assert.Equal(t, note.OID, note.OID)
		assert.Equal(t, note.Slug, actual.Slug)
		assert.Equal(t, note.FileOID, actual.FileOID)
		assert.Equal(t, note.ParentNoteOID, actual.ParentNoteOID)
		assert.Equal(t, note.NoteKind, actual.NoteKind)
		assert.Equal(t, note.Title, actual.Title)
		assert.Equal(t, note.LongTitle, actual.LongTitle)
		assert.Equal(t, note.ShortTitle, actual.ShortTitle)
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
		assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)
		assert.Empty(t, note.DeletedAt)
	})

	t.Run("NewOrExistingNote", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		AssertNoFiles(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())

		// Save the note
		parsedNote, ok := parsedFile.FindNoteByShortTitle("Golang Logo")
		require.True(t, ok)
		previousNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, previousNote.Save())

		// Edit the note text
		ReplaceLine(t, filepath.Join(root, "go.md"), 19,
			"What does the **Golang logo** represent?",
			"What is the **Golang logo**?")
		parsedFile, err = ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedNote, ok = parsedFile.FindNoteByShortTitle("Golang Logo")
		require.True(t, ok)
		newNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, newNote.Save())

		// Compare
		assert.Equal(t, previousNote.OID, newNote.OID) // Must have found the previous file
		assert.Contains(t, newNote.Body, "What is the **Golang logo**?")
	})

	t.Run("Update", func(t *testing.T) {
		root := SetUpRepositoryFromTempDir(t)
		c := FreezeNow(t)
		createdAt := c.Now()

		// First version
		content := `
# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.md"), []byte(UnescapeTestContent(content)), 0644))

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewOrExistingFile(parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())

		// Save the note
		parsedNote, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
		require.True(t, ok)
		createdNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, createdNote.Save())

		// Second version
		content = UnescapeTestContent(`# Go

## Reference: Golang History

‛#go‛ ‛@source: https://go.dev/blog/8years‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.

> Go is almost 20-year old
`)
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.md"), []byte(content), 0644))

		// Reread the file
		updatedAt := c.FastForward(10 * time.Minute)
		parsedFile, err = ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)

		// Update the note
		parsedNote, ok = parsedFile.FindNoteByTitle("Reference: Golang History")
		require.True(t, ok)
		updatedNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, updatedNote.Save())

		// Check all fields has been updated
		updatedNote, err = CurrentRepository().FindNoteByTitle("Reference: Golang History")
		require.NoError(t, err)
		// Some fields must not have changed
		assert.Equal(t, createdNote.OID, updatedNote.OID)
		assert.Equal(t, createdNote.RelativePath, updatedNote.RelativePath)
		assert.Equal(t, createdNote.Wikilink, updatedNote.Wikilink)
		// Some fields must have changed
		assert.NotEqual(t, createdNote.Content, updatedNote.Content)
		assert.NotEqual(t, createdNote.Body, updatedNote.Body)
		assert.NotEqual(t, createdNote.Comment, updatedNote.Comment)
		assert.NotEqual(t, createdNote.Tags, updatedNote.Tags)
		assert.ElementsMatch(t, []string{"go"}, updatedNote.Tags.AsList())
		assert.NotEqual(t, createdNote.Attributes, updatedNote.Attributes)
		assert.Equal(t, AttributeSet(map[string]any{
			"source": "https://go.dev/blog/8years",
			"tags":   []string{"go"},
			"title":  "Golang History",
		}), updatedNote.Attributes)
		assert.WithinDuration(t, createdAt, updatedNote.CreatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedNote.UpdatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedNote.LastCheckedAt, 1*time.Second)
	})

	t.Run("Delete", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		AssertNoNotes(t)
		FreezeNow(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())

		// Save the note
		parsedNote, ok := parsedFile.FindNoteByShortTitle("Golang Logo")
		require.True(t, ok)
		note, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Delete the note
		note.ForceState(Deleted)
		require.NoError(t, note.Save())

		assert.Equal(t, clock.Now(), note.DeletedAt)
		AssertNoNotes(t)
	})
}

func TestNoteFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`---
tags:
- go
---

# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`))

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)

	// Init the note
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)

	t.Run("ToYAML", func(t *testing.T) {
		actual := note.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
slug: go-reference-golang-history
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
parent_note_oid: ""
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
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := note.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "slug": "go-reference-golang-history",
  "file_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "parent_note_oid": "",
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
  "deleted_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := note.ToMarkdown()
		expected := UnescapeTestContent(`
# Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}

func TestSearchNotes(t *testing.T) {
	root := SetUpRepositoryFromGoldenDirNamed(t, "TestNoteFTS")

	CurrentLogger().SetVerboseLevel(VerboseTrace)

	// Insert the note
	parsedFile, err := ParseFileFromRelativePath(root, "note.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: FTS5")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Insert())

	// Search the note using a full-text query
	notes, err := CurrentRepository().SearchNotes("kind:reference fts5")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Update the note content
	note.updateContent("full-text")
	require.NoError(t, note.Update())

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

func TestFormatLongTitle(t *testing.T) {
	tests := []struct {
		name      string
		titles    []markdown.Document // input
		longTitle markdown.Document   // output
	}{
		{
			name:      "Basic",
			titles:    []markdown.Document{"Go", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Empty titles",
			titles:    []markdown.Document{"", "History"},
			longTitle: "History",
		},
		{
			name:      "Duplicate titles",
			titles:    []markdown.Document{"Go", "History", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Common prefix",
			titles:    []markdown.Document{"Go", "Go History"},
			longTitle: "Go History",
		},
		{
			name:      "Not common prefix",
			titles:    []markdown.Document{"Go", "Goroutines"},
			longTitle: "Go / Goroutines",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatLongTitle(tt.longTitle)
			assert.Equal(t, tt.longTitle, actual)
		})
	}
}
