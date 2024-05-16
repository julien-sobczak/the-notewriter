package core

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// reOID matches the Git commit ID format
var reOID = regexp.MustCompile(`\w{40}`)

func TestNewOID(t *testing.T) {
	oid1 := NewOID()
	oid2 := NewOID()
	require.NotEqual(t, oid1, oid2)
	assert.Regexp(t, reOID, oid1)
}

func TestNewOIDFromBytes(t *testing.T) {
	bytes1 := []byte{97, 98, 99, 100, 101, 102}
	bytes2 := []byte{98, 98, 99, 100, 101, 102}
	oid1 := NewOIDFromBytes(bytes1)
	oid2 := NewOIDFromBytes(bytes2)
	require.NotEqual(t, oid1, oid2)
	require.Equal(t, oid1, NewOIDFromBytes(bytes1)) // Does not change
	assert.Regexp(t, reOID, oid1)
}

func TestCommitGraph(t *testing.T) {

	t.Run("New CommitGraph", func(t *testing.T) {
		now := FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		cg := NewCommitGraph()
		assert.Equal(t, now, cg.UpdatedAt)

		// A succession of commits
		now = FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 14, 30, 0, time.UTC))
		err := cg.AppendCommit(NewCommitWithOID("a757e67f5ae2a8df3a4634c96c16af5c8491bea2"))
		require.NoError(t, err)
		err = cg.AppendCommit(NewCommitWithOID("a04d20dec96acfc2f9785802d7e3708721005d5d"))
		require.NoError(t, err)
		err = cg.AppendCommit(NewCommitWithOID("52d614e255d914e2f6022689617da983381c27a3"))
		require.NoError(t, err)
		assert.Equal(t, now, cg.UpdatedAt)

		_, err = cg.LastCommitsFrom("unknown")
		require.ErrorContains(t, err, "unknown commit")
		commits, err := cg.LastCommitsFrom("a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		require.EqualValues(t, []*Commit{
			{
				OID:   "52d614e255d914e2f6022689617da983381c27a3",
				CTime: now,
				MTime: now,
			},
		}, commits)

		buf := new(bytes.Buffer)
		err = cg.Write(buf)
		require.NoError(t, err)
		cgYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
updated_at: 2023-01-01T01:14:30Z
commits:
    - oid: a757e67f5ae2a8df3a4634c96c16af5c8491bea2
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: a04d20dec96acfc2f9785802d7e3708721005d5d
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: 52d614e255d914e2f6022689617da983381c27a3
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
`), strings.TrimSpace(cgYAML))
	})

	t.Run("Existing CommitGraph", func(t *testing.T) {
		in, err := os.CreateTemp("", "commit-graph1")
		require.NoError(t, err)
		defer os.Remove(in.Name())
		out, err := os.CreateTemp("", "commit-graph2")
		require.NoError(t, err)
		defer os.Remove(out.Name())

		in.WriteString(`updated_at: 2023-01-01T01:14:30Z
commits:
    - oid: a757e67f5ae2a8df3a4634c96c16af5c8491bea2
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: a04d20dec96acfc2f9785802d7e3708721005d5d
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: 52d614e255d914e2f6022689617da983381c27a3
      ctime: 2023-01-01T01:14:30Z
      mtime: 2023-01-01T01:14:30Z
      packfiles: []
`) // Caution: spaces are important as we compare hashes at the end of the test
		in.Close()

		// Read in
		in, err = os.Open(in.Name())
		require.NoError(t, err)
		cg := new(CommitGraph)
		err = cg.Read(in)
		in.Close()
		require.NoError(t, err)
		assert.Len(t, cg.Commits, 3)
		assert.Equal(t, cg.Commits[0].OID, "a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		assert.Equal(t, cg.Commits[1].OID, "a04d20dec96acfc2f9785802d7e3708721005d5d")
		assert.Equal(t, cg.Commits[2].OID, "52d614e255d914e2f6022689617da983381c27a3")

		// Write out
		err = cg.Write(out)
		require.NoError(t, err)
		out.Close()

		// Files must match
		hashIn, _ := helpers.HashFromFile(in.Name())
		hashOut, _ := helpers.HashFromFile(out.Name())
		assert.Equal(t, hashIn, hashOut)
	})

	t.Run("Diff", func(t *testing.T) {
		cg1 := NewCommitGraph()
		cg2 := NewCommitGraph()

		c1 := NewCommit()
		c2 := NewCommitWithOID("e71e62ef-5e8a-4446-9833-329271f7fc41")
		c2.PackFiles = []*PackFileRef{
			NewPackFileRefWithOID("b679d872-d529-4d74-845f-80ef1439e66a"),
			NewPackFileRefWithOID("ae38fc3e-4b0e-416a-af09-816d78f4b9ec"),
			NewPackFileRefWithOID("9d4571bc-bed9-48fc-a75b-51a0ac387247"),
		}
		c2Compressed := NewCommitWithOID("e71e62ef-5e8a-4446-9833-329271f7fc41")
		c2Compressed.PackFiles = []*PackFileRef{
			NewPackFileRefWithOID("9d4571bc-bed9-48fc-a75b-51a0ac387247"),
			NewPackFileRefWithOID("250e8fac-baad-40d8-b1f0-7c1d6bb06e86"),
		} // two first pack files were replaced by a new one
		c3 := NewCommit()
		c4 := NewCommit()
		c5 := NewCommit()

		// Start with a common commit
		err := cg1.AppendCommit(c1)
		require.NoError(t, err)
		err = cg2.AppendCommit(c1)
		require.NoError(t, err)

		// Add a common commit with pack files merged by gc only in cg1
		err = cg1.AppendCommit(c2Compressed)
		require.NoError(t, err)
		err = cg2.AppendCommit(c2)
		require.NoError(t, err)

		// New commit only in cg1
		err = cg1.AppendCommit(c3)
		require.NoError(t, err)

		// New commits only in cg2
		err = cg2.AppendCommit(c4)
		require.NoError(t, err)
		err = cg2.AppendCommit(c5)
		require.NoError(t, err)

		// Compare
		diffA := cg1.Diff(cg2)
		diffB := cg2.Diff(cg1)

		// Unique commits in each graph must be reported
		assert.EqualValues(t, []*Commit{c4, c5}, diffA.MissingCommits)
		assert.EqualValues(t, []*Commit{c3}, diffB.MissingCommits)
		// Merge pack files must be reported in addition to the new pack file
		assert.EqualValues(t, []string{"250e8fac-baad-40d8-b1f0-7c1d6bb06e86"}, diffB.MissingPackFiles.OIDs())
		assert.EqualValues(t, []string{
			"b679d872-d529-4d74-845f-80ef1439e66a",
			"ae38fc3e-4b0e-416a-af09-816d78f4b9ec",
		}, diffB.ObsoletePackFiles.OIDs())
	})

}

func TestObjectData(t *testing.T) {
	SetUpRepositoryFromTempDir(t)

	fileSrc := NewEmptyFile("todo.md")
	noteParsedSrc := MustParseNote("## TODO: Backlog\n\n* [ ] Test ObjectData", "")
	noteSrc := NewNote(fileSrc, nil, noteParsedSrc)
	dataSrc, err := NewObjectData(noteSrc)
	require.NoError(t, err)

	// Marshall YAML
	txt, err := yaml.Marshal(dataSrc)
	require.NoError(t, err)
	reBase64 := regexp.MustCompile(`^[A-Za-z0-9+=/]*$`)
	assert.Regexp(t, reBase64, strings.TrimSpace(string(txt)))

	// Unmarshall YAML
	var dataDest ObjectData
	err = yaml.Unmarshal(txt, &dataDest)
	require.NoError(t, err)

	// Unmarshall
	noteDest := new(Note)
	err = dataDest.Unmarshal(noteDest)
	require.NoError(t, err)
	assert.Equal(t, "TODO: Backlog", noteDest.Title)
}

func TestPackFile(t *testing.T) {

	// Make tests reproductible
	UseFixedOID(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("New pack file", func(t *testing.T) {
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
		require.NoError(t, err)

		packFileSrc := NewPackFile()
		// add a bunch of objects
		packFileSrc.AppendObject(f.GetNotes()[0])
		packFileSrc.AppendObject(f.GetFlashcards()[0])

		// Marshmall YAML
		buf := new(bytes.Buffer)
		err = packFileSrc.Write(buf)
		require.NoError(t, err)
		cYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
ctime: 2023-01-01T01:12:30Z
mtime: 2023-01-01T01:12:30Z
objects:
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: note
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: 'note "Reference: Golang History" [93267c32147a4ab7a1100ce82faab56a99fca1cd]'
      data: eJzEUsFq20AQvesrBvngBCJrJTu2vNimNxd6KSGnlqKMtKPVYmlXrNZxA/34Ikt24jYNlAa6t5l58+bN2zFKcFhO4/kin8bRbIEzzBYYRYzllMQFYnY7x+WyyDHKhddWe8lBmsBSQZZ0ToE0FWoZlKp1xj55haoo/TvSBi1pl2rjhk7f93ZKCw7nKZ5TriIO47tThsP2OBg+9oPHXmW0TAfcZc1rS2PdH2qWKnTqkdIGXdntNqmFd1A7VSm94zCWZvTGUHTOqmzvqOUeAEBr9rbDlc41LQ9D0pOOqyGhcGKsDLso3Jr0qrFGWqxrpWXace5R0vWRw6Ec2LoXgDQvgpPPR+CrCz23n1uf2yqliUPi5Ua7znSLBw4/giPqYTSgHrw+/vCP2ww8X3uB365ONNJMBD2GwuQh+CNpQmn8azhgC4JaJTUJyJ7gzmRkHWwtqZZqsjddBj6rHd0AagGfSMN9aeqmNRrQwdYYWREoDTFji8l5xRLbkgOjDJN5nmSMJTmL4ziZL6m4nd4mYpZglsSzqCjyZLi0tEa7E+agOYxHv316DyldXXFYldHmsr4Ky2gzYBx9d2d/f/mnLrW+eGfJL6b/H/PGz+71WzabFUJpqVj7rwjxwaGV5NZ+mlWod35/muuTvsGhVYib91a6CpvNWWxv9+Dzu9+TJXQkUnQcYhZPAxYFLLpnEY9iPmVfvH0j3gb8DAAA//9fO8O2
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: flashcard
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: flashcard "Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]
      data: eJykkE9r20AQxe/6FFOd2gXZu5Jq14us4lMvPRYKLUWMtWtJWNKI1TTJIR8+6E+MkkCwCcxpeO/xe48qo2EXhZttHoUq3mKMxy0qJWVuv4UnxOPXDe52pxxVbry+JMcZV1xbDT+oxraAn1SQd6pqm92W1RLfanG2Rq7ubNYhlxoKWjXGYyx67QEABDCQOGo5a9CdDd23Gn6XyGDI9sClBSFm6poKEgKc7ZztbcvfvSPm54XvMRgzDyBEQV1pnRArb3x9+jtU/vd5YI9lfM3JLzNXyU2tIenSl1hJz47aIl3AJev5t2BM1l06cU45M2PSpYdLxAR7ca9Gz7MsaaypEKgye/9adB+w5r0/dPZhnY55Uxm2D/x64EWDN+NO+suwE+mHRs2dRbYmQ9YQyjAKpAqk+iWVVqGO5B/vf2feFzwFAAD//+eX4GM=
`), strings.TrimSpace(cYAML))

		// Unmarshall YAML
		packFileDest := new(PackFile)
		err = packFileDest.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd", packFileDest.OID)
		require.Len(t, packFileDest.PackObjects, 2)

		// Unmarshall the note
		noteDest := new(Note)
		err = packFileDest.PackObjects[0].Data.Unmarshal(noteDest)
		require.NoError(t, err)
		assert.Equal(t, "Reference: Golang History", noteDest.Title)

		// Unmarshall the note
		flashcardDest := new(Flashcard)
		err = packFileDest.PackObjects[1].Data.Unmarshal(flashcardDest)
		require.NoError(t, err)
		assert.Equal(t, "Golang Logo", flashcardDest.ShortTitle)

		require.EqualValues(t, packFileSrc, packFileDest)

		// Unmarshall a single object by OID
		noteCopy := new(Note)
		err = packFileDest.UnmarshallObject(packFileDest.PackObjects[0].OID, noteCopy)
		require.NoError(t, err)
		require.EqualValues(t, noteDest, noteCopy)
	})

}

func TestIndex(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		// Make tests reproductible
		UseSequenceOID(t)
		now := FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		idx := NewIndex()

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
		require.NoError(t, err)

		// Add a bunch of objects
		noteExample := f.GetNotes()[0]
		flashcardExample := f.GetFlashcards()[0]
		idx.StageObject(noteExample)
		idx.StageObject(flashcardExample)

		// Create a new file
		err = os.WriteFile(filepath.Join(root, "python.md"), []byte(`# Python

## Flashcard: Python's creator

Who invented Python?
---
Guido van Rossum
`), 0644)
		require.NoError(t, err)

		// Stage a new file
		f, err = NewFileFromPath(nil, filepath.Join(root, "python.md"))
		require.NoError(t, err)
		idx.StageObject(f)

		// Create a new commit
		newCommit, newPackFiles := idx.CreateCommitFromStagingArea()
		assert.NotEmpty(t, newCommit.OID)
		assert.NotEmpty(t, newPackFiles)
		assert.Equal(t, now, newCommit.CTime)
		require.Len(t, newCommit.PackFiles, 1)
		require.Len(t, newPackFiles, 1)
		require.Len(t, newPackFiles[0].PackObjects, 3)

		// Search a single object
		commitOID, ok := idx.FindCommitContaining(noteExample.OID)
		require.True(t, ok)
		assert.Equal(t, newCommit.OID, commitOID)
		packFileOID, ok := idx.FindPackFileContaining(noteExample.OID)
		require.True(t, ok)
		assert.Equal(t, newPackFiles[0].OID, packFileOID) // Must be in first pack file logically
	})

	t.Run("Large Staging Area", func(t *testing.T) {
		// Make tests reproductible
		UseSequenceOID(t)
		root := SetUpRepositoryFromTempDir(t)

		// Create a large file containing many notes
		var newFileContent bytes.Buffer
		newFileContent.WriteString("# New File\n\n")
		for i := 0; i < MaxObjectsPerPackFileDefault+1; i++ {
			newFileContent.WriteString(fmt.Sprintf("## Note: Test %d\n\nBlabla\n\n", i+1))
		}
		newFilePath := filepath.Join(root, "large.md")
		err := os.WriteFile(newFilePath, newFileContent.Bytes(), 0644)
		require.NoError(t, err)

		idx := NewIndex()

		// Stage the new file
		newFile, err := NewFileFromPath(nil, newFilePath)
		require.NoError(t, err)
		idx.StageObject(newFile)
		for _, subObject := range newFile.SubObjects() {
			idx.StageObject(subObject)
		}

		// Create a new commit
		newCommit, newPackFiles := idx.CreateCommitFromStagingArea()
		require.Len(t, newCommit.PackFiles, 2) // All notes exceed the limit per pack file
		require.Len(t, newPackFiles[0].PackObjects, MaxObjectsPerPackFileDefault)
		require.Len(t, newPackFiles[1].PackObjects, 2)
	})

	t.Run("Save on disk", func(t *testing.T) {
		// Make tests reproductible
		UseSequenceOID(t)

		root := SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
		require.NoError(t, err)

		idx := NewIndex()
		// add a bunch of objects
		idx.StageObject(f.GetNotes()[0])
		idx.StageObject(f.GetFlashcards()[0])

		err = idx.Save()
		require.NoError(t, err)

		idx = ReadIndex()
		assert.Equal(t, 2, idx.StagingArea.CountByState(Added))
	})

	t.Run("Diff", func(t *testing.T) {
		// The two indices we will compare
		i1 := NewIndex()
		i2 := NewIndex()

		// A few files to create pack files and commits
		python := NewEmptyFile("python.md")
		golang := NewEmptyFile("go.md")
		english := NewEmptyFile("english.md")
		programming := NewEmptyFile("programming.md")
		linux := NewEmptyFile("linux.md")

		// Group files into different pack files...
		p1 := NewPackFile()
		p1.AppendObject(python)
		p1.AppendObject(golang)

		p2 := NewPackFile()
		p2.AppendObject(english)

		p3 := NewPackFile()
		p3.AppendObject(programming)
		p3.AppendObject(linux)

		// ... and different commits
		c1 := NewCommitFromPackFiles(p1, p2)
		c2 := NewCommitFromPackFiles(p3)

		// Append c1 to both index
		i1.AppendPackFile(c1.OID, p1)
		i1.AppendPackFile(c1.OID, p2)
		i2.AppendPackFile(c1.OID, p1)
		i2.AppendPackFile(c1.OID, p2)

		// Append c2 to only i1
		i1.AppendPackFile(c2.OID, p3)

		// Append orphans
		orphanBlob1 := NewIndexOrphanBlobWithOID("a1d1a55c-5f7f-4203-83b2-b5ece49e9f90")
		orphanPackFile1 := NewIndexOrphanPackFileWithOID("1b74bc4c-0cd8-41b8-9ff0-28dac59ec101")
		orphanPackFile2 := NewIndexOrphanPackFileWithOID("56df3bf6-4813-41e2-8c20-22d6cfb3dd64")

		i1.OrphanBlobs = []*IndexOrphanBlob{orphanBlob1}
		i1.OrphanPackFiles = []*IndexOrphanPackFile{orphanPackFile1, orphanPackFile2}
		i2.OrphanPackFiles = []*IndexOrphanPackFile{orphanPackFile1}

		// Compare
		diffA := i1.Diff(i2)
		diffB := i2.Diff(i1)

		// Unique commits in each graph must be reported
		assert.Empty(t, diffA.MissingObjects)  // i1 contains all objects present in i2
		assert.Len(t, diffB.MissingObjects, 2) // i2 lacks the objects in p3
		assert.Equal(t, programming.OID, diffB.MissingObjects[0].OID)
		assert.Equal(t, linux.OID, diffB.MissingObjects[1].OID)

		// Check orphans
		assert.Empty(t, diffA.MissingOrphanBlobs)
		assert.Empty(t, diffA.MissingOrphanPackFiles)
		assert.Equal(t, []string{orphanBlob1.OID}, diffB.MissingOrphanBlobs)
		assert.Equal(t, []string{orphanPackFile2.OID}, diffB.MissingOrphanPackFiles)
	})

}

/* Test Helpers */

func NewIndexOrphanPackFileWithOID(oid string) *IndexOrphanPackFile {
	return &IndexOrphanPackFile{
		OID:   oid,
		DTime: clock.Now(),
	}
}

func NewIndexOrphanBlobWithOID(oid string) *IndexOrphanBlob {
	return &IndexOrphanBlob{
		OID:      oid,
		DTime:    clock.Now(),
		MediaOID: "848be7af-8d4e-4405-8a5c-58c9a9efaace",
	}
}
