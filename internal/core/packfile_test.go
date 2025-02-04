package core

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestObjectData(t *testing.T) {
	SetUpRepositoryFromTempDir(t)

	WriteFileFromRelativePath(t, "project.md", ""+
		"# Project\n"+
		"\n"+
		"## TODO: Backlog\n"+
		"\n"+
		"[ ] Test `ObjectData`\n")
	parsedFile := ParseFileFromRelativePath(t, "project.md")

	dummyPackFile := DummyPackFile()

	fileSrc, err := NewFile(dummyPackFile, parsedFile)
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
	fileDest := new(File)
	err = dataDest.Unmarshal(fileDest)
	require.NoError(t, err)
	assert.Equal(t, "Project", fileDest.Title.String())
}

func TestPackFile(t *testing.T) {
	t.Skip()

	// Make tests reproductible
	oid.UseFixed(t, "93267c32147a4ab7a1100ce82faab56a99fca1cd")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("New pack file", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		dummyPackFile := DummyPackFile()

		// FIXME how to convert a `Markdown.File` to a `File` (including subobjects) and a list of `Media`
		file, err := NewFile(dummyPackFile, parsedFile)
		require.NoError(t, err)
		parsedNote, ok := parsedFile.FindNoteByTitle("Flashcard: Golang Logo")
		require.True(t, ok)
		note, err := NewNote(dummyPackFile, file, parsedNote)
		require.NoError(t, err)
		flashcard, err := NewFlashcard(dummyPackFile, file, note, parsedNote.Flashcard)
		require.NoError(t, err)

		packFileSrc := NewPackFile(file)
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
file_relative_path: go.md
file_mtime: 2023-01-01T01:12:30Z
file_size: 1
ctime: 2023-01-01T01:12:30Z
mtime: 2023-01-01T01:12:30Z
objects:
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: note
      mtime: 2023-01-01T01:12:30Z
      desc: 'note "Flashcard: Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]'
      data: eJy8kU2L2z4Qxu/6FPPfHPKvQYlkx/Fal9JL99JjodBSzNgayyJayViTLYV++JKX3RT6Ar1Up+F5fppnhkneGmirct8MVal3De6wb1BrpQa6L0fEvt5j244D6sGKHI7OgEtyDJinARcrXQoYnQzJJTH6QN3fdZxxochdTHz9eXcnDj5aAy8Rgj0HMrB++6wYeDinwrvk0lqEFF13hR4SbH90RZ7Swjf3ZiwUkP0TdTPydFpq82jFF3/wwceDgbVLq98FIvPi+yNTNgIAgNFdq9OT4NJF/jn0Rp6p4CMZ0I0YUmSKbOCbPLurFfw6XJztDxMy2EQZeCIoiitwOkJRwELzQpkiv77QUspL8QaKwqV5oqUoNhfpv0+nrp//32wfyXrMW5c2+cm9ep6omzBPBuqqHpq+b5XuVW8VNWOLfVtbvG/2taU9laoddziKPtmvL1v8izEXQibbIRsoVVlJpaXS75U2ujSV+iiOs/0z8D0AAP//Mqvm+Q==
    - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
      kind: flashcard
      mtime: 2023-01-01T01:12:30Z
      desc: flashcard "Golang Logo" [93267c32147a4ab7a1100ce82faab56a99fca1cd]
      data: eJyUjTFP8zAQQHf/ivu2D0tO7aS01AtiYmFEQgKh6GpfnAg3F9nXTvx4lIoZqdvp3r13PEUPh67d7UPXuu0et3jco3PWBnpoB8Tj/Q4PhyGgC1ENU6b+NmVmuVUplFGmC/ULyughcXOKquZzWmczZKxjwBJN4oxzMpkTqzpykV4myeTh+QrgZQWCqXoFAGAgsRoKz+LhbUSByFRBRgKtf401pTUUWgpVmuVRHTF8efg218ITaJ14Galo3ajr6t/H+uXzf7M5UZywbhI39ZLuVCiEQrFH8dDatjPWGeterfOu9Z19V+cl/n3wEwAA//8UR3ua
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
		assert.Equal(t, "Flashcard: Golang Logo", noteDest.Title.String())

		// Unmarshall the note
		flashcardDest := new(Flashcard)
		err = packFileDest.PackObjects[1].Data.Unmarshal(flashcardDest)
		require.NoError(t, err)
		assert.Equal(t, "Golang Logo", flashcardDest.ShortTitle.String())

		require.EqualValues(t, packFileSrc, packFileDest)

		// Unmarshall a single object by OID
		noteCopy := new(Note)
		err = packFileDest.UnmarshallObject(packFileDest.PackObjects[0].OID, noteCopy)
		require.NoError(t, err)
		require.EqualValues(t, noteDest, noteCopy)
	})

}
