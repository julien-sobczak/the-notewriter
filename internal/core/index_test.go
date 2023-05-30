package core

import (
	"bytes"
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
      data: eJzEUsFq20AQvesrBvvgBGJrJTu2vDimNxd6KSGnlqKMtKPVYmlHrNZxA/34okhxkjYNlAa6t5157+2bt8NGSVjP4+Uqn8fRYoULzFYYRULklMQFYna5xPW6yDHKVVCYitK/ozToyPrUsh+Yo1GwN1ZJcFSQI5tT4I2vSMLk+rEiYccVWg0fTevZ3U+Ciq1OB9zLXtCW7Pwfeo4q9OaO0gZ9KUHzrFbB0exNZexewkTz+I1H0XtnsoOnVgYAAC0fXIcrvW9aGYZkZ51WQ8rgjJ0Ou1u44/Sscawd1rWxOu00D6jp/EHDox7UujMFzc8u5WD7AfjqQE/0E/WJVhlLEpIgZ+u70B0eJfyYPqBuxwPqNujvH/5xmkHna2/w29mjjOaZortQcR7CaKw51Dw6hyO2oKg12pKC7B6uOSPnYefItFSTu+gq8Nns6QLQKvhEFm5KrpuWLaCHHbOuCIyFWIjV7DRiiW0pQVCGyTJPMiGSXMRxnCzXVFzOLxO1SDBL4kVUFHkybFpao9srPloJk/Fvn95DSl9XEjZltH3Z34RltB0wnr77U76//FNXunpxTpafvf5/wps8pddP2Ww3CKWj4mr0ipFRv4pXj36GRDYhbt/b2SZstidzfbxDru++P47Qk0rRS4hFPJ+KaCqiGxHJKJZz8SU4NOptwM8AAAD//xFjsXI=
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: flashcard
      state: added
      mtime: 2023-01-01T01:12:30Z
      desc: flashcard "Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]
      data: eJykkFGr004Qxd/3U8y/T38DbXfT3lu75Ebuky8+CoIiYZpMNuFuM+vu9Krgh5cktVQFsQgLOwznHH5zuG8s7Df5/a7e5Ga7wy0edmiM1jW9zFvEw9097vdtjaZuVOo4SiW9eLLwmj0ODt6wY9X2nqrbsgaWWy2RPEr/TFVA6Sw4Xh0bJeiSVQAAS3Cs5GsgC1p9OtFpGpr56weh+IzeglGEiaoWa+FoIb/TWkUKJL30PKRR7DEkmidqZfzbyINUR4xPDX8eLLzrUKBhSiAdQZady/DsOMsgUoiUaJBX6oD105Xv23JCfYQscxw6ilm2UtPqvw9jkx//HyvZ6u3fPP3izNXJ0VsoQvkzVpEk8uDKK7hifd5dMRbrUM6cc86ZsQjl4yVihr24V5Pnh6zojw5SrB8Wt8AvAL08LMarF7Aup8T5HKEv8mvFVzf8Vu+sv1Q7s/5TrXUkFGoqFAu5zjdLbZbavNXGmtxu9Ht1Cs2fBd8DAAD//5GN/Ek=
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
