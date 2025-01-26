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

func TestFile(t *testing.T) {
	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`---
tags:
- go
---

# Go

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

`))

	UseSequenceOID(t)
	AssertNoFiles(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)

	// Create
	file, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)
	fileCopy, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)
	require.NotEqual(t, file.OID, fileCopy.OID)

	// Check all fields
	assert.NotNil(t, file.OID)
	assert.Equal(t, "go", file.Slug)
	assert.Equal(t, "go.md", file.RelativePath)
	assert.Equal(t, "go", file.Wikilink)
	assert.Equal(t, markdown.FrontMatter("tags:\n- go\n"), file.FrontMatter)
	assert.Equal(t, AttributeSet(map[string]any{
		"tags": []string{"go"},
	}), file.Attributes)
	assert.Equal(t, markdown.Document("Go"), file.Title)
	assert.Equal(t, markdown.Document("Go"), file.ShortTitle)
	assert.True(t, strings.HasPrefix(file.Body.String(), "# Go"))
	assert.Equal(t, 6, file.BodyLine)
	assert.Equal(t, parsedFile.Markdown.Size, file.Size)
	assert.Equal(t, parsedFile.Markdown.MTime, file.MTime)
	assert.NotEqual(t, parsedFile.Markdown.Body.Hash(), file.Hash) // Must use whole content to determine the hash (including the front matter)
	assert.Equal(t, clock.Now(), file.CreatedAt)
	assert.Equal(t, clock.Now(), file.UpdatedAt)
	assert.Empty(t, file.DeletedAt)
	assert.Empty(t, file.LastIndexedAt)

	// Save
	require.NoError(t, file.Save())
	require.Equal(t, 1, MustCountFiles(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadFileByOID(file.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, file.OID, actual.OID)
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
	assert.Equal(t, file.MTime, actual.MTime)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastIndexedAt, 1*time.Second)
	assert.Empty(t, actual.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	ReplaceLine(t, filepath.Join(root, "go.md"), 19,
		"What does the **Golang logo** represent?",
		"What is the **Golang logo**?")

	// Recreate...
	parsedFile, err = ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	newFile, err := NewOrExistingFile(NilOID, parsedFile)
	require.NoError(t, err)
	require.NoError(t, newFile.Save())
	// ...and compare
	assert.Equal(t, file.OID, newFile.OID) // Must have found the previous one
	assert.Contains(t, newFile.Body, "What is the **Golang logo**?")

	// Retrieve
	updatedFile, err := CurrentRepository().LoadFileByOID(newFile.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedFile.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedFile.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedFile.LastIndexedAt, 1*time.Second)

	// Delete
	require.NoError(t, file.Delete())
	assert.Equal(t, clock.Now(), file.DeletedAt)

	AssertNoFiles(t)
}

func TestFileWithParent(t *testing.T) {
	parentContent := `---
tags:
- go
---`

	childContent := `---
tags: [programming]
---

# Go

## Reference: Golang History

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`
	root := SetUpRepositoryFromTempDir(t)
	err := os.WriteFile(filepath.Join(root, "index.md"), []byte(UnescapeTestContent(parentContent)), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(root, "go.md"), []byte(UnescapeTestContent(childContent)), 0644)
	require.NoError(t, err)

	// Init the parent
	mdParent := markdown.MustParseFile(filepath.Join(root, "index.md"))
	mdChild := markdown.MustParseFile(filepath.Join(root, "go.md"))

	parsedFile, err := ParseFile(root, mdChild, mdParent)
	require.NoError(t, err)
	childFile, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)

	assert.Equal(t, []string{"go", "programming"}, childFile.Attributes.Tags())
}

func TestFileFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`---
tags:
- go
---

# Go

## Reference: Golang History

‛@source: https://en.wikipedia.org/wiki/Go_(programming_language)‛

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
`))

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	file, err := NewFile(NilOID, parsedFile)
	require.NoError(t, err)
	file.MTime = clock.Now() // make tests reproductible

	t.Run("ToYAML", func(t *testing.T) {
		actual := file.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
slug: go
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
mode: 493
size: 243
hash: 45b9ee63ed13a69e2a3cf59afa26c672cacba78a
mtime: 2023-01-01T01:12:30Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := file.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "slug": "go",
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
  "mode": 493,
  "size": 243,
  "hash": "45b9ee63ed13a69e2a3cf59afa26c672cacba78a",
  "mtime": "2023-01-01T01:12:30Z",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "deleted_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := file.ToMarkdown()
		expected := UnescapeTestContent(`
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
