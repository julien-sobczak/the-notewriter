package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandAdd(t *testing.T) {

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

		// Check some files are not created before the first commit
		assert.NoFileExists(t, filepath.Join(root, ".nt/refs/main"))
		assert.NoFileExists(t, filepath.Join(root, ".nt/objects/info/commit-graph"))

		// Commit
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)

		// Check main ref has been updated
		data, err := os.ReadFile(filepath.Join(root, ".nt/refs/main"))
		require.NoError(t, err)
		commitOID := string(data)

		// Check commit object is present
		c, err := CurrentDB().ReadCommit(commitOID)
		require.NoError(t, err)
		assert.Len(t, c.Objects, changes)

		// Check commit graph was updated
		data, err = os.ReadFile(filepath.Join(root, ".nt/objects/info/commit-graph"))
		require.NoError(t, err)
		commitGraph := string(data)
		assert.Contains(t, commitGraph, c.OID)

		// Check staging area is empty
		idx, err = NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		require.Equal(t, 0, idx.CountChanges())
	})

	t.Run("Add Media", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave") // Use a new directory with different medias

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		// Check blobs are present
		media, err := FindMediaByRelativePath("medias/go.svg")
		require.NoError(t, err)
		require.NotNil(t, media)
		for _, blob := range media.Blobs() {
			oid := blob.OID
			assert.NoFileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(oid)))
		}

		// TODO julien

	})

}

func TestCommandRestore(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		CurrentLogger().SetVerboseLevel(VerboseDebug)

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		// Check index file
		idx, err := NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		changes := idx.CountChanges()
		require.Greater(t, changes, 0)
		require.Len(t, idx.Objects, 0)

		// Check database
		file, err := LoadFileByPath("go.md")
		require.NoError(t, err)
		require.NotEqual(t, 0, file.MTime)

		// Restore
		err = CurrentDB().Restore()
		require.NoError(t, err)

		// Check staging area is empty
		idx, err = NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		require.Equal(t, 0, idx.CountChanges())

		// Check database is empty
		file, err = LoadFileByPath("go.md")
		require.NoError(t, err)
		require.Nil(t, file)
	})

}

func TestCommandCommit(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)

		newFilepath := filepath.Join(root, "python.md")
		err = os.WriteFile(newFilepath, []byte(`# Python

## Flashcard: Python's creator

Who invented Python?
---
Guido van Rossum
`), 0644)
		require.NoError(t, err)

		refBefore, _ := CurrentDB().Ref("main")
		err = CurrentDB().Commit("empty commit")
		require.ErrorContains(t, err, "nothing to commit")

		// Check no commit were created
		refAfter, _ := CurrentDB().Ref("main")
		require.Equal(t, refBefore, refAfter)

		// Create a second commit
		CurrentLogger().SetVerboseLevel(VerboseDebug) // FIXME remove
		err = CurrentCollection().Add("python.md")
		require.NoError(t, err)

		err = CurrentDB().Commit("second commit")
		require.NoError(t, err)
		refAfter, _ = CurrentDB().Ref("main")
		require.NotEqual(t, refBefore, refAfter)
	})

}

func TestCommandPushPull(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")
		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Push
		err := CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)
		head, ok := CurrentDB().Ref("origin")
		require.True(t, ok)

		// Check origin
		require.FileExists(t, filepath.Join(origin, "info/commit-graph"))
		require.FileExists(t, filepath.Join(origin, OIDToPath(head)))

		Reset()

		// Pull from a new repository
		root := SetUpCollectionFromTempDir(t)
		// Configure same origin
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}
		err = CurrentDB().Pull()
		require.NoError(t, err)

		// Check local
		require.FileExists(t, filepath.Join(root, ".nt/objects/info/commit-graph"))
		require.FileExists(t, filepath.Join(root, ".nt/objects", OIDToPath(head)))
	})

}