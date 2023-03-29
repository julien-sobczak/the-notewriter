package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLint(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromTempDir(t)
		err := os.WriteFile(filepath.Join(root, ".nt/lint"), []byte(`
rules:
- name: no-duplicate-note-title
`), 0644)
		require.NoError(t, err)
		configOnce.Reset()

		// Create a file violating the rule
		err = os.WriteFile(filepath.Join(root, "lint.md"), []byte(`
# Linter

## Note: Name

This is a first note

## Note: Name

This is a second note
`), 0644)
		require.NoError(t, err)

		result, err := CurrentCollection().Lint(".")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.AnalyzedFiles)
		require.Equal(t, 1, result.AffectedFiles)
		require.Len(t, result.Errors, 1)
		violation := result.Errors[0]
		assert.Equal(t, "duplicated note with title \"Name\"", violation.Message)
	})

}

func TestCommandAdd(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMedias")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		// Check referenced blobs are present
		referencedMedias := []string{
			// audios.md
			"medias/waterfall.flac",
			"medias/thunderstorm.wav",
			"medias/rain.flac",
			"medias/water.mp3",
			// pictures.md
			"medias/branch-portrait-small.jpg",
			"medias/branch-portrait-original.jpg",
			"medias/bird-landscape-large.png",
			"medias/earth-landscape-large.gif",
			"medias/flower-portrait.avif",
			// videos.md
			"medias/forest-large.mp4",
			"medias/forest-large.webm",
			"medias/aurora.avi",
			"medias/aurora.mp4",
		}
		for _, expectedMedia := range referencedMedias {
			media, err := CurrentCollection().FindMediaByRelativePath(expectedMedia)
			require.NoError(t, err)
			require.NotNil(t, media)
			for _, blob := range media.Blobs() {
				oid := blob.OID
				assert.FileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(oid)))
			}
		}

		// Check non-referenced blobs are missing
		unreferencedMedias := []string{
			"medias/branch-portrait.avif",
		}
		for _, unreferencedMedia := range unreferencedMedias {
			media, err := CurrentCollection().FindMediaByRelativePath(unreferencedMedia)
			require.NoError(t, err)
			require.Nil(t, media)
		}

	})

}

func TestCommandRestore(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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
		file, err := CurrentCollection().LoadFileByPath("go.md")
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
		file, err = CurrentCollection().LoadFileByPath("go.md")
		require.NoError(t, err)
		require.Nil(t, file)
	})

}

func TestCommandCommit(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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
		SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")
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
		require.FileExists(t, filepath.Join(origin, "config"))
		require.FileExists(t, filepath.Join(origin, "index"))
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

	t.Run("Pull before push", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")
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

		Reset()

		// Create a new (empty) repository
		root := SetUpCollectionFromTempDir(t)
		// Configure same origin
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Create a new commit from a new file
		newFilepath := filepath.Join(root, "python.md")
		err = os.WriteFile(newFilepath, []byte(`# Python

## Flashcard: Python's creator

Who invented Python?
---
Guido van Rossum
`), 0644)
		require.NoError(t, err)
		err = CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("new commit")
		require.NoError(t, err)

		// Try to push before pulling first
		err = CurrentDB().Push()
		require.ErrorContains(t, err, "missing commits from origin")

		// Pull first
		err = CurrentDB().Pull()
		require.NoError(t, err)

		// Try to repush
		err = CurrentDB().Push()
		require.NoError(t, err)
	})

}

func TestCommandStatus(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		UseSequenceOID(t)

		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		newFilepath := filepath.Join(root, "python.md")
		err = os.WriteFile(newFilepath, []byte(`# Python

## Flashcard: Python's creator

Who invented Python?
---
Guido van Rossum
`), 0644)
		require.NoError(t, err)

		output, err := CurrentCollection().Status()
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
	added:	file "go.md" [0000000000000000000000000000000000000001]
	added:	note "Reference: Golang History" [0000000000000000000000000000000000000002]
	added:	link "https://go.dev/doc/" [0000000000000000000000000000000000000005]
	added:	note "Flashcard: Golang Logo" [0000000000000000000000000000000000000003]
	added:	media medias/go.svg [0000000000000000000000000000000000000006]
	added:	note "TODO: Conferences" [0000000000000000000000000000000000000004]
	added:	reminder #reminder-2023-06-26 [0000000000000000000000000000000000000007]
	added:	flashcard "Golang Logo" [0000000000000000000000000000000000000008]

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
	added:	python.md
		`), strings.TrimSpace(output))

		// Restore
		err = CurrentDB().Restore()
		require.NoError(t, err)

		// Status must report no change
		output, err = CurrentCollection().Status()
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
	added:	go.md
	added:	python.md
		`), strings.TrimSpace(output))

		// Add a new file
		err = CurrentCollection().Add("python.md")
		require.NoError(t, err)

		// Status must report only the new files
		output, err = CurrentCollection().Status()
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
	added:	file "python.md" [0000000000000000000000000000000000000009]
	added:	note "Flashcard: Python's creator" [0000000000000000000000000000000000000010]
	added:	flashcard "Python's creator" [0000000000000000000000000000000000000011]

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
	added:	go.md
		`), strings.TrimSpace(output))

		// Add the old file
		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)

		// Status must report both files
		output, err = CurrentCollection().Status()
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
	added:	file "python.md" [0000000000000000000000000000000000000009]
	added:	note "Flashcard: Python's creator" [0000000000000000000000000000000000000010]
	added:	flashcard "Python's creator" [0000000000000000000000000000000000000011]
	added:	file "go.md" [0000000000000000000000000000000000000012]
	added:	note "Reference: Golang History" [0000000000000000000000000000000000000013]
	added:	link "https://go.dev/doc/" [0000000000000000000000000000000000000016]
	added:	note "Flashcard: Golang Logo" [0000000000000000000000000000000000000014]
	added:	media medias/go.svg [0000000000000000000000000000000000000017]
	added:	note "TODO: Conferences" [0000000000000000000000000000000000000015]
	added:	reminder #reminder-2023-06-26 [0000000000000000000000000000000000000018]
	added:	flashcard "Golang Logo" [0000000000000000000000000000000000000019]
		`), strings.TrimSpace(output))
	})

}

func TestCommandGC(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		err := CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)
		logo, err := CurrentCollection().FindMediaByRelativePath("medias/go.svg")
		require.NoError(t, err)
		require.NotNil(t, logo)
		require.Len(t, logo.BlobRefs, 3)
		logoOriginalBlob := logo.BlobRefs[0]
		// Check local
		require.FileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(logoOriginalBlob.OID)))
		// Check origin
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoOriginalBlob.OID)))

		// Update the media file
		err = os.WriteFile(filepath.Join(root, "go.md"), []byte(`
# Go

## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./medias/go.png)
`), 0644)
		require.NoError(t, err)

		err = CurrentCollection().Add(".") // To force medias cleaning
		require.NoError(t, err)
		err = CurrentDB().Commit("update go.svg -> go.png")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)

		logo, err = CurrentCollection().FindMediaByRelativePath("medias/go.svg")
		require.NoError(t, err)
		require.NotNil(t, logo) // Must still exists as we delay the deletion until next gc
		logo, err = CurrentCollection().FindMediaByRelativePath("medias/go.png")
		require.NoError(t, err)
		require.NotNil(t, logo)
		require.Len(t, logo.BlobRefs, 2)
		logoModifiedBlob := logo.BlobRefs[0]
		require.NotEqual(t, logoOriginalBlob.OID, logoModifiedBlob.OID) // Must be different blobs
		// Check local
		require.FileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(logoOriginalBlob.OID))) // Must still exists
		require.FileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(logoModifiedBlob.OID)))
		// Check origin
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoOriginalBlob.OID))) // Must still exists
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoModifiedBlob.OID)))

		// Run "nt gc"
		err = CurrentDB().GC()
		require.NoError(t, err)
		// Only the referenced blob must now exists
		// Check local
		require.NoFileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(logoOriginalBlob.OID))) // garbage collected
		require.FileExists(t, filepath.Join(root, ".nt/objects/", OIDToPath(logoModifiedBlob.OID)))
		// Check origin
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoOriginalBlob.OID))) // not garbage collected by this command
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoModifiedBlob.OID)))

		// Run "nt origin gc"
		err = CurrentDB().OriginGC()
		require.NoError(t, err)
		require.NoFileExists(t, filepath.Join(origin, OIDToPath(logoOriginalBlob.OID))) // garbage collected
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoModifiedBlob.OID)))
	})

}

func TestCommandCountObjects(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		counters, err := CurrentCollection().Counters()
		require.NoError(t, err)
		assert.Equal(t, 0, counters.CountKind["file"])
		assert.Equal(t, 0, counters.CountKind["note"])
		assert.Equal(t, 0, counters.CountKind["flashcard"])
		assert.Equal(t, 0, counters.CountKind["media"])
		assert.Equal(t, 0, counters.CountKind["link"])
		assert.Equal(t, 0, counters.CountKind["reminder"])

		err = CurrentCollection().Add(".")
		require.NoError(t, err)

		counters, err = CurrentCollection().Counters()
		require.NoError(t, err)
		assert.Greater(t, counters.CountKind["file"], 0)
		assert.Greater(t, counters.CountKind["note"], 0)
		assert.Greater(t, counters.CountKind["flashcard"], 0)
		assert.Greater(t, counters.CountKind["media"], 0)
		assert.Greater(t, counters.CountKind["link"], 0)
		assert.Greater(t, counters.CountKind["reminder"], 0)

		assert.Equal(t, map[string]int{
			"go":      3,
			"history": 1,
		}, counters.CountTags)

		assert.Equal(t, map[string]int{
			"source": 1,
			"tags":   3,
			"title":  3,
		}, counters.CountAttributes)
	})

}
