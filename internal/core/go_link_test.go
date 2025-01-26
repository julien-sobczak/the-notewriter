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

func TestGoLink(t *testing.T) {
	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
# Go

## Reference: Golang History

[Golang](https://go.dev/doc/ "#go/go") was designed at Google in 2007.
`))

	UseSequenceOID(t)
	AssertNoGoLinks(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	note, err := NewNote(NilOID, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Save())

	// Create
	parsedGoLink, ok := parsedNote.FindGoLinkByGoName("go")
	require.True(t, ok)
	goLink := NewGoLink(NilOID, note, parsedGoLink)
	goLinkCopy := NewGoLink(NilOID, note, parsedGoLink)
	require.NotEqual(t, goLink.OID, goLinkCopy.OID)

	// Check all fields
	assert.NotNil(t, goLink.OID)
	assert.Equal(t, note.OID, goLink.NoteOID)
	assert.Equal(t, note.RelativePath, goLink.RelativePath)
	assert.Equal(t, "Golang", goLink.Text.String())
	assert.Equal(t, "https://go.dev/doc/", goLink.URL)
	assert.Equal(t, "", goLink.Title)
	assert.Equal(t, "go", goLink.GoName)
	assert.Equal(t, clock.Now(), goLink.CreatedAt)
	assert.Equal(t, clock.Now(), goLink.UpdatedAt)
	assert.Empty(t, goLink.DeletedAt)
	assert.Empty(t, goLink.LastIndexedAt)

	// Save
	require.NoError(t, goLink.Save())
	require.Equal(t, 1, MustCountGoLinks(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadGoLinkByOID(goLink.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, goLink.OID, actual.OID)
	assert.Equal(t, goLink.NoteOID, actual.NoteOID)
	assert.Equal(t, goLink.RelativePath, actual.RelativePath)
	assert.Equal(t, goLink.Text, actual.Text)
	assert.Equal(t, goLink.URL, actual.URL)
	assert.Equal(t, goLink.Title, actual.Title)
	assert.Equal(t, goLink.GoName, actual.GoName)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastIndexedAt, 1*time.Second)
	assert.Empty(t, actual.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	ReplaceLine(t, filepath.Join(root, "go.md"), 6,
		`[Golang](https://go.dev/doc/ "#go/go") was designed at Google in 2007.`,
		`[Go Language](https://go.dev "Developer Website #go/go") was designed at Google in 2007.`)
	parsedFile, err = ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedNote, ok = parsedFile.FindNoteByShortTitle("Golang History")
	require.True(t, ok)
	newNote, err := NewOrExistingNote(NilOID, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, newNote.Save())
	parsedGoLink, ok = parsedNote.FindGoLinkByGoName("go")
	require.True(t, ok)
	newGoLink, err := NewOrExistingGoLink(NilOID, newNote, parsedGoLink)
	require.NoError(t, err)
	require.NoError(t, newGoLink.Save())
	// ...and compare
	assert.Equal(t, goLink.OID, newGoLink.OID) // Must have found the previous one
	assert.Equal(t, "Go Language", newGoLink.Text.String())

	// Retrieve
	updatedGoLink, err := CurrentRepository().LoadGoLinkByOID(newGoLink.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedGoLink.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedGoLink.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedGoLink.LastIndexedAt, 1*time.Second)

	// Delete
	require.NoError(t, goLink.Delete())
	assert.Equal(t, clock.Now(), goLink.DeletedAt)

	AssertNoGoLinks(t)
}

func TestGoLinkFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
# Go

## Reference: Golang History

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`))

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)

	// Init the go link
	parsedNote, ok := parsedFile.FindNoteByTitle("Reference: Golang History")
	require.True(t, ok)
	note, err := NewNote(NilOID, file, parsedNote)
	require.NoError(t, err)
	parsedGoLink, ok := parsedNote.FindGoLinkByGoName("go")
	require.True(t, ok)
	goLink := NewGoLink(NilOID, note, parsedGoLink)

	t.Run("ToYAML", func(t *testing.T) {
		actual := goLink.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: go.md
text: Golang
url: https://go.dev/doc/
title: ""
go_name: go
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := goLink.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "note_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "relative_path": "go.md",
  "text": "Golang",
  "url": "https://go.dev/doc/",
  "title": "",
  "go_name": "go",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "deleted_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := goLink.ToMarkdown()
		expected := UnescapeTestContent(`[Golang](https://go.dev/doc/)`)
		assert.Equal(t, expected, actual)
	})

}
