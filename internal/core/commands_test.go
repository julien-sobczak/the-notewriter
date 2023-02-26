package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		// Check index file
		idx, err := NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		changes := idx.CountChanges()
		require.Greater(t, changes, 0)
		require.Len(t, idx.Objects, 0) // Objects are only appended to index after a commit

		// Commit
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)

		// Check ref has been updated
		data, err := os.ReadFile(filepath.Join(root, ".nt/refs/main"))
		require.NoError(t, err)
		commitOID := string(data)

		// Check commit object is present
		c, err := CurrentDB().ReadCommit(commitOID)
		require.NoError(t, err)
		assert.Len(t, c.Objects, changes)

		// Check staging area is empty
		idx, err = NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		require.Equal(t, 0, idx.CountChanges())
	})

}
