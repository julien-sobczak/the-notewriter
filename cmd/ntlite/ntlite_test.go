package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLite(t *testing.T) {
	root := core.SetUpRepositoryFromGoldenDir(t)

	// Check input file
	require.FileExists(t, filepath.Join(root, "notes.md"))
	// Same as
	require.FileExists(t, filepath.Join(CurrentRepository().Path, "notes.md"))

	// nt add
	err := CurrentRepository().Add()
	require.NoError(t, err)
	// Check index
	index := ReadIndex()
	assert.Empty(t, index.Objects, 0)
	assert.Len(t, index.StagingArea, 3) // 1 file + 2 notes

	// nt commit
	err = CurrentDB().Commit()
	require.NoError(t, err)
	// Check index
	index = ReadIndex()
	assert.Len(t, index.Objects, 3)
	assert.Empty(t, index.StagingArea)

	// Check relational database
	note1, err := CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 1")
	require.NoError(t, err)
	require.NotNil(t, note1)
	note2, err := CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 2")
	require.NoError(t, err)
	require.NotNil(t, note2)

	assert.Equal(t, "A first note.", note1.Content)
	assert.Equal(t, "A second note.", note2.Content)

	// Check object database
	require.FileExists(t, filepath.Join(CurrentRepository().Path, ".nt/index"))
	require.FileExists(t, filepath.Join(CurrentRepository().Path, ".nt/objects", OIDToPath(note1.OID)))
	require.FileExists(t, filepath.Join(CurrentRepository().Path, ".nt/objects", OIDToPath(note2.OID)))
	// Check a single object
	data, err := os.ReadFile(filepath.Join(CurrentRepository().Path, ".nt/objects", OIDToPath(note1.OID)))
	require.NoError(t, err)
	assert.Contains(t, string(data), "title: 'Note: Example 1")
}
