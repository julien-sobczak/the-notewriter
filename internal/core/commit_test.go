package core

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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

		// Initial parent must be empty
		err := cg.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "invalid parent")
		require.ErrorContains(t, err, "invalid head")

		// A succession of commits
		now = FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 14, 30, 0, time.UTC))
		err = cg.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "")
		require.NoError(t, err)
		err = cg.AppendCommit("a04d20dec96acfc2f9785802d7e3708721005d5d", "a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		require.NoError(t, err)
		err = cg.AppendCommit("52d614e255d914e2f6022689617da983381c27a3", "a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		assert.Equal(t, now, cg.UpdatedAt)

		// Repeat the last commit must fail as head as changed
		err = cg.AppendCommit("52d614e255d914e2f6022689617da983381c27a3", "a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.ErrorContains(t, err, "invalid head")

		_, err = cg.LastCommitsFrom("unknown")
		require.ErrorContains(t, err, "unknown commit")
		commits, err := cg.LastCommitsFrom("a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		require.EqualValues(t, []string{"52d614e255d914e2f6022689617da983381c27a3"}, commits)

		buf := new(bytes.Buffer)
		err = cg.Write(buf)
		require.NoError(t, err)
		cgYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
updated_at: 2023-01-01T01:14:30Z
commits:
    - a757e67f5ae2a8df3a4634c96c16af5c8491bea2
    - a04d20dec96acfc2f9785802d7e3708721005d5d
    - 52d614e255d914e2f6022689617da983381c27a3
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
    - a757e67f5ae2a8df3a4634c96c16af5c8491bea2
    - a04d20dec96acfc2f9785802d7e3708721005d5d
    - 52d614e255d914e2f6022689617da983381c27a3
`) // Caution: spaces are important as we compare hashes at the end of the test
		in.Close()

		// Read in
		in, err = os.Open(in.Name())
		require.NoError(t, err)
		cg := new(CommitGraph)
		err = cg.Read(in)
		in.Close()
		require.NoError(t, err)
		assert.Equal(t, []string{"a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "a04d20dec96acfc2f9785802d7e3708721005d5d", "52d614e255d914e2f6022689617da983381c27a3"}, cg.CommitOIDs)

		// Write out
		err = cg.Write(out)
		require.NoError(t, err)
		out.Close()

		// Files must match
		hashIn, _ := hashFromFile(in.Name())
		hashOut, _ := hashFromFile(out.Name())
		assert.Equal(t, hashIn, hashOut)
	})
}

func TestObjectData(t *testing.T) {
	noteSrc := NewNote(NewEmptyFile(), "TODO: Backlog", "* [ ] Test ObjectData", 2)
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

func TestCommit(t *testing.T) {

	// Make tests reproductible
	UseFixedOID(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("New commit", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		f, err := NewFileFromPath(filepath.Join(root, "go.md"))
		require.NoError(t, err)

		cSrc := NewCommit()
		// add a bunch of objects
		cSrc.AppendObject(f.GetNotes()[0])
		cSrc.AppendObject(f.GetFlashcards()[0])

		// Marshmall YAML
		buf := new(bytes.Buffer)
		err = cSrc.Write(buf)
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
      data: eJzEklFr2zAQx9/1KW7uQ1qoY0nOklmkec1gL6P0aWO4Z+ssi9iSkZVmhX344STNGIzCWGGPJ/3vx++k81YrKHK5XNW5FIsVLrBaoRCc1/RBNojV+yUWRVOjqDVrbEfl37UMGMjF0vl47kwStrNOKxAs2tiRgtk9NRTI1aRg6zt0Bj7aMfrwPGNj60Msz8HfL1mgDqN9onLA2Cowft5rdrA721m3UzAz/uoVMsYYbLWPNCoGADD6fZhybYzDqLKM3HxiDaQtzn0w2VRlW19eD8GbgH1vnSkn5h4N3bCI5kxKoT0bdtaRgpzV3sXpFQIeFPxIj6nHq3PqkR3r9bs0/VcJSNPNifb1NO636xeS8XNNT5n2dQbJlfGZ8ckNHHAETaM1jjRUz3DvKwoRtoHsSD2F2+kEPtsd3QI6DZ/IwUPr+2H0DjDC1nvTEVgHkvPV/DJoi2OroKoWfKl1XRcNr6SQJHNZoCBe5IhS6Eu8x7DT/uAUzP6P+OyXeew7Beths0ZoAzV3yR9EEjhu5N2Lz+Ykvc5w89Zm62zYXOQifY+XTX7zvwuEkXSJUYHkMk+5SLl44EIJqXL+he0H/XrgZwAAAP//EyBnoA==
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: flashcard
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: flashcard "Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]
      data: eJyUkEGP0zAQhe/+FUNPEKmNne5uqZUN2hMXjkhIIBRNk4kTbZox9nQBiR+PGpeqLBKrnjwavTfv8+OhtbBdF3ebZl2Ymw3e4G6Dxmjd0NuiQ9zd3uF22zVomlbFnoPUMshIFt7ziJODD+xYdcNI9XW3JpZrLYIuWgUAsATHSn56sqDVtwMd5qFNzzAJhSccLRhFGKnusBEOFopbrVUgTzLIwFM8ikf0kdJEnRzfLvAk9R7DY8vfJwufehRomSJIT5Blp3+P7DjLIJAPFGmSd2qHzeOF79dyRn2ALHPsewpZtlLz6tWXY2lfX6/yPbUDxtzxKj65N6foXvajhdJXfyeXUQJPrrrIL/PT7gKjzH2VUNKdE0bpq4fzicRzdq9mzx9ZOewdxNDcL57xLQBHuV8c2ReQV7MpEQv9kOdFXWD+U1LSnwtKOC+V0wRCobZGsVDoYr3UZqnNR22sKexaf1YH3/5f8DsAAP//xGXohw==
`), strings.TrimSpace(cYAML))

		// Unmarshall YAML
		cDest := new(Commit)
		err = cDest.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd", cDest.OID)
		require.Len(t, cDest.Objects, 2)

		// Unmarshall the note
		noteDest := new(Note)
		err = cDest.Objects[0].Data.Unmarshal(noteDest)
		require.NoError(t, err)
		assert.Equal(t, "Reference: Golang History", noteDest.Title)

		// Unmarshall the note
		flashcardDest := new(Flashcard)
		err = cDest.Objects[1].Data.Unmarshal(flashcardDest)
		require.NoError(t, err)
		assert.Equal(t, "Golang Logo", flashcardDest.ShortTitle)

		require.EqualValues(t, cSrc, cDest)

		// Unmarshall a single object by OID
		noteCopy := new(Note)
		err = cDest.UnmarshallObject(cDest.Objects[0].OID, noteCopy)
		require.NoError(t, err)
		require.EqualValues(t, noteDest, noteCopy)
	})

}

func TestIndex(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		// Make tests reproductible
		UseFixedOID(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")
		now := FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		idx := NewIndex()

		f, err := NewFileFromPath(filepath.Join(root, "go.md"))
		require.NoError(t, err)

		c := NewCommit()
		// Add a bunch of objects
		noteExample := f.GetNotes()[0]
		flashcardExample := f.GetFlashcards()[0]
		c.AppendObject(noteExample)
		c.AppendObject(flashcardExample)

		// Add the commit
		idx.AppendCommit(c)

		// Search a commit
		commitOID, ok := idx.FindCommitContaining(noteExample.OID)
		require.True(t, ok)
		assert.Equal(t, c.OID, commitOID)

		// Create a new file
		err = os.WriteFile(filepath.Join(root, "python.md"), []byte(`# Python

## Flashcard: Python's creator

Who invented Python?
---
Guido van Rossum
`), 0644)
		require.NoError(t, err)

		// Stage the new file
		f, err = NewFileFromPath(filepath.Join(root, "python.md"))
		require.NoError(t, err)
		idx.StageObject(f)
		for _, obj := range f.SubObjects() {
			idx.StageObject(obj)
		}

		// Create a new commit
		newCommit := idx.CreateCommitFromStagingArea()
		assert.NotEmpty(t, newCommit.OID)
		assert.Equal(t, now, newCommit.CTime)
		require.Len(t, newCommit.Objects, 3)
	})

	t.Run("Save on disk", func(t *testing.T) {
		// Make tests reproductible
		UseFixedOID(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")

		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		f, err := NewFileFromPath(filepath.Join(root, "go.md"))
		require.NoError(t, err)

		idx := NewIndex()
		// add a bunch of objects
		idx.StageObject(f.GetNotes()[0])
		idx.StageObject(f.GetFlashcards()[0])

		err = idx.Save()
		require.NoError(t, err)

		idx, err = NewIndexFromPath(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		assert.Len(t, idx.StagingArea.Added, 2)
	})

}
