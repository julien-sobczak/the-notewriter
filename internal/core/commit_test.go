package core

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/helpers"
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
		err := cg.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		require.NoError(t, err)
		err = cg.AppendCommit("a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		err = cg.AppendCommit("52d614e255d914e2f6022689617da983381c27a3")
		require.NoError(t, err)
		assert.Equal(t, now, cg.UpdatedAt)

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
		hashIn, _ := helpers.HashFromFile(in.Name())
		hashOut, _ := helpers.HashFromFile(out.Name())
		assert.Equal(t, hashIn, hashOut)
	})

	t.Run("Diff", func(t *testing.T) {
		cg1 := NewCommitGraph()
		cg2 := NewCommitGraph()

		// Start with a common commit
		err := cg1.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		require.NoError(t, err)
		err = cg2.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		require.NoError(t, err)

		// New commit only in cg1
		err = cg1.AppendCommit("5bb55dad2b3157a81893bc25f809d85a1fab2911")
		require.NoError(t, err)

		// New commits only in cg2
		err = cg2.AppendCommit("f3aaf5433ec0357844d88f860c42e044fe44ee61")
		require.NoError(t, err)
		err = cg2.AppendCommit("3c2fbfe30b58a9737ddfc45ef54587339b2a6c79")
		require.NoError(t, err)

		assert.EqualValues(t, []string{"f3aaf5433ec0357844d88f860c42e044fe44ee61", "3c2fbfe30b58a9737ddfc45ef54587339b2a6c79"}, cg1.MissingCommitsFrom(cg2))
		assert.EqualValues(t, []string{"5bb55dad2b3157a81893bc25f809d85a1fab2911"}, cg2.MissingCommitsFrom(cg1))
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

func TestCommit(t *testing.T) {

	// Make tests reproductible
	UseFixedOID(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("New commit", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
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
      data: eJzEkk9r20wQxu/6FINycAKxtZL/yF4c8978Qi8l5NRSlJF2tFos7YjVOm6gH77Ilu0WQqA00L3N7PM8/GZ32CgJq2mySItpEs9SnGGeYhwLUdAyKRHz+QJXq7LAuFBBaWrK/szSoiPrM8t+cIZhsDNWSXBUkiNbUOCNr0nC6PHckbDlGq2G/03n2b2Ogq5i57NB+Ptl4KhGb14oa9FXEjRPGhUczM7Uxu4kjDTfvJOM3juT7z11MgAA6Hjvel3lfdvJKCI76bNaUgYn7HTUV9GWs9vWsXbYNMbqrM/co6a7Y4ZHPaT1ZwyafymqAfsofHOgq/1ivdpqY0nCMijY+v5lHR4k/BgfVc83g+o5ONX//eU0Q87XE+C323OM5omil0hxEUF4oznSHN7BATtQ1BltSUH+Co+ck/OwdWQ6asjd9x34bHZ0D2gVfCILTxU3bccW0MOWWdcExkIiRDq5jFhhV0mgdJYIUczzco6qpJwWS5ql6aws0iQtaH6RN+h2ig9WwujfgI+u5L6pJazbzRqhclQ+hG+AhKc1eDjzbE7Q6wg3H022jtrNBc7Td3/ZvQ//O0foSWXoJSQimY5FPBbxk4hlnMip+BLsW/W+4GcAAAD//yWOfhQ=
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: flashcard
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: flashcard "Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]
      data: eJyUkEGL1EAQhe/9K8o5aWCSTmZ3x2mykT158SgIioSapNIJ20m13TWjgj9eJhmHcQWXPVVRvFfv4/HQGthtirttsynymy3e4H6Lea51Q2+LDnF/e4e7Xddg3rQq9hyklkEcGXjPDicLH9iy6gZH9ct+TSwvtQRyKMORao/SG7Ccjq0StNEoAIA1WFby05MBrb4d6DAv7TKGSSgc0RnIFWGkusNGOBgobrVWgTzJIANP8SR26CMtG3Vyml3gSeoRw2PL3ycDn3oUaJkiSE+QJOcyHFtOEgjkA0Wa5J3aY/N45fu1nlEfIEks+55CkqRqPr36cmry6+s0G6kdMGaW03i0b87RvYzOQOmrv5PLKIEnW13ll9n5doVRZr5aUJY/Z4zSVw+XFwvPxZ3Onj+ychgtxNDcr57wrQCd3K9O7CvIqtm0EAv9kKdFXWH+U9KivxS04DxXThMIhdoaxUChi81a52udf9S5yQuz0Z/Vwbf/F/wOAAD//4rN8Cg=
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
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		idx := NewIndex()

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
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
		f, err = NewFileFromPath(nil, filepath.Join(root, "python.md"))
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

		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
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
