package core

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestIndexFilesFirst(t *testing.T) {
	var tests = []struct {
		name   string
		paths  []string // input
		result []string // output
	}{

		{
			name: "Without index file",
			paths: []string{
				"projects/A/features.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"projects/A/features.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With index.md",
			paths: []string{
				"todo/do.md",
				"todo/index.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"todo/index.md", // UP
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With INDEX.markdown",
			paths: []string{
				"todo/do.md",
				"todo/INDEX.markdown",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"todo/INDEX.markdown", // UP
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With mulitple index.md",
			paths: []string{
				"appendix.md",
				"index.md",
				"references/books/a.md",
				"references/books/index.md",
				"references/index.md",
				"todo/do.md",
				"todo/index.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"index.md",
				"appendix.md",
				"references/books/index.md",
				"references/books/a.md",
				"references/index.md",
				"todo/index.md",
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slices.SortFunc(tt.paths, IndexFilesFirst)
			assert.Equal(t, tt.result, tt.paths)
		})
	}
}

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

	root := SetUpCollectionFromGoldenDir(t)
	require.Equal(t, root, CurrentConfig().RootDirectory)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relpath, err := CurrentCollection().GetNoteRelativePath(filepath.Join(root, tt.referencePath), tt.noteRelativePath)
			require.NoError(t, err)
			assert.Equal(t, tt.collectionRelativePath, relpath)
		})
	}
}
