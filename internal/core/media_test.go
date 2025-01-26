package core

import (
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
	root := SetUpRepositoryFromFileContent(t, "go.md", UnescapeTestContent(`
## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./go.svg)
`))

	UseSequenceOID(t)
	AssertNoMedias(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)

	// Create
	parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
	require.True(t, ok)
	media, err := NewMedia(NilOID, parsedMedia)
	require.NoError(t, err)
	mediaCopy, err := NewMedia(NilOID, parsedMedia)
	require.NoError(t, err)
	require.NotEqual(t, media.OID, mediaCopy.OID)

	// Check all fields
	assert.Equal(t, "0000000000000000000000000000000000000001", media.OID)
	assert.Equal(t, "go.svg", media.RelativePath)
	assert.Equal(t, KindPicture, media.MediaKind)
	assert.Equal(t, true, media.Dangling)
	assert.Equal(t, ".svg", media.Extension)
	assert.Empty(t, "", media.MTime)
	assert.Equal(t, "", media.Hash)
	assert.Equal(t, int64(0), media.Size)
	assert.Equal(t, clock.Now(), media.CreatedAt)
	assert.Equal(t, clock.Now(), media.UpdatedAt)
	assert.Empty(t, media.DeletedAt)
	assert.Empty(t, media.LastIndexedAt)

	// Save
	require.NoError(t, media.Save())
	require.Equal(t, 1, MustCountMedias(t))

	// Reread and recheck all fields
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
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastIndexedAt, 1*time.Second)
	assert.Empty(t, actual.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	touch(t, "go.svg")

	// Recreate...
	parsedFile, err = ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedMedia, ok = parsedFile.FindMediaByFilename("go.svg")
	require.True(t, ok)
	newMedia, err := NewOrExistingMedia(NilOID, parsedMedia)
	require.NoError(t, err)
	require.Equal(t, media.OID, newMedia.OID)
	require.NoError(t, newMedia.Save())
	// ...and compare
	assert.Equal(t, media.OID, newMedia.OID) // Must have found the previous one
	assert.False(t, newMedia.Dangling)
	assert.NotEqual(t, media.MTime, newMedia.MTime)
	assert.NotEqual(t, media.Hash, newMedia.Hash)

	// Retrieve
	updatedMedia, err := CurrentRepository().LoadMediaByOID(newMedia.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedMedia.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedMedia.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedMedia.LastIndexedAt, 1*time.Second)

	// Delete
	require.NoError(t, media.Delete())
	assert.Equal(t, clock.Now(), media.DeletedAt)

	AssertNoReminders(t)
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
	touch(t, "go.svg")
	parsedFile, err := ParseFileFromRelativePath(root, "go.md")
	require.NoError(t, err)
	parsedMedia, ok := parsedFile.FindMediaByFilename("go.svg")
	require.True(t, ok)
	media, err := NewMedia(NilOID, parsedMedia)
	require.NoError(t, err)

	// Force blobs generation to check the whole model
	media.GenerateBlobs()
	media.MTime = clock.Now() // make test reproductible

	t.Run("ToYAML", func(t *testing.T) {
		actual := media.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: go.svg
kind: picture
dangling: false
extension: .svg
mtime: 2023-01-01T01:12:30Z
hash: da39a3ee5e6b4b0d3255bfef95601890afd80709
size: 0
mode: 420
blobs:
  - oid: cc79c943c616af40bfbaf88b061603985d811210
    mime: image/avif
    attributes: {}
    tags:
      - preview
      - lossy
  - oid: 8a3343b1b444b671ced4acd9201949a0116c6e81
    mime: image/avif
    attributes: {}
    tags:
      - large
      - lossy
  - oid: 98958cb47ae1bcb5f8f4e5a04af170ed6ef41c5e
    mime: image/avif
    attributes: {}
    tags:
      - original
      - lossy
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
  "dangling": false,
  "extension": ".svg",
  "mtime": "2023-01-01T01:12:30Z",
  "hash": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
  "size": 0,
  "mode": 420,
  "blobs": [
    {
      "oid": "cc79c943c616af40bfbaf88b061603985d811210",
      "mime": "image/avif",
      "attributes": null,
      "tags": [
        "preview",
        "lossy"
      ]
    },
    {
      "oid": "8a3343b1b444b671ced4acd9201949a0116c6e81",
      "mime": "image/avif",
      "attributes": null,
      "tags": [
        "large",
        "lossy"
      ]
    },
    {
      "oid": "98958cb47ae1bcb5f8f4e5a04af170ed6ef41c5e",
      "mime": "image/avif",
      "attributes": null,
      "tags": [
        "original",
        "lossy"
      ]
    }
  ],
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

func TestDetectMediaKind(t *testing.T) {
	var tests = []struct {
		name     string    // name
		filename string    // input
		kind     MediaKind // output
	}{
		{
			name:     "Directory",
			filename: "./test/",
			kind:     KindUnknown,
		},
		{
			name:     "Picture",
			filename: "pic.jpeg",
			kind:     KindPicture,
		},
		{
			name:     "Audio",
			filename: "case.mp3",
			kind:     KindAudio,
		},
		{
			name:     "Video",
			filename: "funny.webm",
			kind:     KindVideo,
		},
		{
			name:     "Case insensitive",
			filename: "case.PNG",
			kind:     KindPicture,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := DetectMediaKind(tt.filename)
			assert.Equal(t, tt.kind, actual)
		})
	}
}

/* Test Helpers */

// touch creates an empty file if not existing.
func touch(t *testing.T, relativePath string) {
	_, err := os.OpenFile(filepath.Join(CurrentConfig().RootDirectory, relativePath), os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)
}
