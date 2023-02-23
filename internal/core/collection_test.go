package core

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionGetRelativePath(t *testing.T) {
	var tests = []struct {
		name                   string // name
		referencePath          string // input
		noteRelativePath       string // input
		collectionRelativePath string // output
	}{
		{
			name:                   "Same directory",
			referencePath:          "./projects/the-notetaker/todo.md",
			noteRelativePath:       "ideas.md",
			collectionRelativePath: "projects/the-notetaker/ideas.md",
		},
		{
			name:                   "Medias file",
			referencePath:          "./skills/programming.md",
			noteRelativePath:       "./medias/go.svg",
			collectionRelativePath: "skills/medias/go.svg",
		},
		{
			name:                   "Move to parent directory",
			referencePath:          "./projects/the-notetaker/todo.md",
			noteRelativePath:       "../../skills/programming.md",
			collectionRelativePath: "skills/programming.md",
		},
	}

	dirname := SetUpCollectionFromGoldenDir(t)
	require.Equal(t, dirname, CurrentConfig().RootDirectory)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relpath, err := CurrentCollection().GetNoteRelativePath(filepath.Join(dirname, tt.referencePath), tt.noteRelativePath)
			require.NoError(t, err)
			assert.Equal(t, tt.collectionRelativePath, relpath)
		})
	}
}

func TestCollection(t *testing.T) {
	// Make tests reproductible
	UseFixedOID("42d74d967d9b4e989502647ac510777ca1e22f4a")
	defer ResetOID()
	clock.FreezeAt(time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
	defer clock.Unfreeze()
	dirname := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

	t.Run("YAML", func(t *testing.T) {
		collectionSrc, err := NewCollection(nil, nil)
		require.NoError(t, err)

		// Marshall
		buf := new(bytes.Buffer)
		err = collectionSrc.Write(buf)
		require.NoError(t, err)
		collectionYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
path: `+dirname+`
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(collectionYAML))

		// Unmarshall
		collectionDest := new(Collection)
		err = collectionDest.Read(buf)
		require.NoError(t, err)
		collectionSrc.new = false
		assert.EqualValues(t, collectionSrc, collectionDest)
	})

}
