package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoLink(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)

	AssertNoGoLinks(t)

	createdAt := clock.Now()
	goLink := &GoLink{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "project.md",
		Text:         "Golang",
		URL:          "https://go.dev/doc/",
		Title:        "",
		GoName:       "go",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		IndexedAt:    createdAt,
	}

	// Save
	require.NoError(t, goLink.Save())
	require.Equal(t, 1, MustCountGoLinks(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadGoLinkByOID(goLink.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, goLink.OID, actual.OID)
	assert.Equal(t, goLink.PackFileOID, actual.PackFileOID)
	assert.Equal(t, goLink.NoteOID, actual.NoteOID)
	assert.Equal(t, goLink.RelativePath, actual.RelativePath)
	assert.Equal(t, goLink.Text, actual.Text)
	assert.Equal(t, goLink.URL, actual.URL)
	assert.Equal(t, goLink.Title, actual.Title)
	assert.Equal(t, goLink.GoName, actual.GoName)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.IndexedAt, 1*time.Second)

	// Force update
	goLink.Text = "Go Language"
	goLink.URL = "https://go.dev"
	require.NoError(t, goLink.Save())
	require.Equal(t, 1, MustCountGoLinks(t))

	// ...and compare again
	actual, err = CurrentRepository().LoadGoLinkByOID(goLink.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"), actual.OID) // Must have found the previous one
	assert.Equal(t, "Go Language", actual.Text.String())
	assert.Equal(t, "https://go.dev", actual.URL)

	// Delete
	require.NoError(t, goLink.Delete())
	AssertNoGoLinks(t)
}

func TestGoLinkFormats(t *testing.T) {
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	goLink := &GoLink{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "go.md",
		Text:         "Golang",
		URL:          "https://go.dev/doc/",
		Title:        "",
		GoName:       "go",
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		IndexedAt:    clock.Now(),
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := goLink.ToYAML()

		expected := text.UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
note_oid: 52d02a28a961471db62c6d40d30639dafe4aba00
relative_path: go.md
text: Golang
url: https://go.dev/doc/
title: ""
go_name: go
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := goLink.ToJSON()
		expected := text.UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "note_oid": "52d02a28a961471db62c6d40d30639dafe4aba00",
  "relative_path": "go.md",
  "text": "Golang",
  "url": "https://go.dev/doc/",
  "title": "",
  "go_name": "go",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := goLink.ToMarkdown()
		expected := text.UnescapeTestContent(`[Golang](https://go.dev/doc/)`)
		assert.Equal(t, expected, actual)
	})

}
