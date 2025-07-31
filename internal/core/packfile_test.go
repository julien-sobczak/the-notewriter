package core

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestObjectData(t *testing.T) {
	tr := NewTestRepository(t)

	tr.WriteFile("project.md", ""+
		"# Project\n"+
		"\n"+
		"## TODO: Backlog\n"+
		"\n"+
		"[ ] Test `ObjectData`\n")
	parsedFile := tr.ParseFile("project.md")

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

	t.Run("NewPackFileFromParsedFile", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		parsedFile := tr.ParseFile("go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.FileExists(t, packFile.ObjectPath())

		// A blob must have been created for the original file
		assert.Len(t, packFile.BlobRefs, 1)
		for _, blob := range packFile.BlobRefs {
			assert.FileExists(t, blob.ObjectPath())
		}
		// The same blob can be found searching by mimetype
		blob := packFile.FindFirstBlobWithMimeType("text/markdown")
		require.NotNil(t, blob)
	})

	t.Run("NewPackFileFromParsedFile_DryRun", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		CurrentConfig().DryRun = true

		parsedFile := tr.ParseFile("go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("NewPackFileFromParsedFile_Retry", func(t *testing.T) {
		FreezeNow(t)
		tr := NewTestRepository(t)

		tr.WriteFile("go.md", "# Go")

		// Create a pack file from a parsed file
		parsedFile := tr.ParseFile("go.md")
		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)
		assert.Len(t, packFile.BlobRefs, 1)
		blob := packFile.BlobRefs[0]

		// Reread the same file must not trigger a new pack file
		parsedFile = tr.ParseFile("go.md")
		newPackFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)
		assert.Equal(t, packFile.OID, newPackFile.OID) // Same pack file
		assert.Len(t, newPackFile.BlobRefs, 1)
		newBlob := newPackFile.BlobRefs[0]
		assert.Equal(t, blob.OID, newBlob.OID) // Same blobs

		// Edit the Markdown file
		tr.WriteFile("go.md", "# Golang")

		// An updated file must trigger a new pack file
		parsedFile = tr.ParseFile("go.md")
		newPackFile, err = NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)
		assert.NotEqual(t, packFile.OID, newPackFile.OID) // New pack file
		assert.Len(t, newPackFile.BlobRefs, 1)
		newBlob = newPackFile.BlobRefs[0]
		assert.NotEqual(t, blob.OID, newBlob.OID) // New blob
	})

	t.Run("NewPackFileFromParsedMedia", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		parsedFile := tr.ParseFile("go.md")
		require.Len(t, parsedFile.Medias, 1)
		parsedMedia := parsedFile.Medias[0]
		packFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)

		// Pack must have been save to disk
		assert.FileExists(t, packFile.ObjectPath())

		// Blob must have been created
		assert.GreaterOrEqual(t, len(packFile.BlobRefs), 1)
		for _, blob := range packFile.BlobRefs {
			assert.FileExists(t, blob.ObjectPath())
		}
	})

	t.Run("NewPackFileFromParsedMedia_DryRun", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		CurrentConfig().DryRun = true

		parsedFile := tr.ParseFile("go.md")
		require.Len(t, parsedFile.Medias, 1)
		parsedMedia := parsedFile.Medias[0]
		packFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)

		// Pack must have been save to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs must have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("NewPackFileFromParsedMedia_Retry", func(t *testing.T) {
		tr := NewTestRepository(t)
		FreezeNow(t)

		tr.WriteFileRaw("smallest.gif", smallestGIF)

		convertionDone := false
		CurrentConfig().Converter().OnPreGeneration(func(cmd string, args ...string) {
			convertionDone = true
		})
		parsedMedia := ParseMedia(tr.Root, filepath.Join(tr.Root, "smallest.gif"))
		packFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)
		assert.True(t, convertionDone)

		// Reread the same file must not trigger a new pack file, neither new conversions
		convertionDone = false
		newPackFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)
		assert.Equal(t, packFile.OID, newPackFile.OID) // Same pack file
		assert.False(t, convertionDone)                // No convertion was done

		// An updated media must trigger a new pack file
		tr.WriteFileRaw("smallest.gif", []byte("invalid gif"))
		convertionDone = false
		newPackFile, err = NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)
		assert.NotEqual(t, packFile.OID, newPackFile.OID) // Different files, different pack files
		assert.True(t, convertionDone)                    // Reconvertion
	})

	t.Run("LoadPackFileFromPath", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		parsedFile := tr.ParseFile("go.md")

		packFileOriginal, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		packFileRead, err := LoadPackFileFromPath(packFileOriginal.ObjectPath())
		require.NoError(t, err)

		// reread pack file must be identical
		assert.Equal(t, packFileOriginal.OID, packFileRead.OID)
		assert.Equal(t, len(packFileOriginal.PackObjects), len(packFileRead.PackObjects))
		assert.Equal(t, len(packFileOriginal.BlobRefs), len(packFileRead.BlobRefs))
	})

	t.Run("LoadPackFileFromPath", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		parsedFile := tr.ParseFile("go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		ref := packFile.Ref()
		assert.Equal(t, PackFileRef{
			RelativePath: "go.md",
			OID:          packFile.OID,
			CTime:        packFile.CTime,
		}, ref)
	})

	t.Run("YAML", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30")
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

		parsedFile := tr.ParseFile("go.md")

		packFileSrc, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Marshmall YAML
		buf := new(bytes.Buffer)
		err = packFileSrc.Write(buf)
		require.NoError(t, err)
		cYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 4c578e5279f7b0eadf52c1ff5e8492bdb9a426fe
file_relative_path: go.md
file_mtime: 2023-01-01T12:30:00Z
file_size: 1
ctime: 2023-01-01T12:30:00Z
objects:
    - oid: "1000000000000000000000000000000000000000"
      kind: file
      ctime: 2023-01-01T12:30:00Z
      desc: file "go.md" [1000000000000000000000000000000000000000]
      data: eJyUk01v2zgQhu/6FbP2JdZGH1ZiO+Fld7FtXaBFUxQBCjQIZEocUYQljkBS+WjT/15QUp2gQINUp5l3XjziDIekBIPZMn3ZNwts00sGkoKOl/tKNZgPhNNytTnDVbY5rzZFilxUq6xcVtUKz07Ps0IU5/w0W1cYGGy4UzeYd9zVHhS3IrhVe9UovR/AlSHt8pY7h4bBQwAA4Li0bIgib+HOGVX0DifxsXywOOUaZLClwNZkXP6YFyTuGTz8PfjnXhmjOXwgN1gariW8VdaRuR+Lu3k9prsp/9dSb0pkUDvXWZYkqGPfRYdC8ZiMTHyWbCk/6gxJw9tWaZl7cs8lLibO1fiz66OfGEmxwJtEUJnAbC4pkTRbwC23INAqqVFAcQ+fqEDjYGtQWWzRHHsFPqo9HgPXAt6hhsua2s6SBu5gSyQbBKUhS9NNHBw6ftNwW5fciEPb70lOA/lccweC0IKrEcJwMjQkKQzBYGfQonb/jO4oisbgPwhDSV2NJgzjUfrrylOvj+Kk9eOxvkt7IxeP57i8eHXB4H/SFRrUJdqxEvoBeVRJGl73hjp8OqqpEmOfLGA3N9gqLdBEWZqdROk6yta7YLjuvFEaGawDq74ig2VQc1v/wdK2TrXIYOQuo3R5uczYScrS9EtQNFQcdnN8TdmLX9O4sO1Ad3jnkpabvaBbPVWeLDp8+z6Jv247GSWV5s0T6YApDXKHIufuN8fvO/G8wY/07jnDjwAAAP//Rfo2xQ==
    - oid: "3000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Note: Golang History" [3000000000000000000000000000000000000000]'
      data: eJzEkU+L2zAQxe/6FENyyC6sY1lx/lin3lIolFL21FK8sjSWRRyNkZTdBvrhi7NJ05ZlWeihuj3Pb968h8kZCZMFf9ubsNgfrARLmaeEmaVeeZt1LiYKRzYovWtdj/XJtdTL9QaXYl2164ajMu1S6KJtl7gpK9GYplKlWLXIriuT4s1B0nFACR8pIUsu9ShhNgoJ21MmeP+cacZ68rY+I1uC/C+AxY5CugJ/zAL2KrlHrAeVurH2fG/Yk9u53vmdhJml6ctHVUrBNYeEUTIAgEiHoFFCl9IQZZ6jn482Axqn5hRsPqp8S/XNEMgGtd87b+vR86As3p48krJnt/FlYOk3cfkFV+gXcB32zqOEDdPkE/ok4Ud2IqZTeKkGOw0fpuf1h7N+949lzj5fn499u7nYWJobfMwN6RwmU0u5pcktPKkIBqOzHg00R/hMDYYE24Au4h7D3fgFPrkd3oHyBj6gh/uO9kMkDyrBlsj2CM6D4Hw9v3SvOxU7CUtU2GhdVtjqalEpLLGqVuuiLYQoV6rSvGyEXiFryBwlzP5P6BnTAVVCU6skQXCxyHiR8eK+EHLBJedf2GEwrwPOG/z+GvAzAAD//9tVPgU=
    - oid: "4000000000000000000000000000000000000000"
      kind: link
      ctime: 2023-01-01T12:30:00Z
      desc: link "https://go.dev/doc/" [4000000000000000000000000000000000000000]
      data: eJyMzUFugzAQheG9T2F5nzAYKGEu0At01Q0aPGNi1cEWnUQ5ftWq64i3/j+9khit6+HYnKkUvmLKMv/BPgzjRQY/TnFcQIjj4EMb4yCXfvILLxP1/i2K2Yr+E9cd/tolk6aHzJX0inYt5xsblaeifS+ZttXc94z2qlq/sWnWcmZ5NFxCYzRpFrTOmbXMG93kl5uwC6nwTIrWg+9O0J6g/Wg9doAAn+Ze+XWQNpbnq+AnAAD//8vXWvg=
    - oid: "5000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Flashcard: Golang Logo" [5000000000000000000000000000000000000000]'
      data: eJyskcGO0zAQhu9+iiE9FCK5ddKmaSwWxIW9cERCAqFoHI8dq944StyFlXh4lLTdggSrPeycxvP/nvlGE5yWkBTieZGw0R+tBBu48Ti2DQ6a2+Cxs9wHG1iPzcE4T/Xcd9sU5Z6KvKxMqQShNkXeZMYUtN9WudKqwm2+M8SuX5Ls2SjxoScJHy8cLLroScLysSLhdkaDT8GGJfOhs/XZdBtg/afKxjYM8apehYE8RndPdY+xnTZf3Wn2wx2cd91BwtKGxf8GYoyDU8dIo2QAABHtOZuCgw3sWpqf3nUkIStZE7pIXZTwi8/qYgH/nsJm+UuLEXSgEWJLkKZnw3SSNIWB+oFG6uL7k5tzfko+QJra0Lc0pOnqVHr1ber6/fVqfUfa4bi2YTXe2zcXorrFsZVQbIqmVKoSmRJKCypNhaoqNO7LXaFpR7mozBYNU0E/PG7xcphvZzq4HIdPx7lJ/kJOAH28SaZ1Eli/Y81AGEnXGCXkIt9wkXGRfc5yuRFSiK/s2OunDa7T9PMpw+8AAAD//2Fl9Io=
    - oid: "6000000000000000000000000000000000000000"
      kind: flashcard
      ctime: 2023-01-01T12:30:00Z
      desc: flashcard "Golang Logo" [6000000000000000000000000000000000000000]
      data: eJyMjk9r6zAQxO/6FIuPAiWyE+ePeHmPd+qlx0KhF7O2VrKJYhlpE3rohy82Db2F6CRm5jc7cbAGip1+7hViwu7shkDNAm67en+gutof3b7VhNbVVVc6V9Nhe6xa2x5xW+0ciV+kKJ++NUa+Q/XTUKKAPNyomZB7Az6uLlbkcPXzX7mAue8wWeVjwNGrEH0UuY+JGx44kIGXxYDX2WD02QgAAAU+CpfiyAbee2SwkTJwTyDlDzFXSQmJpkSZRv4nWuzOBr7U0vAfpPRx6ilJuRKL9OdCdkC4T1bz5FOxiHnt4yrffAEY+FTMawpY/xVdImSyDbKBSlcbpUuly7eyMhtttP4Q18k+Dgyjpc9Hge8AAAD//068k60=
    - oid: "7000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "TODO: Conferences" [7000000000000000000000000000000000000000]'
      data: eJykj0GP0zAQhe/+FaP2EEDK1nabpPEV0B730hMIZR17kljNeiJnAqzEj0ctW3Ul0KoSc7L9Pb/3hoI3sKrkbbMS87j0BnrKmTzljmKHCaPDWUzWHbswYnO23Lmi2mOhq7qrWonWd4V2qusK3O9q3fq2tjtddiiuX1bq5hb8PKGBA3kSHHhEA9nh4dODgY/XQpkYKfbNC78n2LymYh4o8YW+BglHy+E7NpPl4bTq3ZMXP8IxjCEeDWQ9rf+RZZlTaBfG2QgAALb9y+k0OfQkrk/n6xgiGtB74SgyRjbwKz/T9Rr+ChBn8gG+3tM0YHIU4fOSaMJv7wbmaTabTX8hd7hs3sPjOuFTiB5TrqXe5rLMdfl4CWsGOw8GnKqdd1XrrXKy7mSppC5Vva33pWpRlVjZuuiqrWjJPxvI/r9AJlxCy+gbywb+AJVLdVDabKWR8otYJv+24GT68y3B7wAAAP//VDXUzg==
    - oid: "8000000000000000000000000000000000000000"
      kind: reminder
      ctime: 2023-01-01T12:30:00Z
      desc: 'reminder #reminder-2023-06-26 [8000000000000000000000000000000000000000]'
      data: eJyMzb9OwzAQBvDdT2GFITCksd389Y54ASYQihzfObFobcu9IB4fAaq6Vbnx7vvdFz1oXgxi3xQsGfvp/AmnP9jYth+wVf3o+lmgAdcqK51rcWhGNcM8mkZ1DtmNFHJ3V4h0Rf1ulPFkyH/hlAytmi/xcAYGeLHZJ/IxaF6+v8S0YrYx8Octx4QfjytRuui6Xq6XA271U8nILJqXDxnPPgDmSgl1rERXqa5kNqMhhMmQ5v97WQn5KpU+Ci3EG9sS3A/8/vy+F/gJAAD//2OZbo4=
blobs:
    - oid: "2000000000000000000000000000000000000000"
      mime: text/markdown
      attributes: {}
      tags:
        - original
        - markdown
`), strings.TrimSpace(cYAML))

		// Unmarshall YAML
		packFileDest := new(PackFile)
		err = packFileDest.Read(buf)
		require.NoError(t, err)
		require.Equal(t, oid.MustParse("4c578e5279f7b0eadf52c1ff5e8492bdb9a426fe"), packFileDest.OID)
		require.Len(t, packFileDest.PackObjects, 7)

		// Unmarshall the note
		noteDest := new(Note)
		err = packFileDest.PackObjects[0].Data.Unmarshal(noteDest)
		require.NoError(t, err)
		assert.Equal(t, "Go", noteDest.Title.String())

		// Unmarshall the note
		flashcardDest := new(Flashcard)
		err = packFileDest.PackObjects[1].Data.Unmarshal(flashcardDest)
		require.NoError(t, err)
		assert.Equal(t, "Golang History", flashcardDest.ShortTitle.String())

		// Unmarshall a single object by OID
		noteCopy := new(Note)
		err = packFileDest.UnmarshallObject(packFileDest.PackObjects[0].OID, noteCopy)
		require.NoError(t, err)
		require.EqualValues(t, noteDest, noteCopy)
	})

}
