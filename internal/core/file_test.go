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

func TestFile(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)

	AssertNoFiles(t)

	createdAt := clock.Now()
	file := &File{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		Slug:         "go",
		RelativePath: "go.md",
		Wikilink:     "go",
		FrontMatter:  markdown.FrontMatter("tags:\n- go\n"),
		Attributes: AttributeSet(map[string]any{
			"tags": []string{"go"},
		}),
		Title:      markdown.Document("Go"),
		ShortTitle: markdown.Document("Go"),
		Body: markdown.Document(text.UnescapeTestContent(`# Go

## Reference: Golang History

‛#history‛

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.


## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./medias/go.svg)


## TODO: Conferences

* [Gophercon Europe](https://gophercon.eu/) ‛#reminder-2023-06-26‛

`)),
		BodyLine:  6,
		Size:      243,
		Hash:      "45b9ee63ed13a69e2a3cf59afa26c672cacba78a",
		MTime:     createdAt,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		IndexedAt: createdAt,
	}

	// Create
	require.NoError(t, file.Save())
	require.Equal(t, 1, MustCountFiles(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadFileByOID(file.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, file.OID, actual.OID)
	assert.Equal(t, file.PackFileOID, actual.PackFileOID)
	assert.Equal(t, file.RelativePath, actual.RelativePath)
	assert.Equal(t, file.Wikilink, actual.Wikilink)
	expectedFrontMatter, err := file.FrontMatter.AsBeautifulYAML()
	assert.NoError(t, err)
	actualFrontMatter, err := actual.FrontMatter.AsBeautifulYAML()
	assert.NoError(t, err)
	assert.Equal(t, expectedFrontMatter, actualFrontMatter)
	assert.Equal(t, file.Attributes.Tags(), actual.Attributes.Tags())
	assert.Equal(t, file.Body, actual.Body)
	assert.Equal(t, file.BodyLine, actual.BodyLine)
	assert.Equal(t, file.Size, actual.Size)
	assert.Equal(t, file.Hash, actual.Hash)
	assert.WithinDuration(t, file.MTime, actual.MTime, 1*time.Second)
	assert.WithinDuration(t, createdAt, actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, createdAt, actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, createdAt, actual.IndexedAt, 1*time.Second)

	// Force update
	actual.Title = markdown.Document("Golang")
	require.NoError(t, actual.Save())
	require.Equal(t, 1, MustCountFiles(t))

	// Recheck
	actual, err = CurrentRepository().LoadFileByOID(file.OID)
	require.NoError(t, err)
	assert.Equal(t, "Golang", actual.Title.String())

	// Delete
	require.NoError(t, actual.Delete())
	AssertNoFiles(t)
}

func TestFileFormats(t *testing.T) {
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	createdAt := clock.Now()
	file := &File{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		Slug:         "go",
		RelativePath: "go.md",
		Wikilink:     "go",
		FrontMatter:  markdown.FrontMatter("tags:\n- go\n"),
		Attributes: AttributeSet(map[string]any{
			"tags": []string{"go"},
		}),
		Title:      markdown.Document("Go"),
		ShortTitle: markdown.Document("Go"),
		Body: markdown.Document(text.UnescapeTestContent(`# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`)),
		BodyLine:  6,
		Size:      243,
		Hash:      "45b9ee63ed13a69e2a3cf59afa26c672cacba78a",
		MTime:     createdAt,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		IndexedAt: createdAt,
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := file.ToYAML()

		expected := text.UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
slug: go
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
relative_path: go.md
wikilink: go
front_matter: |
  tags:
  - go
attributes:
  tags:
    - go
title: Go
short_title: Go
body: |
  # Go

  ## Reference: Golang History

  ‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

  [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
body_line: 6
size: 243
hash: 45b9ee63ed13a69e2a3cf59afa26c672cacba78a
mtime: 2023-01-01T01:12:30Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := file.ToJSON()
		expected := text.UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "slug": "go",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "relative_path": "go.md",
  "wikilink": "go",
  "front_matter": "tags:\n- go\n",
  "attributes": {
    "tags": [
      "go"
    ]
  },
  "title": "Go",
  "short_title": "Go",
  "body": "# Go\n\n## Reference: Golang History\n\n‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n",
  "body_line": 6,
  "size": 243,
  "hash": "45b9ee63ed13a69e2a3cf59afa26c672cacba78a",
  "mtime": "2023-01-01T01:12:30Z",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := file.ToMarkdown()
		expected := text.UnescapeTestContent(`
---
tags:
- go
---

# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

}
