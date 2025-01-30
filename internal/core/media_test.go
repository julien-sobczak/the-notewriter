package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMedia(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)

	media := &Media{
		OID:           "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:   "a1ea23ae1287416c8796a0981f558206690b8a76",
		RelativePath:  "go.svg",
		MediaKind:     KindPicture,
		Dangling:      true,
		Extension:     ".svg",
		MTime:         time.Time{},
		Hash:          "",
		Size:          0,
		CreatedAt:     clock.Now(),
		UpdatedAt:     clock.Now(),
		LastIndexedAt: clock.Now(),
	}

	// Save
	require.NoError(t, media.Save())
	require.Equal(t, 1, MustCountMedias(t))

	// Check
	actual, err := CurrentRepository().LoadMediaByOID(media.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, media.OID, actual.OID)
	assert.Equal(t, media.PackFileOID, actual.PackFileOID)
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

	// Update
	actual.Dangling = false
	actual.Hash = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	actual.MTime = clock.Now()
	actual.Size = 42
	require.NoError(t, actual.Save())
	require.Equal(t, 1, MustCountMedias(t))

	// Check again
	actual, err = CurrentRepository().LoadMediaByOID(media.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, media.OID, actual.OID) // Must have found the previous one
	assert.False(t, actual.Dangling)
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", actual.Hash)
	assert.Equal(t, int64(42), actual.Size)

	// Delete
	require.NoError(t, media.Delete())
	AssertNoReminders(t)
}

func TestMediaFormats(t *testing.T) {
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	media := &Media{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "a1ea23ae1287416c8796a0981f558206690b8a76",
		RelativePath: "go.svg",
		MediaKind:    KindPicture,
		Dangling:     false,
		Extension:    ".svg",
		MTime:        clock.Now(),
		Hash:         "a960552f37b945e79b5add3ee0b99312724aaa41",
		Size:         42,
		BlobRefs: []*BlobRef{
			{
				OID:      "cc79c943c616af40bfbaf88b061603985d811210",
				MimeType: "image/avif",
				Tags:     []string{"preview", "lossy"},
			},
			{
				OID:      "8a3343b1b444b671ced4acd9201949a0116c6e81",
				MimeType: "image/avif",
				Tags:     []string{"large", "lossy"},
			},
			{
				OID:      "98958cb47ae1bcb5f8f4e5a04af170ed6ef41c5e",
				MimeType: "image/avif",
				Tags:     []string{"original", "lossy"},
			},
		},
		CreatedAt:     clock.Now(),
		UpdatedAt:     clock.Now(),
		LastIndexedAt: clock.Now(),
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := media.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
packfile_oid: a1ea23ae1287416c8796a0981f558206690b8a76
relative_path: go.svg
kind: picture
dangling: false
extension: .svg
mtime: 2023-01-01T01:12:30Z
hash: a960552f37b945e79b5add3ee0b99312724aaa41
size: 42
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
last_indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := media.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "packfile_oid": "a1ea23ae1287416c8796a0981f558206690b8a76",
  "relative_path": "go.svg",
  "kind": "picture",
  "dangling": false,
  "extension": ".svg",
  "mtime": "2023-01-01T01:12:30Z",
  "hash": "a960552f37b945e79b5add3ee0b99312724aaa41",
  "size": 42,
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
  "last_indexed_at": "2023-01-01T01:12:30Z"
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
