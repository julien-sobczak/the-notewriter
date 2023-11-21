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
      packfiles: []
    - oid: a04d20dec96acfc2f9785802d7e3708721005d5d
      ctime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: 52d614e255d914e2f6022689617da983381c27a3
      ctime: 2023-01-01T01:14:30Z
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
      packfiles: []
    - oid: a04d20dec96acfc2f9785802d7e3708721005d5d
      ctime: 2023-01-01T01:14:30Z
      packfiles: []
    - oid: 52d614e255d914e2f6022689617da983381c27a3
      ctime: 2023-01-01T01:14:30Z
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

		c1 := NewCommitWithOID("a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		c2 := NewCommitWithOID("a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		c3 := NewCommitWithOID("5bb55dad2b3157a81893bc25f809d85a1fab2911")
		c4 := NewCommitWithOID("f3aaf5433ec0357844d88f860c42e044fe44ee61")
		c5 := NewCommitWithOID("3c2fbfe30b58a9737ddfc45ef54587339b2a6c79")

		// Start with a common commit
		err := cg1.AppendCommit(c1)
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

		assert.EqualValues(t, []*Commit{c4, c5}, cg1.MissingCommitsFrom(cg2))
		assert.EqualValues(t, []*Commit{c3}, cg2.MissingCommitsFrom(cg1))
	})

}

func TestObjectData(t *testing.T) {
	SetUpCollectionFromTempDir(t)
	noteSrc := NewNote(NewEmptyFile("todo.md"), nil, "TODO: Backlog", "* [ ] Test ObjectData", 2)
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
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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
objects:
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: note
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: 'note "Reference: Golang History" [93267c32147a4ab7a1100ce82faab56a99fca1cd]'
      data: eJzEUlFr2zAQfvevOJyHtNDEspMmjkjC3jLYyyh92hju2TrLwrZkZKVZYT9+uHaSZusKY4XpTXff9+m7T2eU4LCaRYtlNovC+RLnmC4xDBnLKI5yxPR2gatVnmGYCS9XFSV/R2nQknaJNm5g+r5XKi04WMrJks7Ic8pVxGF8d6xw2JkKtYSPqnXGPo29ymiZDLjLntcWxro/9CxV6NQjJQ26goM001p4B1WqSumSw1ia0RuPonNWpXtHLfcAAFqztx2ucK5peRCQnnZaDQmFU2Nl0N2CnUmuGmukxbpWWiad5h4lXT9rOJSDWncmIM2LSzHYfga+OtCZfqKeaZXSxCH2MqNdF7rFA4cfk2fUw2hAPXj9/cM/TjPofO0Nfrs6ykgzFfQYCJMF4I+kCaTxr+GALQhqldQkIH2CO5OSdbCzpFqqyd50FfisSroB1AI+kYb7wtRNazSgg50xsiJQGiLGltPTiAW2BQdGKcaLLE4ZizMWRVG8WFF+O7uNxTzGNI7mYZ5n8bBpSY22FOagOYxHv316DylcXXFYF+H2sr8OinA7YBx9d6d8f/mnrrS5OCfLL17/P+GNz+n1UzbbNUJhKd/4rxjxwaGV5DZ+klaoS79fzc3R35DQOsDteztdB832ZLaPe8j53ffJEjoSCToOEYtmExZOWHjPQh5GfMa+ePtGvA34GQAA///axbcB
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
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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
		root := SetUpCollectionFromTempDir(t)

		// Create a large file containing many notes
		var newFileContent bytes.Buffer
		newFileContent.WriteString("# New File\n\n")
		for i := 0; i < MaxObjectsPerPackFileDefault + 1; i++ {
			newFileContent.WriteString(fmt.Sprintf("## Note: Test %d\n\nBlabla\n\n", i + 1))
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

		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

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

}
