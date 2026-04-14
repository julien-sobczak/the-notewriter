package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLint(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		NewTestRepository(t,

			// Enable a single rule
			WithFile(".nt/config.jsonnet", `
{
    core: { extensions: ["md"] },
	noteTypes: {
		"Note": {
			name: "Note"
		}
	},
	linter: {
		rules: [
			{ name: "no-duplicate-note-title" }
		]
	},
}
`),

			// Create a file violating the rule
			WithFile("lint.md", `
# Linter

## Note: Name

This is a first note

## Note: Name

This is a second note
`))
		configOnce.Reset() // Force the config to be reloaded

		result, err := CurrentRepository().Lint(AnyPath, nil)
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
		NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
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
		err = CurrentRepository().Commit(true)
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
		NewTestRepository(t, FromGoldenDirNamed("TestMedias"))

		_, err := CurrentRepository().Add(AnyPath)
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

	t.Run("Links", func(t *testing.T) {
		NewTestRepository(t, FromGoldenDirNamed("TestLinks"))

		_, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		// Check links have been created
		links, err := CurrentRepository().FindLinks()
		require.NoError(t, err)
		require.NotEmpty(t, links)

		// Check implicit links
		fileD, err := CurrentRepository().FindFileByRelativePath("d.md")
		require.NoError(t, err)
		require.NotNil(t, fileD)
		notes, err := CurrentRepository().FindNotesByFileOID(fileD.OID)
		require.NoError(t, err)
		require.Len(t, notes, 3)

		noteAncestor := notes[0]
		noteParent := notes[1]
		noteChild := notes[2]
		require.Equal(t, "Ancestor", noteAncestor.ShortTitle.String())
		require.Equal(t, "Parent", noteParent.ShortTitle.String())
		require.Equal(t, "Child", noteChild.ShortTitle.String())

		assert.Contains(t, links, &Link{
			SourceOID:  noteAncestor.OID,
			SourceKind: "note",
			TargetOID:  noteParent.OID,
			TargetKind: "note",
			Type:       "includes",
		})
		assert.Contains(t, links, &Link{
			SourceOID:  noteParent.OID,
			SourceKind: "note",
			TargetOID:  noteChild.OID,
			TargetKind: "note",
			Type:       "includes",
		})
	})

	t.Run("Repetitive", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		idx := MustReadIndex()
		require.Len(t, idx.Entries, 2) // markdown + 1 referenced media
		require.Len(t, idx.Objects, 9)
		require.Len(t, idx.Blobs, 4)

		// Check 1: Try to add the same file edited several times
		tr.ReplaceLine("go.md", 19, "What does the **Golang logo** represent?", "(Go) What does the **Golang logo** represent?")
		_, err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		// Edit again before the commit
		tr.ReplaceLine("go.md", 19, "(Go) What does the **Golang logo** represent?", "(Go) What does the **logo** represent?")
		_, err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// Check 2: Try to commit the same file repeatability
		tr.ReplaceLine("go.md", 19, "(Go) What does the **logo** represent?", "What is the **logo**?")
		_, err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)
		tr.ReplaceLine("go.md", 19, "What is the **logo**?", "What represents the **logo**?")
		_, err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// Check the file is still listed only once
		idx = MustReadIndex()
		assert.Len(t, idx.Entries, 2)
		assert.Len(t, idx.Objects, 9)
		assert.Len(t, idx.Blobs, 4)
	})

	t.Run("Force", func(t *testing.T) {
		NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		// First add + commit
		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// Add again without any file change: nothing should be staged
		result, err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		assert.Empty(t, result.Upserted)
		assert.Empty(t, result.Deleted)

		// Add again with --force: the file should be reparsed even without mtime change
		CurrentConfig().Force = true
		defer func() { CurrentConfig().Force = false }()

		result, err = CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Upserted)
	})

}

func TestCommandReset(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		CurrentLogger().SetVerboseLevel(VerboseDebug)

		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
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
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		_, err := CurrentRepository().Add(PathSpecs{"go.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		tr.RequireNoFileExists("python.md")
		tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)

		err = CurrentRepository().Commit(true)
		require.ErrorContains(t, err, "nothing to commit")

		// Create a second commit
		_, err = CurrentRepository().Add(PathSpecs{"python.md"})
		require.NoError(t, err)

		err = CurrentRepository().Commit(true)
		require.NoError(t, err)
	})

}

func TestCommandPushPull(t *testing.T) {

	t.Run("Push", func(t *testing.T) {
		NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		// Configure origin
		origin := t.TempDir()
		CurrentConfigFile().Remotes = []*ConfigRemote{
			{
				Name: "origin",
				Type: "fs",
				Dir:  origin,
			},
		}

		// Push
		_, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)
		require.NoError(t, CurrentConfig().Save()) // Simulate Cobra PostRun logic
		err = CurrentRepository().Push("origin", false, false)
		require.NoError(t, err)

		// Check origin
		require.FileExists(t, filepath.Join(origin, "index"))
		require.FileExists(t, filepath.Join(origin, "config.json"))
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
		NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		// but with the same origin
		CurrentConfigFile().Remotes = []*ConfigRemote{
			{
				Name: "origin",
				Type: "fs",
				Dir:  origin,
			},
		}
		err = CurrentRepository().Pull("origin", false, false)
		require.NoError(t, err)
		// We must now have the same number of entries, objects and blobs as pushed before
		assert.Equal(t, countEntries, len(CurrentIndex().Entries))
		assert.Equal(t, countObjects, len(CurrentIndex().Objects))
		assert.Equal(t, countBlobs, len(CurrentIndex().Blobs))
	})

	t.Run("Push/Pull with staged changes", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		// Configure origin
		origin := t.TempDir()
		CurrentConfigFile().Remotes = []*ConfigRemote{
			{
				Name: "origin",
				Type: "fs",
				Dir:  origin,
			},
		}

		// Commit
		_, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// Stage a few changes
		tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)
		_, err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		// Push
		err = CurrentRepository().Push("origin", false, false)
		require.ErrorContains(t, err, "changes not committed")
		// Pull
		err = CurrentRepository().Pull("origin", false, false)
		require.ErrorContains(t, err, "changes not committed")
	})

}

func TestCommandStatus(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"), WithSequenceOIDs())

		// Add
		_, err := CurrentRepository().Add([]PathSpec{"go.md"})
		require.NoError(t, err)

		// Edit a new file
		tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)
		require.NoError(t, err)

		result, err := CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, FileStatuses{
			{
				RelativePath: "go.md",
				Status:       "added",
				ObjectsAdded: 8,
			},
			{
				RelativePath: "medias/go.svg",
				Status:       "added",
				ObjectsAdded: 1,
			},
		}, result.ChangesStaged)
		assert.Equal(t, FileStatuses{
			{
				RelativePath: "python.md",
				Status:       "added",
			},
		}, result.ChangesNotStaged)

		// Reset
		err = CurrentRepository().Reset(AnyPath)
		require.NoError(t, err)

		// Status must report no change
		result, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Empty(t, result.ChangesStaged)
		assert.Equal(t, FileStatuses{
			{
				RelativePath: "go.md",
				Status:       "added",
			},
			{
				RelativePath: "medias/go.svg",
				Status:       "added",
			},
			{
				RelativePath: "python.md",
				Status:       "added",
			},
		}, result.ChangesNotStaged)

		// Add a new file
		_, err = CurrentRepository().Add([]PathSpec{"python.md"})
		require.NoError(t, err)

		// Status must report only the new files
		result, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, FileStatuses{
			{
				RelativePath: "python.md",
				Status:       "added",
				ObjectsAdded: 3,
			},
		}, result.ChangesStaged)
		assert.Equal(t, FileStatuses{
			{
				RelativePath: "go.md",
				Status:       "added",
			},
			{
				RelativePath: "medias/go.svg",
				Status:       "added",
			},
		}, result.ChangesNotStaged)

		// Add the old file
		_, err = CurrentRepository().Add([]PathSpec{"go.md"})
		require.NoError(t, err)

		// Status must report both files
		result, err = CurrentRepository().Status(AnyPath)
		require.NoError(t, err)
		assert.Equal(t, FileStatuses{
			{
				RelativePath: "go.md",
				Status:       "added",
				ObjectsAdded: 8,
			},
			{
				RelativePath: "medias/go.svg",
				Status:       "added",
				ObjectsAdded: 1,
			},
			{
				RelativePath: "python.md",
				Status:       "added",
				ObjectsAdded: 3,
			},
		}, result.ChangesStaged)
		assert.Empty(t, result.ChangesNotStaged)
	})

}

func TestCommandDiff(t *testing.T) {

	t.Run("Diff", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"), WithSequenceOIDs(), WithFreezeNow())

		// Step 1: Nothing staged

		diffs, err := CurrentRepository().Diff(AnyPath, true)
		require.NoError(t, err)
		assert.Empty(t, diffs)

		diffs, err = CurrentRepository().Diff(AnyPath, false) // Must contains all new objects
		require.NoError(t, err)

		assert.NotNil(t, diffs.FindFileByTitle("go.md", "Go"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "Note: Golang History"))
		assert.NotNil(t, diffs.FindGotoByName("go.md", "go"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "Flashcard: Golang Logo"))
		assert.NotNil(t, diffs.FindFlashcardByShortTitle("go.md", "Golang Logo"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "TODO: Conferences"))
		assert.NotNil(t, diffs.FindReminderWithTag("go.md", "#reminder-2023-06-26"))
		assert.NotNil(t, diffs.FindMedia("medias/go.svg"))

		// Step 2: Add a file

		_, err = CurrentRepository().Add([]PathSpec{"go.md"})
		require.NoError(t, err)

		diffs, err = CurrentRepository().Diff(AnyPath, true) // Only the file staged must be returned
		require.NoError(t, err)
		assert.NotNil(t, diffs.FindFileByTitle("go.md", "Go"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "Note: Golang History"))
		assert.NotNil(t, diffs.FindGotoByName("go.md", "go"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "Flashcard: Golang Logo"))
		assert.NotNil(t, diffs.FindFlashcardByShortTitle("go.md", "Golang Logo"))
		assert.NotNil(t, diffs.FindNoteByTitle("go.md", "TODO: Conferences"))
		assert.NotNil(t, diffs.FindReminderWithTag("go.md", "#reminder-2023-06-26"))
		// And also the media as referenced
		assert.NotNil(t, diffs.FindMedia("medias/go.svg"))

		diffs, err = CurrentRepository().Diff(AnyPath, false) // No other file are present, must be empty
		require.NoError(t, err)
		assert.Empty(t, diffs)

		// Step 3: Commit the staged file

		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		diffs, err = CurrentRepository().Diff(AnyPath, true) // Staging area is empty = must be empty
		require.NoError(t, err)
		assert.Empty(t, diffs)

		diffs, err = CurrentRepository().Diff(AnyPath, false) // No local change = must be empty too
		require.NoError(t, err)
		assert.Empty(t, diffs)

		// Step 4: Edit a single note file

		tr.WriteFile("go.md", `
# Go

## Note: Golang History

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.


## TODO: Conferences

* [Gophercon Europe](https://gophercon.eu/) `+"`#reminder-2023-06-26`"+`
`)

		tr.FastForward(1 * time.Minute) // Force a new timestamp when creating the new pack file

		diffs, err = CurrentRepository().Diff(AnyPath, true) // Staging area is empty = must be empty
		require.NoError(t, err)
		assert.Empty(t, diffs)

		diffs, err = CurrentRepository().Diff(AnyPath, false) // Must report the updated and deleted notes
		require.NoError(t, err)

		// The file must have been modified
		diff := diffs.FindFileByTitle("go.md", "Go")
		require.NotNil(t, diff)
		assert.True(t, diff.Modified())
		// The edited note must have been modified
		diff = diffs.FindNoteByTitle("go.md", "Note: Golang History")
		require.NotNil(t, diff)
		assert.True(t, diff.Modified())
		// The flashcard must have been deleted (and the associated note)
		diff = diffs.FindNoteByTitle("go.md", "Flashcard: Golang Logo")
		require.NotNil(t, diff)
		assert.True(t, diff.Deleted())
		diff = diffs.FindFlashcardByShortTitle("go.md", "Golang Logo")
		require.NotNil(t, diff)
		assert.True(t, diff.Deleted())
	})

}

func TestCommandGC(t *testing.T) {

	t.Run("Reset File", func(t *testing.T) {
		tr := NewTestRepository(t)

		// Add a new file without committing
		tr.WriteFile("go.md", `# Go`)
		_, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		// A pack file must have been created
		indexEntry := CurrentIndex().GetEntry("go.md")
		require.NotNil(t, indexEntry)
		assert.FileExists(t, indexEntry.Ref().ObjectPath())

		// Reset
		err = CurrentRepository().Reset(AnyPath)
		require.NoError(t, err)

		// The pack file must no longer be present in the index
		require.Nil(t, CurrentIndex().GetEntry("go.md"))
		// But the raw pack file still exists to speed up the next add
		assert.FileExists(t, indexEntry.Ref().ObjectPath())

		_, err = CurrentDB().GC(false)
		require.NoError(t, err)

		// GC must have reclaimed the pack file
		assert.NoFileExists(t, indexEntry.Ref().ObjectPath())
	})

	t.Run("Unreferenced media", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		// Add
		_, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		entryMarkdown := CurrentIndex().GetEntry("go.md")
		require.NotNil(t, entryMarkdown)
		entryMedia := CurrentIndex().GetEntry("medias/go.svg")
		require.NotNil(t, entryMedia)
		assert.FileExists(t, entryMarkdown.Ref().ObjectPath())
		assert.FileExists(t, entryMedia.Ref().ObjectPath())

		// Rewrite the file without referencing the media
		tr.WriteFile("go.md", `# Go`)
		_, err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// The media must have been removed from the index
		require.Nil(t, CurrentIndex().GetEntry("medias/go.svg"))
		// But the file has not been reclaimed
		assert.FileExists(t, entryMedia.Ref().ObjectPath())

		_, err = CurrentDB().GC(false)
		require.NoError(t, err)

		// GC must have reclaimed the media files but not the markdown file
		assert.FileExists(t, entryMarkdown.Ref().ObjectPath())
		assert.NoFileExists(t, entryMedia.Ref().ObjectPath())
	})

	t.Run("Modified/Unmodified", func(t *testing.T) {
		tr := NewTestRepository(t,
			// Step 1: Add new minimal files
			WithFile("index.md", `
---
tags: programming
---
`),
			WithFile("go.md", `
# Go

## Flashcard: Golang Creators

(Golang) Who are the creators of Golang?

---

Robert Greisemer, Rob Pike, and Ken Thompson.
`))
		resultAdd, err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		require.Len(t, resultAdd.Upserted, 2) // 2 pack files
		require.Len(t, resultAdd.Deleted, 0)

		resultGC, err := CurrentDB().GC(false) // Nothing has changed, nothing to reclaim
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 0)
		assert.Len(t, resultGC.ReclaimedBlobs, 0)

		// Step 2: Modify the file to force a new pack file + new blob
		time.Sleep(1 * time.Millisecond) // Ensure mtimes are different
		tr.WriteFile("go.md", `
# Go

## Flashcard: Golang Creators

(Golang) Who created Golang?

---

Robert Greisemer, Rob Pike, and Ken Thompson.
`)
		resultAdd, err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		require.Len(t, resultAdd.Upserted, 1) // Content changed = new pack file

		resultGC, err = CurrentDB().GC(false) // The old files must not be reclaimed as still not committed
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 0)
		assert.Len(t, resultGC.ReclaimedBlobs, 0)

		// Commit the changes
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		// The old files can now be reclaimed
		resultGC, err = CurrentDB().GC(false)
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 1)
		assert.Len(t, resultGC.ReclaimedBlobs, 1)

		// Step 3: Edit the index file
		// All pack files/blobs to be recreated
		time.Sleep(1 * time.Millisecond) // Ensure mtimes are different
		tr.WriteFile("index.md", `
---
tags: go
---
`)

		resultAdd, err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		require.Len(t, resultAdd.Upserted, 2) // Both content changed = new pack files
		require.Len(t, resultAdd.Deleted, 1)  // The go.md entry has now a tombstone

		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		resultGC, err = CurrentDB().GC(false) // All old files must not be reclaimed as still not committed
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 1) // The go.md pack file has been overwritten as the hash stayed the same
		assert.Len(t, resultGC.ReclaimedBlobs, 1)     // Idem for the associated blob

		// Step 4: Try to add the same unchanged file again with GC between each.
		// Nothing should change, nothing to reclaim.
		for range []int{1, 2, 3} {
			resultAdd, err = CurrentRepository().Add(AnyPath)
			require.NoError(t, err)
			require.Len(t, resultAdd.Upserted, 0) // Nothing has changed, nothing to add
			require.Len(t, resultAdd.Deleted, 0)  // Nothing has changed, nothing to delete

			resultGC, err := CurrentDB().GC(false) // Nothing has changed, nothing to reclaim
			require.NoError(t, err)
			assert.Len(t, resultGC.ReclaimedPackFiles, 0)
			assert.Len(t, resultGC.ReclaimedBlobs, 0)
		}

		// Step 5: Delete go.md
		tr.DeleteFile("go.md")
		resultAdd, err = CurrentRepository().Add(AnyPath)
		require.NoError(t, err)
		require.Len(t, resultAdd.Upserted, 0)
		require.Len(t, resultAdd.Deleted, 1) // The go.md pack file has a tombstone (the associated blob will be garbage-collected later)

		resultGC, err = CurrentDB().GC(false) // Still not committed, nothing to reclaim
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 0)
		assert.Len(t, resultGC.ReclaimedBlobs, 0)

		// Commit the changes
		err = CurrentRepository().Commit(true)
		require.NoError(t, err)

		resultGC, err = CurrentDB().GC(false)
		require.NoError(t, err)
		assert.Len(t, resultGC.ReclaimedPackFiles, 1) // The go.md pack file has been deleted
		assert.Len(t, resultGC.ReclaimedBlobs, 1)     // Idem for the associated blob

		// Step 6: Try to add the same unchanged file again with GC between each.
		// Nothing should change, nothing to reclaim.
		for range []int{1, 2, 3} {
			resultAdd, err = CurrentRepository().Add(AnyPath)
			require.NoError(t, err)
			require.Len(t, resultAdd.Upserted, 0) // Nothing has changed, nothing to add
			require.Len(t, resultAdd.Deleted, 0)  // Nothing has changed, nothing to delete

			resultGC, err := CurrentDB().GC(false) // Nothing has changed, nothing to reclaim
			require.NoError(t, err)
			assert.Len(t, resultGC.ReclaimedPackFiles, 0)
			assert.Len(t, resultGC.ReclaimedBlobs, 0)
		}
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
