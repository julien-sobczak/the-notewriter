package core

import (
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

}

func TestCommandRestore(t *testing.T) {

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
		file, err := CurrentCollection().LoadFileByPath("go.md")
		require.NoError(t, err)
		require.NotEqual(t, 0, file.MTime)

		// Reset
		err = CurrentDB().Reset()
		require.NoError(t, err)

		// Check staging area is empty
		idx = ReadIndex()
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

// ReplaceLine replaces a line inside a file.
func ReplaceLine(t *testing.T, path string, lineNumber int, oldLine string, newLine string) {
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(string(data), "\n")
	require.LessOrEqual(t, lineNumber, len(lines))
	require.Equal(t, oldLine, lines[lineNumber-1])
	lines[lineNumber-1] = newLine
	content := strings.Join(lines, "\n")
	os.WriteFile(path, []byte(content), 0644)
}
