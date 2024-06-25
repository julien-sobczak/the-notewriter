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

func TestObjectData(t *testing.T) {
	root := SetUpRepositoryFromTempDir(t)

	err := os.WriteFile(filepath.Join(root, "project.md"), []byte(""+
		"# Project\n"+
		"\n"+
		"## TODO: Backlog\n"+
		"\n"+
		"[ ] Test `ObjectData`\n"), 0644)
	require.NoError(t, err)
	parsedFile, err := ParseFileFromRelativePath(root, "project.md")
	require.NoError(t, err)

	fileSrc, err := NewFile(nil, parsedFile)
	require.NoError(t, err)
	dataSrc, err := NewObjectData(fileSrc)
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
	noteDest := new(File)
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

		parsedFile, err := ParseFileFromRelativePath(root, "go.md")
		require.NoError(t, err)

		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		note, err := NewNote(file, nil, parsedFile.Notes[0])
		require.NoError(t, err)
		flashcard, err := NewFlashcard(file, note, parsedFile.Notes[0].Flashcard)
		require.NoError(t, err)

		packFileSrc := NewPackFile()
		// add a bunch of objects
		packFileSrc.AppendObject(note)
		packFileSrc.AppendObject(flashcard)

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
