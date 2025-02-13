package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	godiffpatch "github.com/sourcegraph/go-diff-patch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLint(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		SetUpRepositoryFromTempDir(t)

		// Enable a single rule
		WriteFileFromRelativePath(t, ".nt/lint", `
rules:
- name: no-duplicate-note-title
`)
		configOnce.Reset()

		// Create a file violating the rule
		WriteFileFromRelativePath(t, "lint.md", `
# Linter

## Note: Name

This is a first note

## Note: Name

This is a second note
`)

		result, err := CurrentRepository().Lint(nil, AnyPath)
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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		// Check index file
		idx := MustReadIndex()
		assert.Len(t, idx.Entries, 2) // go.md + medias/go.svg
		entry1 := idx.Entries[0]
		entry2 := idx.Entries[1]
		assert.Equal(t, "medias/go.svg", entry1.RelativePath) // Medias are processed first
		assert.Equal(t, "go.md", entry2.RelativePath)
		// Must be staged
		assert.True(t, entry1.Staged)
		assert.FileExists(t, PackFilePath(entry1.StagedPackFileOID))
		assert.True(t, entry2.Staged)
		assert.FileExists(t, PackFilePath(entry2.StagedPackFileOID))

		// Commit
		err = CurrentRepository().Commit("initial commit")
		require.NoError(t, err)

		// Check index file
		idx = MustReadIndex()
		assert.Len(t, idx.Entries, 2) // not changed
		entry1 = idx.Entries[0]
		entry2 = idx.Entries[1]
		assert.Equal(t, "medias/go.svg", entry1.RelativePath)
		assert.Equal(t, "go.md", entry2.RelativePath)
		// Must no longer be staged
		assert.False(t, entry1.Staged)
		assert.FileExists(t, PackFilePath(entry1.PackFileOID))
		assert.False(t, entry2.Staged)
		assert.FileExists(t, PackFilePath(entry1.PackFileOID))
	})

	t.Run("Add Media", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMedias")

		err := CurrentRepository().Add(AnyPath)
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
			media, err := CurrentRepository().FindMediaByRelativePath(expectedMedia)
			require.NoError(t, err)
			require.NotNil(t, media)
			for _, blob := range media.Blobs() {
				assert.FileExists(t, blob.ObjectPath())
			}
		}

		// Check non-referenced medias are missing
		unreferencedMedias := []string{
			"medias/branch-portrait.avif",
		}
		for _, unreferencedMedia := range unreferencedMedias {
			media, err := CurrentRepository().FindMediaByRelativePath(unreferencedMedia)
			require.NoError(t, err)
			require.Nil(t, media)
		}

	})

	t.Run("Repetitive", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit("Initial commit")
		require.NoError(t, err)

		idx := MustReadIndex()
		require.Len(t, idx.Entries, 2) // markdown + 1 referenced media
		require.Len(t, idx.Objects, 8)
		require.Len(t, idx.Blobs, 4)

		// Check 1: Try to add the same file edited several times
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "What does the **Golang logo** represent?", "(Go) What does the **Golang logo** represent?")
		err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		// Edit again before the commit
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "(Go) What does the **Golang logo** represent?", "(Go) What does the **logo** represent?")
		err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit("First commit")
		require.NoError(t, err)

		// Check 2: Try to commit the same file repeatability
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "(Go) What does the **logo** represent?", "What is the **logo**?")
		err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit("Second commit")
		require.NoError(t, err)
		ReplaceLine(t, filepath.Join(root, "go.md"), 19, "What is the **logo**?", "What represents the **logo**?")
		err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit("Third commit")
		require.NoError(t, err)

		// Check the file is still listed only once
		idx = MustReadIndex()
		assert.Len(t, idx.Entries, 2)
		assert.Len(t, idx.Objects, 8)
		assert.Len(t, idx.Blobs, 4)
	})

}

func TestCommandReset(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		CurrentLogger().SetVerboseLevel(VerboseDebug)

		err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		// Check index file
		idx := MustReadIndex()
		// Entries have been staged
		require.Greater(t, len(idx.Entries), 0)
		require.Greater(t, len(idx.Objects), 0)
		require.Greater(t, len(idx.Blobs), 0)
		firstEntry := idx.Entries[0]
		firstEntryPath := PackFilePath(firstEntry.StagedPackFileOID)
		assert.FileExists(t, firstEntryPath)

		// Check database
		// Staged entries are added in database before commit
		file, err := CurrentRepository().FindFileByRelativePath("go.md")
		require.NoError(t, err)
		require.NotNil(t, file)

		// Reset
		err = CurrentRepository().Reset(AnyPath)
		require.NoError(t, err)

		// Check index again
		idx = MustReadIndex()
		require.Empty(t, idx.Entries)
		require.Empty(t, idx.Objects)
		require.Empty(t, idx.Blobs)
		assert.FileExists(t, firstEntryPath) // We don't delete the pack files.
		// If the add command is rerun, the packfile will be reused.
		// (= great for medias to avoid regenerating the blobs)

		// Check database is empty
		file, err = CurrentRepository().FindFileByRelativePath("go.md")
		require.NoError(t, err)
		require.Nil(t, file)
	})

}

func TestCommandCommit(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit("initial commit")
		require.NoError(t, err)

		require.NoFileExists(t, filepath.Join(root, "python.md"))
		MustWriteFile(t, "python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)

		err = CurrentRepository().Commit("empty commit")
		require.ErrorContains(t, err, "nothing to commit")

		// Create a second commit
		err = CurrentRepository().Add(PathSpecs{"python.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit("second commit")
		require.NoError(t, err)
	})

}

func TestCommandPushPull(t *testing.T) {

	t.Run("Push", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Push
		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit("initial commit")
		require.NoError(t, err)
		err = CurrentRepository().Push(false, false)
		require.NoError(t, err)

		// Check origin
		require.FileExists(t, filepath.Join(origin, "index"))
		// require.FileExists(t, filepath.Join(origin, "config")) // TODO push config
		CurrentIndex().Walk(AnyPath, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
			// The origin FS must contains a file for every pack file and blob
			assert.FileExists(t, filepath.Join(origin, entry.Ref().ObjectRelativePath()))
			for _, blob := range blobs {
				assert.FileExists(t, filepath.Join(origin, blob.Ref().ObjectRelativePath()))
			}
			return nil
		})
		countEntries := len(CurrentIndex().Entries)
		countObjects := len(CurrentIndex().Objects)
		countBlobs := len(CurrentIndex().Blobs)

		// Force a new temp repository
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		// but with the same origin
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}
		err = CurrentRepository().Pull(false, false)
		require.NoError(t, err)
		// We must now have the same number of entries, objects and blobs as pushed before
		assert.Equal(t, countEntries, len(CurrentIndex().Entries))
		assert.Equal(t, countObjects, len(CurrentIndex().Objects))
		assert.Equal(t, countBlobs, len(CurrentIndex().Blobs))
	})

	t.Run("Push/Pull with staged changes", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		// Configure origin
		origin := t.TempDir()
		CurrentConfig().ConfigFile.Remote = ConfigRemote{
			Type: "fs",
			Dir:  origin,
		}

		// Commit
		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit("initial commit")
		require.NoError(t, err)

		// Stage a few changes
		MustWriteFile(t, "python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)
		err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		// Push
		err = CurrentRepository().Push(false, false)
		require.ErrorContains(t, err, "changes not commited")
		// Pull
		err = CurrentRepository().Pull(false, false)
		require.ErrorContains(t, err, "changes not commited")
	})

}

func TestCommandStatus(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		oid.UseSequence(t)

		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		// Add
		err := CurrentRepository().Add([]PathSpec{"go.md"})
		require.NoError(t, err)

		// Edit a new file
		MustWriteFile(t, "python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)
		require.NoError(t, err)

		output, err := CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
       added: go.md (+7)
       added: medias/go.svg (+1)

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
       added: python.md
		`), strings.TrimSpace(output))

		// Reset
		err = CurrentRepository().Reset(AnyPath)
		require.NoError(t, err)

		// Status must report no change
		output, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
       added: go.md
       added: medias/go.svg
       added: python.md
		`), strings.TrimSpace(output))

		// Add a new file
		err = CurrentRepository().Add([]PathSpec{"python.md"})
		require.NoError(t, err)

		// Status must report only the new files
		output, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
       added: python.md (+3)

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
       added: go.md
       added: medias/go.svg
		`), strings.TrimSpace(output))

		// Add the old file
		err = CurrentRepository().Add([]PathSpec{"go.md"})
		require.NoError(t, err)

		// Status must report both files
		output, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(`
Changes to be committed:
  (use "nt restore..." to unstage)
       added: go.md (+7)
       added: medias/go.svg (+1)
       added: python.md (+3)
		`), strings.TrimSpace(output))
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
