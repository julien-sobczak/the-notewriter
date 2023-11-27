package core

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"
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

		result, err := CurrentCollection().Lint(nil, ".")
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
		idx := ReadIndex()
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

		// Check commit is present in commit-graph
		commit, ok := CurrentDB().ReadCommit(commitOID)
		require.True(t, ok)
		assert.Len(t, commit.PackFiles, 1)

		// Check pack file is present
		packFileRef := commit.PackFiles[0]
		packFile, err := CurrentDB().ReadPackFile(packFileRef.OID)
		require.NoError(t, err)
		assert.Len(t, packFile.PackObjects, changes)

		// Check commit graph was updated
		data, err = os.ReadFile(filepath.Join(root, ".nt/objects/info/commit-graph"))
		require.NoError(t, err)
		commitGraph := string(data)
		assert.Contains(t, commitGraph, commit.OID)

		// Check staging area is empty
		idx = ReadIndex()
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

	t.Run("Repetitive", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)
		err = CurrentDB().Commit("Initial commit")
		require.NoError(t, err)

		idx := ReadIndex()
		require.Equal(t, 0, idx.CountChanges())
		require.Equal(t, 8, len(idx.Objects))

		// Check 1: Try to add the same file edited several times
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "What does the **Golang logo** represent?", "(Go) What does the **Golang logo** represent?")
		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)
		idx = ReadIndex()
		require.Equal(t, 3, idx.CountChanges()) // the file + the note + the flashcard
		initialChanges := idx.CountChanges()

		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "(Go) What does the **Golang logo** represent?", "(Go) What does the **logo** represent?")
		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)
		// Check only the changes was overriden and not duplicated
		// We change the same file twice but the second change must override the first one
		idx = ReadIndex()
		require.Equal(t, initialChanges, idx.CountChanges())

		err = CurrentDB().Commit("First commit")
		require.NoError(t, err)

		// Check 2: Try to commit the same file repeatability
		initialObjectsCount := len(idx.Objects)
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "(Go) What does the **logo** represent?", "What is the **logo**?")

		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)
		err = CurrentDB().Commit("Second commit")
		require.NoError(t, err)

		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "What is the **logo**?", "What represents the **logo**?")

		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)
		err = CurrentDB().Commit("Third commit")
		require.NoError(t, err)

		// Check the file is only listed once in the list of all known objects
		idx = ReadIndex()
		require.Len(t, idx.Objects, initialObjectsCount) // must not have changed as we have always edited an existing file
	})

	t.Run("Slug", func(t *testing.T) {
		SetUpCollectionFromTempDir(t)

		// Step 1: Add a new file containing a note note with an explicit slug
		MustWriteFile(t, "python.md", `
---
slug: code
---

# Python

## Flashcard: Python's creator

”@slug: flashcard-python-creator”

Who invented Python?
---
Guido van Rossum

## Reference: The History of Python

Python was conceived in the Netherlands as a successor to the ABC programming language.
`)
		err := CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("first commit")
		require.NoError(t, err)

		notePythonInventorInitial := MustFindNoteByPathAndTitle(t, "python.md", "Flashcard: Python's creator")
		notePythonHistoryInitial := MustFindNoteByPathAndTitle(t, "python.md", "Reference: The History of Python")

		assert.Equal(t, "flashcard-python-creator", notePythonInventorInitial.Slug)
		assert.Equal(t, "code-reference-the-history-of-python", notePythonHistoryInitial.Slug)

		// Rename the file, the title and update the content of the notes
		MustWriteFile(t, "programming.md", `
---
slug: code
---

# Programming

## Flashcard: The Creator of Python

”@slug: flashcard-python-creator”

Who invented Python?
---
Guido van Rossum

## Reference: The History of Python Language

Python was conceived in the Netherlands by Guido van Rossum as a successor to the ABC programming language.
`)
		MustDeleteFile(t, "python.md")
		// The wikilink, the hash and title are now different.
		// Only the note with the specified slug are updated.
		// The other note is recreated from scratch as no match can be made.

		err = CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("second commit")
		require.NoError(t, err)

		notePythonInventorAfter := MustFindNoteByPathAndTitle(t, "programming.md", "Flashcard: The Creator of Python")
		notePythonHistoryAfter := MustFindNoteByPathAndTitle(t, "programming.md", "Reference: The History of Python Language")

		assert.Equal(t, "flashcard-python-creator", notePythonInventorAfter.Slug)
		assert.Equal(t, "code-reference-the-history-of-python-language", notePythonHistoryAfter.Slug)

		assert.Equal(t, notePythonInventorInitial.OID, notePythonInventorAfter.OID) // Same
		assert.NotEqual(t, notePythonHistoryInitial, notePythonHistoryAfter.OID)    // Different

		// Check that we still have only two notes in database
		fmt.Println(CurrentConfig().RootDirectory)
		stats, err := CurrentCollection().StatsInDB()
		require.NoError(t, err)
		assert.Equal(t, 1, stats.Objects["file"])
		assert.Equal(t, 2, stats.Objects["note"])
	})

}

func TestCommandReset(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		CurrentLogger().SetVerboseLevel(VerboseDebug)

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		// Check index file
		idx := ReadIndex()
		changes := idx.CountChanges()
		require.Greater(t, changes, 0)
		require.Len(t, idx.Objects, 0)

		// Check database
		file, err := CurrentCollection().FindFileByRelativePath("go.md")
		require.NoError(t, err)
		require.NotEqual(t, 0, file.MTime)

		// Reset
		err = CurrentDB().Reset()
		require.NoError(t, err)

		// Check staging area is empty
		idx = ReadIndex()
		require.Equal(t, 0, idx.CountChanges())

		// Check database is empty
		file, err = CurrentCollection().FindFileByRelativePath("go.md")
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
		headCommit, ok := CurrentDB().ReadCommit(head)
		require.True(t, ok)
		require.Len(t, headCommit.PackFiles, 1)
		packFile := headCommit.PackFiles[0]
		require.FileExists(t, filepath.Join(origin, OIDToPath(packFile.OID)))

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

	t.Run("Push with staged changes", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")
		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Commit
		err := CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)

		// Stage a few changes
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

		// Push
		err = CurrentDB().Push()
		require.NoError(t, err)

		// Check staged changes are not pushed
		require.FileExists(t, filepath.Join(origin, "index"))
		originIndex, err := NewIndexFromPath(filepath.Join(origin, "index"))
		require.NoError(t, err)
		assert.Empty(t, originIndex.StagingArea) // Must not include non-committed changes
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

		// Reset
		err = CurrentDB().Reset()
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

	t.Run("Reclaim Orphan Blobs", func(t *testing.T) {
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
		require.Nil(t, logo) // Old file must not longer exist
		logo, err = CurrentCollection().FindMediaByRelativePath("medias/go.png")
		require.NoError(t, err)
		require.NotNil(t, logo) // No file must now exist
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

		// Run "nt push"
		err = CurrentDB().Push() // apply local gc changes
		require.NoError(t, err)
		require.NoFileExists(t, filepath.Join(origin, OIDToPath(logoOriginalBlob.OID))) // garbage collected
		require.FileExists(t, filepath.Join(origin, OIDToPath(logoModifiedBlob.OID)))
	})

	t.Run("Edit PackFiles", func(t *testing.T) {
		root := SetUpCollectionFromTempDir(t)

		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Limit the number of objects per pack file to ease debugging
		maxObjectsPerPackFile := 3
		CurrentConfig().ConfigFile.Core.MaxObjectsPerPackFile = maxObjectsPerPackFile

		// Step 1:
		// ------
		// Create two files containing many notes by ensuring we have several packfiles:
		// - A pack file containing only objects coming from file A
		// - A pack file containing objects coming from file A AND file B
		// - A pack file containing only objects coming from file B
		maxNotesPerFile := maxObjectsPerPackFile + 1
		var contentA bytes.Buffer
		contentA.WriteString("# New File A\n\n")
		for i := 0; i < maxNotesPerFile; i++ {
			contentA.WriteString(fmt.Sprintf("## Note: A%d\n\nBlabla\n\n", i+1))
		}
		err := os.WriteFile(filepath.Join(root, "a.md"), contentA.Bytes(), 0644)
		require.NoError(t, err)

		var contentB bytes.Buffer
		contentB.WriteString("# New File A\n\n")
		for i := 0; i < maxNotesPerFile; i++ {
			contentB.WriteString(fmt.Sprintf("## Note: B%d\n\nBlabla\n\n", i+1))
		}
		err = os.WriteFile(filepath.Join(root, "b.md"), contentB.Bytes(), 0644)
		require.NoError(t, err)

		// Commit
		err = CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)

		// Inspect commit
		initialCommit := CurrentDB().Head()
		require.NotNil(t, initialCommit)
		require.Len(t, initialCommit.PackFiles, 4)
		initialPackFilesOIDs := initialCommit.PackFiles.OIDs() // Backup to compare later after gc

		// Inspect stats
		statsOnDisk, err := CurrentDB().StatsOnDisk()
		require.NoError(t, err)
		assert.Equal(t, statsOnDisk.Commits, 1)
		assert.Equal(t, statsOnDisk.Objects["file"], 2)
		assert.Equal(t, statsOnDisk.Objects["note"], maxNotesPerFile*2) // file A + file B
		assert.Equal(t, statsOnDisk.IndexObjects, maxNotesPerFile*2+2)

		CurrentDB().PrintIndex()

		// Step 2:
		// ------
		// We will now completely edit the file A and commit again
		contentA.Reset()
		contentA.WriteString("# New File A\n\n")
		for i := 0; i < maxNotesPerFile; i++ {
			contentA.WriteString(fmt.Sprintf("## Note: A%d\n\nBlablaBlaBla\n\n", i+1)) // New text
		}
		err = os.WriteFile(filepath.Join(root, "a.md"), contentA.Bytes(), 0644)
		require.NoError(t, err)

		// Commit
		err = CurrentCollection().Add(".")
		require.NoError(t, err)
		err = CurrentDB().Commit("second commit")
		require.NoError(t, err)
		err = CurrentDB().Push()
		require.NoError(t, err)

		// We now have two commits. The second commit rewrite all objects coming from file A.
		// It means:
		// - The first pack file of the initial commit can be reclaimed (contains only old versions).
		// - The second pack file of the initial commit can be edited to remove objects from file A.
		// - The third pack file of the initial commit must be left untouched as the pack files from the second commit.

		statsOnDisk, err = CurrentDB().StatsOnDisk()
		require.NoError(t, err)
		assert.Equal(t, statsOnDisk.Commits, 2)
		assert.Equal(t, statsOnDisk.Objects["file"], 3)                 // old file A + actual file A + actual file B
		assert.Equal(t, statsOnDisk.Objects["note"], maxNotesPerFile*3) // Same logic
		assert.Equal(t, statsOnDisk.IndexObjects, maxNotesPerFile*2+2)

		CurrentDB().PrintIndex()

		// Step 3:
		// ------
		// Run the GC locally
		// We must still have 2 commits but the old revisions must no longer exist.

		// Run "nt gc"
		err = CurrentDB().GC()
		require.NoError(t, err)

		// Inspect stats
		statsOnDisk, err = CurrentDB().StatsOnDisk()
		require.NoError(t, err)
		assert.Equal(t, statsOnDisk.Commits, 2)                         // unchanged
		assert.Equal(t, statsOnDisk.Objects["file"], 2)                 // only actual files ...
		assert.Equal(t, statsOnDisk.Objects["note"], maxNotesPerFile*2) // ... and actual notes

		// Let's inspect the first commit content to validate
		initialCommitEdited, ok := CurrentDB().ReadCommit(initialCommit.OID)
		require.True(t, ok)
		assert.Len(t, initialCommitEdited.PackFiles, 3) // The first pack file must have been dropped
		packFile1, err := CurrentDB().ReadPackFile(initialCommitEdited.PackFiles[0].OID)
		require.NoError(t, err)
		packFile2, err := CurrentDB().ReadPackFile(initialCommitEdited.PackFiles[1].OID)
		require.NoError(t, err)
		// Ensure only objects from file B remains
		var allPackObjects []*PackObject
		allPackObjects = append(allPackObjects, packFile1.PackObjects...)
		allPackObjects = append(allPackObjects, packFile2.PackObjects...)
		for _, packObject := range allPackObjects {
			obj := packObject.ReadObject()
			switch typedObject := obj.(type) {
			case *File:
				require.Equal(t, "b.md", typedObject.RelativePath)
			case *Note:
				require.Equal(t, "b.md", typedObject.RelativePath)
			}
		}

		// Step 4:
		// ------
		// Push to apply GC changes remotely
		// We must reclaim the same objects as in step 3.
		// But first check for timestamps before the operation
		path1 := filepath.Join(origin, OIDToPath(initialPackFilesOIDs[0]))
		path2 := filepath.Join(origin, OIDToPath(initialPackFilesOIDs[1]))
		path3 := filepath.Join(origin, OIDToPath(initialPackFilesOIDs[2]))
		path4 := filepath.Join(origin, OIDToPath(initialPackFilesOIDs[3]))
		timestamp2Before := MustReadMTime(t, path2)
		timestamp3Before := MustReadMTime(t, path3)
		timestamp4Before := MustReadMTime(t, path4)
		err = CurrentDB().Push()
		require.NoError(t, err)
		require.NoFileExists(t, path1) // garbage collected
		require.FileExists(t, path2)   // edited
		require.FileExists(t, path3)   // unchanged
		require.FileExists(t, path4)   // unchanged
		timestamp2After := MustReadMTime(t, path2)
		timestamp3After := MustReadMTime(t, path3)
		timestamp4After := MustReadMTime(t, path4)
		assert.NotEqual(t, timestamp2Before, timestamp2After) // edited
		assert.Equal(t, timestamp3Before, timestamp3After)    // unchanged
		assert.Equal(t, timestamp4Before, timestamp4After)    // unchanged
	})

}

func TestCommandDiff(t *testing.T) {

	t.Run("Diff", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		// Step 1: Nothing staged

		diff, err := CurrentCollection().Diff(true)
		require.NoError(t, err)
		assert.Equal(t, "", diff)

		diff, err = CurrentCollection().Diff(false) // Must contains all new objects
		require.NoError(t, err)
		expected := "" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,5 @@\n" +
			"+`#history`\n" +
			"+\n" +
			"+`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n" +
			"+\n+[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n" +
			"\\ No newline at end of file\n" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,7 @@\n" +
			"+What does the **Golang logo** represent?\n" +
			"+\n" +
			"+---\n" +
			"+\n" +
			"+A **gopher**.\n" +
			"+\n" +
			"+![Logo](./medias/go.svg)\n" +
			"\\ No newline at end of file\n" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,1 @@\n" +
			"+* [Gophercon Europe](https://gophercon.eu/) `#reminder-2023-06-26`\n" +
			"\\ No newline at end of file\n"
		assert.Equal(t, expected, diff) // No diff as no objects in staging area

		// Step 2: Add a file

		err = CurrentCollection().Add("go.md")
		require.NoError(t, err)

		diff, err = CurrentCollection().Diff(true) // Only the file staged must be returned
		require.NoError(t, err)
		expected = "--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,5 @@\n" +
			"+`#history`\n" +
			"+\n" +
			"+`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n" +
			"+\n" +
			"+[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n" +
			"\\ No newline at end of file\n" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,7 @@\n" +
			"+What does the **Golang logo** represent?\n" +
			"+\n" +
			"+---\n" +
			"+\n" +
			"+A **gopher**.\n" +
			"+\n" +
			"+![Logo](./medias/go.svg)\n" +
			"\\ No newline at end of file\n" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -0,0 +1,1 @@\n" +
			"+* [Gophercon Europe](https://gophercon.eu/) `#reminder-2023-06-26`\n" +
			"\\ No newline at end of file\n"
		assert.Equal(t, expected, diff)

		diff, err = CurrentCollection().Diff(false) // No other file are present, must be empty
		require.NoError(t, err)
		assert.Equal(t, "", diff)

		// Step 3: Commit the staged file

		err = CurrentDB().Commit("initial commit")
		require.NoError(t, err)

		diff, err = CurrentCollection().Diff(true) // Staging area is empty = must be empty
		require.NoError(t, err)
		assert.Equal(t, "", diff)

		diff, err = CurrentCollection().Diff(false) // No local change = must be empty too
		require.NoError(t, err)
		assert.Equal(t, "", diff)

		// Step 4: Edit a single note file

		newFilepath := filepath.Join(root, "go.md")
		err = os.WriteFile(newFilepath, []byte(`
# Go

## Reference: Golang History

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.


## TODO: Conferences

* [Gophercon Europe](https://gophercon.eu/) `+"`#reminder-2023-06-26`"+`
`), 0644)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		diff, err = CurrentCollection().Diff(true) // Staging area is empty = must be empty
		require.NoError(t, err)
		assert.Equal(t, "", diff)

		diff, err = CurrentCollection().Diff(false) // Must report the updated and deleted notes
		require.NoError(t, err)
		expected = "--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -1,5 +1,1 @@\n" +
			"-`#history`\n" +
			"-\n" +
			"-`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n" +
			"-\n" +
			" [Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n" +
			"\\ No newline at end of file\n" +
			"--- a/go.md\n" +
			"+++ b/go.md\n" +
			"@@ -1,7 +0,0 @@\n" +
			"-What does the **Golang logo** represent?\n" +
			"-\n" +
			"----\n" +
			"-\n" +
			"-A **gopher**.\n" +
			"-\n" +
			"-![Logo](./medias/go.svg)\n" +
			"\\ No newline at end of file\n"
		assert.Equal(t, expected, diff) // TODO BUG the deleted flashcard is missing
	})

}

/* Learning Tests */

func TestSourcegraphGoDiff(t *testing.T) {
	// Learning test to demonstrate the working of the library
	inputA := `
{
	SSID:      "CoffeeShopWiFi",
	IPAddress: net.IPv4(192, 168, 0, 1),
	NetMask:   net.IPv4Mask(255, 255, 0, 0),
	Clients: []Client{{
		Hostname:  "ristretto",
		IPAddress: net.IPv4(192, 168, 0, 116),
	}, {
		Hostname:  "aribica",
		IPAddress: net.IPv4(192, 168, 0, 104),
		LastSeen:  time.Date(2009, time.November, 10, 23, 6, 32, 0, time.UTC),
	}, {
		Hostname:  "macchiato",
		IPAddress: net.IPv4(192, 168, 0, 153),
		LastSeen:  time.Date(2009, time.November, 10, 23, 39, 43, 0, time.UTC),
	}, {
		Hostname:  "espresso",
		IPAddress: net.IPv4(192, 168, 0, 121),
	}, {
		Hostname:  "latte",
		IPAddress: net.IPv4(192, 168, 0, 219),
		LastSeen:  time.Date(2009, time.November, 10, 23, 0, 23, 0, time.UTC),
	}, {
		Hostname:  "americano",
		IPAddress: net.IPv4(192, 168, 0, 188),
		LastSeen:  time.Date(2009, time.November, 10, 23, 3, 5, 0, time.UTC),
	}},
}
`
	inputB := `
{
	SSID:      "CoffeeShopWiFi",
	IPAddress: net.IPv4(192, 168, 0, 2),
	NetMask:   net.IPv4Mask(255, 255, 0, 0),
	Clients: []Client{{
		Hostname:  "ristretto",
		IPAddress: net.IPv4(192, 168, 0, 116),
	}, {
		Hostname:  "aribica",
		IPAddress: net.IPv4(192, 168, 0, 104),
		LastSeen:  time.Date(2009, time.November, 10, 23, 6, 32, 0, time.UTC),
	}, {
		Hostname:  "macchiato",
		IPAddress: net.IPv4(192, 168, 0, 153),
		LastSeen:  time.Date(2009, time.November, 10, 23, 39, 43, 0, time.UTC),
	}, {
		Hostname:  "espresso",
		IPAddress: net.IPv4(192, 168, 0, 121),
	}, {
		Hostname:  "latte",
		IPAddress: net.IPv4(192, 168, 0, 221),
		LastSeen:  time.Date(2009, time.November, 10, 23, 0, 23, 0, time.UTC),
	}},
}
`
	patch := godiffpatch.GeneratePatch("test.txt", inputA, inputB)
	expected := "" +
		"--- a/test.txt\n" +
		"+++ b/test.txt\n" +
		"@@ -1,7 +1,7 @@\n" +
		" \n" +
		" {\n" +
		" 	SSID:      \"CoffeeShopWiFi\",\n" +
		"-	IPAddress: net.IPv4(192, 168, 0, 1),\n" +
		"+	IPAddress: net.IPv4(192, 168, 0, 2),\n" +
		" 	NetMask:   net.IPv4Mask(255, 255, 0, 0),\n" +
		" 	Clients: []Client{{\n" +
		" 		Hostname:  \"ristretto\",\n" +
		"@@ -19,11 +19,7 @@\n" +
		" 		IPAddress: net.IPv4(192, 168, 0, 121),\n" +
		" 	}, {\n" +
		" 		Hostname:  \"latte\",\n" +
		"-		IPAddress: net.IPv4(192, 168, 0, 219),\n" +
		"+		IPAddress: net.IPv4(192, 168, 0, 221),\n" +
		" 		LastSeen:  time.Date(2009, time.November, 10, 23, 0, 23, 0, time.UTC),\n" +
		"-	}, {\n" +
		"-		Hostname:  \"americano\",\n" +
		"-		IPAddress: net.IPv4(192, 168, 0, 188),\n" +
		"-		LastSeen:  time.Date(2009, time.November, 10, 23, 3, 5, 0, time.UTC),\n" +
		" 	}},\n" +
		" }\n"

	assert.Equal(t, expected, patch)
}

/* Test Helpers */

// MustReadMTime returns the last modification time for a local file using stat.
func MustReadMTime(t *testing.T, path string) time.Time {
	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	return fileInfo.ModTime()
}
