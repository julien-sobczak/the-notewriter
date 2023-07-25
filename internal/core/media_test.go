package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestMedia(t *testing.T) {

	t.Run("YAML", func(t *testing.T) {

		// Make tests reproductible
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		// Set up a collection
		mediaSrc := NewMedia("medias/go.svg")
		mediaSrc.MTime = clock.Now()
		// Force blobs generation to check the whole model
		mediaSrc.UpdateBlobs()

		// Marshall
		buf := new(bytes.Buffer)
		err := mediaSrc.Write(buf)
		require.NoError(t, err)
		mediaYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: medias/go.svg
kind: picture
dangling: false
extension: .svg
mtime: 2023-01-01T01:12:30Z
hash: 0cd82f33352563c9cf918d9f4fa0504cc6b84526
size: 2288
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
`), strings.TrimSpace(mediaYAML))

		// Unmarshall
		mediaDest := new(Media)
		err = mediaDest.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, cleanMedia(mediaSrc), cleanMedia(mediaDest))
	})

}

/* Test Helpers */

// cleanMedia ignore some values as EqualValues is very strict.
func cleanMedia(m *Media) *Media {
	// Do not compare state management attributes
	m.new = false
	m.stale = false
	for _, b := range m.BlobRefs {
		if b.Attributes != nil && len(b.Attributes) == 0 {
			b.Attributes = nil
		}
	}
	return m
}
