package main

import (
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
	require.Len(t, index.Entries, 1) // 1 pack file
	uniqueEntry := index.Entries[0]
	assert.True(t, uniqueEntry.Staged)
	assert.FileExists(t, filepath.Join(CurrentRepository().Path, ".nt/objects", OIDToPath(uniqueEntry.PackFileOID)))

	// Check relational database
	note1, err := CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 1")
	require.NoError(t, err)
	require.NotNil(t, note1)
	note2, err := CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 2")
	require.NoError(t, err)
	require.NotNil(t, note2)

	// nt commit
	err = CurrentRepository().Commit()
	require.NoError(t, err)
	// Check index
	index = ReadIndex()
	assert.Len(t, index.Entries, 1) // Still the same pack file
	uniqueEntry = index.Entries[0]
	assert.False(t, uniqueEntry.Staged)

	// Check relational database again
	note1, err = CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 1")
	require.NoError(t, err)
	require.NotNil(t, note1)
	note2, err = CurrentRepository().FindNoteByTitle("notes.md", "Note: Example 2")
	require.NoError(t, err)
	require.NotNil(t, note2)

	assert.Equal(t, "## Note: Example 1\n\nA first note.", note1.Content)
	assert.Equal(t, "## Note: Example 2\n\nA second note.", note2.Content)

	// Check object database
	require.FileExists(t, filepath.Join(CurrentRepository().Path, ".nt/index"))
}
