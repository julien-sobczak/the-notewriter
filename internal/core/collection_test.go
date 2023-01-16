package core

import (
	"path/filepath"
	"testing"

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
