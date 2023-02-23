package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
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
	// Make tests reproductible
	UseFixedOID("42d74d967d9b4e989502647ac510777ca1e22f4a")
	defer ResetOID()
	clock.FreezeAt(time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
	defer clock.Unfreeze()
	SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

	t.Run("YAML", func(t *testing.T) {
		// Set up a collection
		mediaSrc := NewMedia("medias/go.svg")
		mediaSrc.MTime = clock.Now()

		// Marshall
		buf := new(bytes.Buffer)
		err := mediaSrc.Write(buf)
		require.NoError(t, err)
		mediaYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: medias/go.svg
kind: 2
dangling: false
links: 0
extension: .svg
mtime: 2023-01-01T01:12:30Z
hash: 974a75814a1339c82cb497ea1ab56383
size: 2288
mode: 420
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(mediaYAML))

		// Unmarshall
		mediaDest := new(Media)
		err = mediaDest.Read(buf)
		require.NoError(t, err)
		mediaSrc.new = false
		mediaSrc.stale = false
		assert.EqualValues(t, mediaSrc, mediaDest)
	})

}
