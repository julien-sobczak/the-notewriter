package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMedia(t *testing.T) {

	content := `
## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./go.svg)
`

	t.Run("NewMedia", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		media, err := NewMedia(parsedMedia)
		require.NoError(t, err)

		// Check all fields
		assert.Equal(t, "42d74d967d9b4e989502647ac510777ca1e22f4a", media.OID)
		assert.Equal(t, "go.svg", media.RelativePath)
		assert.Equal(t, KindPicture, media.MediaKind)
		assert.Equal(t, true, media.Dangling)
		assert.Equal(t, ".svg", media.Extension)
		assert.Empty(t, "", media.MTime)
		assert.Equal(t, "", media.Hash)
		assert.Equal(t, int64(0), media.Size)
		assert.Equal(t, fs.FileMode(0x0), media.Mode)
		assert.Equal(t, clock.Now(), media.CreatedAt)
		assert.Equal(t, clock.Now(), media.UpdatedAt)
		assert.Empty(t, media.DeletedAt)
		assert.Empty(t, media.LastCheckedAt)
	})

	t.Run("Save", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		AssertNoMedias(t)

		// Save the media
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		media, err := NewMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, media.Save())

		require.Equal(t, 1, MustCountMedias(t))

		// Reread and check the media
		actual, err := CurrentRepository().LoadMediaByOID(media.OID)
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Equal(t, media.OID, actual.OID)
		assert.Equal(t, media.RelativePath, actual.RelativePath)
		assert.Equal(t, media.MediaKind, actual.MediaKind)
		assert.Equal(t, media.Extension, actual.Extension)
		assert.Equal(t, media.Dangling, actual.Dangling)
		assert.Equal(t, media.MTime, actual.MTime)
		assert.Equal(t, media.Hash, actual.Hash)
		assert.Equal(t, media.Size, actual.Size)
		assert.Equal(t, media.Mode, actual.Mode)
		assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
		assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
		assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)
		assert.Empty(t, actual.DeletedAt)
	})

	t.Run("NewOrExistingMedia", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		AssertNoMedias(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		previousMedia, err := NewMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, previousMedia.Save())

		// Create the file on disk
		_, err = os.OpenFile(filepath.Join(root, "go.svg"), os.O_RDONLY|os.O_CREATE, 0666)
		require.NoError(t, err)
		parsedFile, err = ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok = parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		newMedia, err := NewOrExistingMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, newMedia.Save())

		// Compare
		assert.Equal(t, previousMedia.OID, newMedia.OID) // Must have found the previous one
		assert.False(t, newMedia.Dangling)
		assert.NotEqual(t, previousMedia.MTime, newMedia.MTime)
		assert.NotEqual(t, previousMedia.Hash, newMedia.Hash)
		assert.NotEqual(t, previousMedia.Mode, newMedia.Mode)
	})

	t.Run("Update", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		c := FreezeNow(t)
		createdAt := c.Now()
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the media
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		createdMedia, err := NewMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, createdMedia.Save())

		// Edit the media file
		updatedAt := c.FastForward(10 * time.Minute)
		// Create the file on disk
		_, err = os.OpenFile(filepath.Join(root, "go.svg"), os.O_RDONLY|os.O_CREATE, 0666)
		require.NoError(t, err)
		parsedFile, err = ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok = parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		updatedMedia, err := NewOrExistingMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, updatedMedia.Save())

		// Check all fields has been updated
		updatedMedia, err = CurrentRepository().LoadMediaByOID(updatedMedia.OID)
		require.NoError(t, err)
		// Some fields must not have changed
		assert.Equal(t, createdMedia.OID, updatedMedia.OID)
		assert.Equal(t, createdMedia.RelativePath, updatedMedia.RelativePath)
		// Some fields must have changed
		assert.WithinDuration(t, createdAt, updatedMedia.CreatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedMedia.UpdatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedMedia.LastCheckedAt, 1*time.Second)
	})

	t.Run("Delete", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Save the media
		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)
		parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
		require.True(t, ok)
		media, err := NewMedia(parsedMedia)
		require.NoError(t, err)
		require.NoError(t, media.Save())

		// Delete the reminder
		require.NoError(t, media.Delete())

		assert.Equal(t, clock.Now(), media.DeletedAt)
		AssertNoReminders(t)
	})
}

func TestMediaFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./go.svg)
`))

	// Init the media
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
	require.True(t, ok)
	media, err := NewMedia(parsedMedia)
	require.NoError(t, err)

	t.Run("ToYAML", func(t *testing.T) {
		actual := media.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: go.svg
kind: picture
dangling: true
extension: .svg
mtime: 0001-01-01T00:00:00Z
hash: ""
size: 0
mode: 0
blobs: []
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := media.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "relative_path": "go.svg",
  "kind": "picture",
  "dangling": true,
  "extension": ".svg",
  "mtime": "0001-01-01T00:00:00Z",
  "hash": "",
  "size": 0,
  "mode": 0,
  "blobs": null,
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "deleted_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := media.ToMarkdown()
		expected := "![](go.svg)"
		assert.Equal(t, expected, actual)
	})

}
