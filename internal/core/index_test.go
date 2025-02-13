package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexEntry(t *testing.T) {

	t.Run("String", func(t *testing.T) {
		FreezeOn(t, "2023-01-01 12:30:00")

		entry := &IndexEntry{
			RelativePath: "go.md",
			PackFileOID:  oid.MustParse("1234567890123456789012345678901234567890"),
			MTime:        clock.Now(),
			Size:         1,
			Staged:       false,
		}
		assert.Equal(t, `entry "go.md" (packfile: 1234567890123456789012345678901234567890)`, entry.String())

		entryStaged := &IndexEntry{
			RelativePath:      "go.md",
			PackFileOID:       oid.MustParse("1234567890123456789012345678901234567890"),
			MTime:             clock.Now(),
			Size:              1,
			Staged:            true,
			StagedPackFileOID: oid.MustParse("33f1c9b2e0f94af4ac8c374051d7cf31724140ac"),
			StagedMTime:       clock.Now(),
			StagedSize:        2,
		}
		assert.Equal(t, `entry "go.md" (packfile: 1234567890123456789012345678901234567890 => 33f1c9b2e0f94af4ac8c374051d7cf31724140ac)`, entryStaged.String())

		entryDeleted := &IndexEntry{
			RelativePath:    "go.md",
			PackFileOID:     oid.MustParse("1234567890123456789012345678901234567890"),
			MTime:           clock.Now(),
			Size:            1,
			Staged:          true,
			StagedTombstone: clock.Now(),
		}
		assert.Equal(t, `entry "go.md" (packfile: !1234567890123456789012345678901234567890)`, entryDeleted.String())
	})
}

func TestIndex(t *testing.T) {

	t.Run("Empty Index", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		idx := NewIndex()
		err := idx.Save()
		require.NoError(t, err)
		idx, err = ReadIndex()
		require.NoError(t, err)
		assert.Equal(t, 0, len(idx.Entries))
	})

	t.Run("Empty/Stage/Commit", func(t *testing.T) {
		// Make tests reproductible
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30:00")
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		idx := NewIndex()

		parsedFile := ParseFileFromRelativePath(t, "go.md")
		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Stage the pack file
		err = idx.Stage(packFile)
		require.NoError(t, err)
		assert.Len(t, idx.Entries, 1) // Only 1 pack file staged
		assert.Len(t, idx.Objects, len(packFile.PackObjects))
		entry := idx.GetEntry("go.md")
		assert.NotNil(t, entry)
		entry, ok := idx.GetEntryByPackFileOID(packFile.OID)
		assert.True(t, ok)

		assert.Equal(t, &IndexEntry{
			RelativePath: "go.md",
			PackFileOID:  packFile.OID,
			MTime:        clock.Now(),
			Size:         1,
			// File has just been staged
			Staged:            true,
			StagedPackFileOID: packFile.OID,
			StagedTombstone:   time.Time{},
			StagedMTime:       clock.Now(),
			StagedSize:        1,
		}, entry)

		// Save the index
		err = idx.Save()
		require.NoError(t, err)

		// Reread
		idx, err = ReadIndex()
		require.NoError(t, err)
		assert.Equal(t, 1, len(idx.Entries)) // Entry still there

		// Commit
		require.NoError(t, idx.Commit())

		entry = idx.GetEntry("go.md")
		assert.Equal(t, &IndexEntry{
			RelativePath: "go.md",
			PackFileOID:  packFile.OID,
			MTime:        clock.Now(),
			Size:         1,
			// File has just been staged
			Staged:            false,
			StagedPackFileOID: oid.Nil,
			StagedTombstone:   time.Time{},
			StagedMTime:       time.Time{},
			StagedSize:        0,
		}, entry)
	})

	t.Run("Existing/Stage/Commit", func(t *testing.T) {
		// Make tests reproductible
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30:00")
		SetUpRepositoryFromTempDir(t)

		idx := NewIndex()

		WriteFileFromRelativePath(t, "programming.md", "## Note: Go\n\nGo is a statically typed, compiled high-level general purpose programming language.")

		parsedFile1 := ParseFileFromRelativePath(t, "programming.md")
		packFile1, err := NewPackFileFromParsedFile(parsedFile1)
		require.NoError(t, err)

		// Stage the pack file
		err = idx.Stage(packFile1)
		require.NoError(t, err)
		assert.Len(t, idx.Entries, 1) // Only 1 pack file staged
		assert.Len(t, idx.Objects, 2) // 1 file + 1 note

		// Edit file to add a new note
		WriteFileFromRelativePath(t, "programming.md", "## Note: Go\n\nGo is a statically typed, compiled high-level general purpose programming language.\n\n## Note: Python\n\nPython is a high-level, general-purpose programming language.")

		parsedFile2 := ParseFileFromRelativePath(t, "programming.md")
		packFile2, err := NewPackFileFromParsedFile(parsedFile2)
		require.NoError(t, err)

		// Stage the pack file
		err = idx.Stage(packFile2)
		require.NoError(t, err)
		assert.Len(t, idx.Entries, 1)   // Still the same entry updated with a new pack file
		assert.Len(t, idx.Objects, 2+3) // One more note but old objects are still there

		// Commit
		require.NoError(t, idx.Commit())
		assert.Len(t, idx.Entries, 1)
		assert.Len(t, idx.Objects, 3) // Old entries have been removed
	})

	t.Run("Reset", func(t *testing.T) {
		// Make tests reproductible
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30:00")
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		idx := NewIndex()

		// Create and commit a first pack file
		WriteFileFromRelativePath(t, "go.md", "## Note: Go\n\nGo is a statically typed, compiled high-level general purpose programming language.")
		packFile1 := NewPackFileFromRelativePath(t, "go.md")
		require.NoError(t, idx.Stage(packFile1))
		require.NoError(t, idx.Commit())

		// Create but only stage a second pack file
		WriteFileFromRelativePath(t, "python.md", "## Note: Python\n\nPython is a high-level, general-purpose programming language.")
		packFile2 := NewPackFileFromRelativePath(t, "python.md")
		require.NoError(t, idx.Stage(packFile2))

		entry1 := idx.GetEntry("go.md")
		entry2 := idx.GetEntry("python.md")
		assert.NotNil(t, entry1)
		assert.NotNil(t, entry2)
		// First entry must not be staged
		assert.Equal(t, &IndexEntry{
			RelativePath: "go.md",
			PackFileOID:  packFile1.OID,
			MTime:        clock.Now(),
			Size:         1,
			// File has just been staged
			Staged:            false,
			StagedPackFileOID: oid.Nil,
			StagedTombstone:   time.Time{},
			StagedMTime:       time.Time{},
			StagedSize:        0,
		}, entry1)
		// Second entry must be staged
		assert.Equal(t, &IndexEntry{
			RelativePath: "python.md",
			PackFileOID:  packFile2.OID,
			MTime:        clock.Now(),
			Size:         1,
			// File has just been staged
			Staged:            true,
			StagedPackFileOID: packFile2.OID,
			StagedTombstone:   time.Time{},
			StagedMTime:       clock.Now(),
			StagedSize:        1,
		}, entry2)

		// Reset
		err := idx.Reset(AnyPath)
		require.NoError(t, err)

		entry1 = idx.GetEntry("go.md")
		entry2 = idx.GetEntry("python.md")
		assert.NotNil(t, entry1)
		assert.Nil(t, entry2) // no longer exist as never committed

		// Restage the second pack file and commit
		require.NoError(t, idx.Stage(packFile2))
		require.NoError(t, idx.Commit())

		// No entry must be staged
		entry1 = idx.GetEntry("go.md")
		entry2 = idx.GetEntry("python.md")
		assert.NotNil(t, entry1)
		assert.NotNil(t, entry2)
		assert.False(t, entry1.Staged)
		assert.False(t, entry2.Staged)

		// Recreate a new pack file for python.md
		WriteFileFromRelativePath(t, "python.md", "## Note: Python Lang\n\nPython is a high-level, general-purpose programming language.")
		newPackFile2 := NewPackFileFromRelativePath(t, "python.md")
		require.NoError(t, idx.Stage(newPackFile2))

		// The entry must be staged
		entry2 = idx.GetEntry("python.md")
		assert.NotNil(t, entry2)
		assert.Equal(t, &IndexEntry{
			RelativePath: "python.md",
			PackFileOID:  packFile2.OID,
			MTime:        clock.Now(),
			Size:         1,
			// File has been staged again
			Staged:            true,
			StagedPackFileOID: newPackFile2.OID,
			StagedTombstone:   time.Time{},
			StagedMTime:       clock.Now(),
			StagedSize:        1,
		}, entry2)

		// Reset
		require.NoError(t, idx.Reset(AnyPath))
		// The entry must still be there because committed previously but no longer staged
		entry2 = idx.GetEntry("python.md")
		assert.NotNil(t, entry2)
		assert.False(t, entry2.Staged)
	})

	t.Run("GetParentEntry", func(t *testing.T) {
		// Make tests reproductible
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30:00")
		SetUpRepositoryFromTempDir(t)

		idx := NewIndex()

		WriteFileFromRelativePath(t, "index.md", "")
		WriteFileFromRelativePath(t, "skills/index.md", "# Skills")
		WriteFileFromRelativePath(t, "skills/programming/index.md", "# Programming")
		WriteFileFromRelativePath(t, "skills/programming/go.md", "# Go")
		WriteFileFromRelativePath(t, "skills/running.md", "# Running")
		WriteFileFromRelativePath(t, "projects/the-notewriter.md", "# The NoteWriter")
		WriteFileFromRelativePath(t, "todo.md", "# TODO")

		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "index.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/index.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/programming/index.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/programming/go.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/running.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "projects/the-notewriter.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "todo.md")))

		require.NoError(t, idx.Commit())

		assert.Nil(t, idx.GetParentEntry("index.md"))
		assert.Equal(t, "index.md", idx.GetParentEntry("skills/index.md").RelativePath)
		assert.Equal(t, "skills/index.md", idx.GetParentEntry("skills/programming/index.md").RelativePath)
		assert.Equal(t, "skills/programming/index.md", idx.GetParentEntry("skills/programming/go.md").RelativePath)
		assert.Equal(t, "skills/index.md", idx.GetParentEntry("skills/running.md").RelativePath)
		assert.Equal(t, "index.md", idx.GetParentEntry("projects/the-notewriter.md").RelativePath)
		assert.Equal(t, "index.md", idx.GetParentEntry("todo.md").RelativePath)
	})

	t.Run("Tombstone", func(t *testing.T) {
		// Make tests reproductible
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30:00")
		SetUpRepositoryFromTempDir(t)

		idx := NewIndex()

		WriteFileFromRelativePath(t, "go.md", "## Note: Go\n\nGo is a statically typed, compiled high-level general purpose programming language.")
		WriteFileFromRelativePath(t, "python.md", "## Note: Python\n\nPython is a high-level, general-purpose programming language.")

		packFile1 := NewPackFileFromRelativePath(t, "go.md")
		packFile2 := NewPackFileFromRelativePath(t, "python.md")

		// Stage and commit
		require.NoError(t, idx.Stage(packFile1))
		require.NoError(t, idx.Stage(packFile2))
		require.NoError(t, idx.Commit())
		countObjectsBefore := len(idx.Objects)

		// Delete the first file
		idx.SetTombstone(packFile1.FileRelativePath)
		countObjectsAfter := len(idx.Objects)
		assert.NotNil(t, idx.GetEntry("go.md"))                             // Still there...
		assert.Equal(t, clock.Now(), idx.GetEntry("go.md").StagedTombstone) // ...with a tombstone
		assert.Equal(t, countObjectsBefore, countObjectsAfter)              // no change in the number of objects as the pack file is still there with a tombstone

		// Commit
		require.NoError(t, idx.Commit())
		assert.Len(t, idx.Entries, 1)        // Only 1 pack file with 1 note remaining
		assert.Nil(t, idx.GetEntry("go.md")) // The entry has been removed
	})

	t.Run("Walk", func(t *testing.T) {
		SetUpRepositoryFromTempDir(t)

		idx := NewIndex()

		// Create and stage some pack files
		WriteFileFromRelativePath(t, "index.md", "# Index")
		WriteFileFromRelativePath(t, "skills/programming/go.md", "# Go")
		WriteFileFromRelativePath(t, "skills/running.md", "# Running")
		WriteFileFromRelativePath(t, "projects/the-notewriter.md", "# The NoteWriter")

		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "index.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/programming/go.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "skills/running.md")))
		require.NoError(t, idx.Stage(NewPackFileFromRelativePath(t, "projects/the-notewriter.md")))

		require.NoError(t, idx.Commit())

		t.Run("Match all entries", func(t *testing.T) {
			var matchedEntries []*IndexEntry
			err := idx.Walk(AnyPath, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
				matchedEntries = append(matchedEntries, entry)
				return nil
			})
			require.NoError(t, err)
			assert.Len(t, matchedEntries, 4)
		})

		t.Run("Match specific entries", func(t *testing.T) {
			var matchedEntries []*IndexEntry
			err := idx.Walk(PathSpecs{"skills/*"}, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
				matchedEntries = append(matchedEntries, entry)
				return nil
			})
			require.NoError(t, err)
			assert.Len(t, matchedEntries, 2)
			assert.Equal(t, "skills/programming/go.md", matchedEntries[0].RelativePath)
			assert.Equal(t, "skills/running.md", matchedEntries[1].RelativePath)
		})

		t.Run("Match no entries", func(t *testing.T) {
			var matchedEntries []*IndexEntry
			err := idx.Walk(PathSpecs{"nonexistent/*"}, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
				matchedEntries = append(matchedEntries, entry)
				return nil
			})
			require.NoError(t, err)
			assert.Len(t, matchedEntries, 0)
		})

		t.Run("Error in callback", func(t *testing.T) {
			expectedErr := fmt.Errorf("callback error")
			err := idx.Walk(AnyPath, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
				return expectedErr
			})
			assert.Equal(t, expectedErr, err)
		})
	})

	t.Run("Modified", func(t *testing.T) {
		SetUpRepositoryFromTempDir(t)
		FreezeNow(t)

		idx := NewIndex()

		// Create and stage a pack file
		WriteFileFromRelativePath(t, "go.md", "## Note: Go\n\nGo is a statically typed, compiled high-level general purpose programming language.")
		packFile := NewPackFileFromRelativePath(t, "go.md")
		require.NoError(t, idx.Stage(packFile))
		require.NoError(t, idx.Commit())

		// File not in index
		assert.True(t, idx.Modified("python.md", clock.Now()))

		// File in index, not modified
		assert.False(t, idx.Modified("go.md", clock.Now()))

		// File in index, modified
		assert.True(t, idx.Modified("go.md", clock.Now().Add(1*time.Hour)))
	})

	t.Run("ShortOID", func(t *testing.T) {
		idx := NewIndex()

		packFile := &PackFile{
			OID:              "bb0ef4484a60459a811f83bbd64129eb51051cf3",
			FileRelativePath: "test.md",
			FileMTime:        clock.Now(),
			FileSize:         1,
			CTime:            clock.Now(),
		}

		packFile.AppendObject(NewTestGoLink(packFile, "1be4098fdc2549a9b78169533076e5540758fa8f"))
		packFile.AppendObject(NewTestGoLink(packFile, "96eefc2919a8491aacd3f8f1d348cb03740140bd"))
		packFile.AppendBlob(NewTestBlob(packFile, "c38d88ca474f4376ab871a363052ffc99f7b5fff"))
		idx.Stage(packFile)

		tests := []struct {
			name     string
			oid      oid.OID
			expected string
		}{
			{
				name:     "unique prefix",
				oid:      oid.MustParse("49b7fdda50844a2685403169dd8b83e38007eed4"), // completetly different
				expected: "49b7",                                                    // 4 characters at minimum
			},
			{
				name:     "minimum length",
				oid:      oid.MustParse("c38d89ca474f4376ab871a363052ffc99f7b5fff"), // 5 first characters are the same
				expected: "c38d89",                                                  // As many characters as required to be unique
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := idx.ShortOID(tt.oid)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

}

func TestIndexDiff(t *testing.T) {

	t.Run("Empty", func(t *testing.T) {
		idx1 := NewIndex()
		idx1.Entries = []*IndexEntry{
			CreateCommitedEntry("go.md", oid.Test("12345")),
			CreateStagedEntry("python.md", oid.Test("22222"), oid.Test("33333")),
			CreateDeletedEntry("java.md", oid.Test("44444")),
		}
		idx1.Blobs = []*IndexBlob{
			CreateIndexBlob(oid.Test("b22222"), oid.Test("22222")),
			CreateIndexBlob(oid.Test("b33333"), oid.Test("33333")),
		}
		idx2 := NewIndex()

		// idx2 is empty, so nothing is missing
		diff := idx1.Diff(idx2)
		assert.Empty(t, diff.MissingPackFiles)
		assert.Empty(t, diff.MissingBlobs)

		// idx2 is empty, so everything is missing
		diff = idx2.Diff(idx1)
		assert.ElementsMatch(t, diff.MissingPackFiles.OIDs(), []oid.OID{
			oid.Test("12345"),
			oid.Test("22222"),
			oid.Test("44444"),
		})
		assert.ElementsMatch(t, diff.MissingBlobs.OIDs(), []oid.OID{
			oid.Test("b22222"),
		})
		// Deleted changes are ignored (still not committed)
		// Staged changes are ignored too
	})

	t.Run("Same", func(t *testing.T) {
		idx1 := NewIndex()
		idx1.Entries = []*IndexEntry{
			CreateCommitedEntry("go.md", oid.Test("12345")),
			CreateStagedEntry("python.md", oid.Test("22222"), oid.Test("33333")),
			CreateDeletedEntry("java.md", oid.Test("44444")),
		}
		idx1.Blobs = []*IndexBlob{
			CreateIndexBlob(oid.Test("b22222"), oid.Test("22222")),
			CreateIndexBlob(oid.Test("b33333"), oid.Test("33333")),
		}

		idx2 := NewIndex()
		idx2.Entries = []*IndexEntry{
			CreateCommitedEntry("go.md", oid.Test("12345")),
			CreateStagedEntry("python.md", oid.Test("22222"), oid.Test("33333")),
			CreateDeletedEntry("java.md", oid.Test("44444")),
		}
		idx2.Blobs = []*IndexBlob{
			CreateIndexBlob(oid.Test("b22222"), oid.Test("22222")),
			CreateIndexBlob(oid.Test("b33333"), oid.Test("33333")),
		}

		// Both indexes are the same, so nothing is missing
		diff := idx1.Diff(idx2)
		assert.Empty(t, diff.MissingPackFiles)
		assert.Empty(t, diff.MissingBlobs)

		diff = idx2.Diff(idx1)
		assert.Empty(t, diff.MissingPackFiles)
		assert.Empty(t, diff.MissingBlobs)
	})

	t.Run("Different", func(t *testing.T) {
		idx1 := NewIndex()
		idx1.Entries = []*IndexEntry{
			CreateCommitedEntry("go.md", oid.Test("12345")),
			CreateStagedEntry("python.md", oid.Test("22222"), oid.Test("33333")),
		}
		idx1.Blobs = []*IndexBlob{
			CreateIndexBlob(oid.Test("b22222"), oid.Test("22222")),
		}

		idx2 := NewIndex()
		idx2.Entries = []*IndexEntry{
			CreateCommitedEntry("go.md", oid.Test("12345")),
			CreateStagedEntry("java.md", oid.Test("44444"), oid.Test("55555")),
		}
		idx2.Blobs = []*IndexBlob{
			CreateIndexBlob(oid.Test("b44444"), oid.Test("44444")),
		}

		// idx1 is missing java.md and its blob
		diff := idx1.Diff(idx2)
		assert.ElementsMatch(t, diff.MissingPackFiles.OIDs(), []oid.OID{
			oid.Test("44444"),
		})
		assert.ElementsMatch(t, diff.MissingBlobs.OIDs(), []oid.OID{
			oid.Test("b44444"),
		})

		// idx2 is missing python.md and its blob
		diff = idx2.Diff(idx1)
		assert.ElementsMatch(t, diff.MissingPackFiles.OIDs(), []oid.OID{
			oid.Test("22222"),
		})
		assert.ElementsMatch(t, diff.MissingBlobs.OIDs(), []oid.OID{
			oid.Test("b22222"),
		})
	})

}

func TestIndexOnDisk(t *testing.T) {
	SetUpRepositoryFromTempDir(t)

	idx := NewIndex()

	// Create and stage some pack files
	WriteFileFromRelativePath(t, "index.md", "# Index")
	WriteFileFromRelativePath(t, "skills/programming/go.md", "# Go")
	WriteFileFromRelativePath(t, "skills/running.md", "# Running")
	WriteFileFromRelativePath(t, "projects/the-notewriter.md", "# The NoteWriter")

	packFile1 := NewPackFileFromRelativePath(t, "index.md")
	packFile2 := NewPackFileFromRelativePath(t, "skills/programming/go.md")
	packFile3 := NewPackFileFromRelativePath(t, "skills/running.md")
	packFile4 := NewPackFileFromRelativePath(t, "projects/the-notewriter.md")
	require.NoError(t, packFile1.Save())
	require.NoError(t, packFile2.Save())
	require.NoError(t, packFile3.Save())
	require.NoError(t, packFile4.Save())

	require.NoError(t, idx.Stage(packFile1))
	require.NoError(t, idx.Stage(packFile2))
	require.NoError(t, idx.Stage(packFile3))
	require.NoError(t, idx.Stage(packFile4))
	require.NoError(t, idx.Commit())

	// Read a pack file on disk
	readPackFile1, err := idx.ReadPackFile(packFile1.OID)
	require.NoError(t, err)
	assert.Equal(t, packFile1.OID, readPackFile1.OID)

	// Read a pack object on disk
	packObject1 := packFile1.PackObjects[0]
	assert.Equal(t, "file", packObject1.Kind)
	readPackObject1, err := idx.ReadPackObject(packObject1.OID) // TODO now debug
	require.NoError(t, err)
	assert.Equal(t, packObject1.OID, readPackObject1.OID)

	// Read an object from the pack object on disk
	object1, err := idx.ReadObject(packObject1.OID)
	require.NoError(t, err)
	assert.IsType(t, &File{}, object1)

	// Read a blob from the pack object on disk
	blobRef1 := packFile1.BlobRefs[0]
	blob1, err := idx.ReadBlob(blobRef1.OID)
	require.NoError(t, err)
	assert.Equal(t, blobRef1.OID, blob1.OID)

	// Read a blob content from the pack object on disk
	data1, err := idx.ReadBlobData(blobRef1.OID)
	require.NoError(t, err)
	assert.Equal(t, "# Index", string(data1))
}

func TestShortenToUniquePrefix(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		knownValues []string
		expected    string
	}{
		{
			name:        "no prefix possible",
			value:       "abcdef",
			knownValues: []string{"abc", "abcd", "abcde"},
			expected:    "abcdef",
		},
		{
			name:        "unique prefix possible",
			value:       "abcdef",
			knownValues: []string{"abc", "abcd"},
			expected:    "abcde",
		},
		{
			name:        "multiple prefixes possible",
			value:       "abcdef",
			knownValues: []string{"abc", "abcd", "ab"},
			expected:    "abcde",
		},
		{
			name:        "value already present",
			value:       "abcdef",
			knownValues: []string{"abc", "abcd", "abcdef"},
			expected:    "abcdef",
		},
		{
			name:        "known values very different",
			value:       "abcdef",
			knownValues: []string{"xyz", "uvw"},
			expected:    "a",
		},
		{
			name:        "no known values",
			value:       "abcdef",
			knownValues: []string{},
			expected:    "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenToUniquePrefix(tt.value, tt.knownValues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortedEntries(t *testing.T) {

	t.Run("Empty Index", func(t *testing.T) {
		idx := NewIndex()
		sortedEntries := idx.SortedEntries()
		assert.Empty(t, sortedEntries)
	})

	t.Run("Single Entry", func(t *testing.T) {
		idx := NewIndex()
		entry := CreateCommitedEntry("go.md", oid.Test("12345"))
		idx.Entries = []*IndexEntry{
			entry,
		}
		sortedEntries := idx.SortedEntries()
		require.Len(t, sortedEntries, 1)
		assert.Equal(t, "go.md", sortedEntries[0].RelativePath)
	})

	t.Run("Multiple Entries", func(t *testing.T) {
		idx := NewIndex()
		idx.Entries = []*IndexEntry{
			CreateCommitedEntry("b.md", oid.Test("12345")),
			CreateCommitedEntry("a.md", oid.Test("67890")),
			CreateCommitedEntry("c.md", oid.Test("54321")),
		}

		sortedEntries := idx.SortedEntries()
		require.Len(t, sortedEntries, 3)
		assert.Equal(t, "a.md", sortedEntries[0].RelativePath)
		assert.Equal(t, "b.md", sortedEntries[1].RelativePath)
		assert.Equal(t, "c.md", sortedEntries[2].RelativePath)

		// Original entries must not have been modified
		assert.Equal(t, "b.md", idx.Entries[0].RelativePath)
		assert.Equal(t, "a.md", idx.Entries[1].RelativePath)
		assert.Equal(t, "c.md", idx.Entries[2].RelativePath)
	})

	t.Run("Entries with Subdirectories", func(t *testing.T) {
		idx := NewIndex()
		idx.Entries = []*IndexEntry{
			CreateCommitedEntry("dir/b.md", oid.Test("12345")),
			CreateCommitedEntry("a.md", oid.Test("67890")),
			CreateCommitedEntry("dir/a.md", oid.Test("54321")),
		}

		sortedEntries := idx.SortedEntries()
		require.Len(t, sortedEntries, 3)
		assert.Equal(t, "a.md", sortedEntries[0].RelativePath)
		assert.Equal(t, "dir/a.md", sortedEntries[1].RelativePath)
		assert.Equal(t, "dir/b.md", sortedEntries[2].RelativePath)
	})
}

/* Helpers */

func NewTestPackFile(newOID oid.OID) *PackFile {
	return &PackFile{
		OID:              newOID,
		FileRelativePath: fmt.Sprintf("%s.md", newOID),
		FileMTime:        clock.Now(),
		FileSize:         1,
		CTime:            clock.Now(),
	}
}

func NewTestGoLink(packFile *PackFile, newOID oid.OID) *GoLink {
	return &GoLink{
		OID:          newOID,
		PackFileOID:  packFile.OID,
		NoteOID:      oid.Nil,
		RelativePath: packFile.FileRelativePath,
		Text:         markdown.Document(newOID),
		URL:          fmt.Sprintf("https//%s.fr", newOID),
		Title:        newOID.String(),
		GoName:       newOID.String(),
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
	}
}

func NewTestBlob(packFile *PackFile, newOID oid.OID) *BlobRef {
	return &BlobRef{
		OID:        newOID,
		MimeType:   "application/octet-stream",
		Attributes: nil,
		Tags:       nil,
	}
}

func CreateCommitedEntry(relativePath string, newOID oid.OID) *IndexEntry {
	return &IndexEntry{
		RelativePath: relativePath,
		PackFileOID:  newOID,
		MTime:        clock.Now(),
		Size:         1,
		Staged:       false,
	}
}

func CreateNewStagedEntry(relativePath string, newOID oid.OID) *IndexEntry {
	return &IndexEntry{
		RelativePath:      relativePath,
		PackFileOID:       newOID,
		MTime:             clock.Now(),
		Size:              1,
		Staged:            true,
		StagedPackFileOID: newOID,
		StagedMTime:       clock.Now(),
		StagedSize:        1,
		StagedTombstone:   time.Time{},
	}
}

func CreateStagedEntry(relativePath string, oldOID, newOID oid.OID) *IndexEntry {
	return &IndexEntry{
		RelativePath:      relativePath,
		PackFileOID:       oldOID,
		MTime:             clock.Now(),
		Size:              1,
		Staged:            true,
		StagedPackFileOID: newOID,
		StagedMTime:       clock.Now(),
		StagedSize:        1,
		StagedTombstone:   time.Time{},
	}
}

func CreateDeletedEntry(relativePath string, oldOID oid.OID) *IndexEntry {
	return &IndexEntry{
		RelativePath:      relativePath,
		PackFileOID:       oldOID,
		MTime:             clock.Now(),
		Size:              1,
		Staged:            true,
		StagedPackFileOID: oid.Nil,
		StagedMTime:       clock.Now(),
		StagedSize:        0,
		StagedTombstone:   clock.Now(),
	}
}

func CreateIndexBlob(blobOID, packFileOID oid.OID) *IndexBlob {
	return &IndexBlob{
		OID:         blobOID,
		MimeType:    "application/octet-stream",
		PackFileOID: packFileOID,
	}
}

func CreatePackFileRef(oid oid.OID, relativePath string) PackFileRef {
	return PackFileRef{
		OID:          oid,
		RelativePath: relativePath,
		CTime:        clock.Now(),
	}
}

func CreateBlobRef(oid oid.OID) BlobRef {
	return BlobRef{
		OID:        oid,
		MimeType:   "application/octet-stream",
		Attributes: nil,
		Tags:       nil,
	}
}
