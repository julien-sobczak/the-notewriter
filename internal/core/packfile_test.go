package core

import (
	"bytes"
	"os"
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

	t.Run("NewPackFileFromParsedFile", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		CurrentConfig().DryRun = true

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("NewPackFileFromParsedFile_Retry", func(t *testing.T) {
		FreezeNow(t)
		SetUpRepositoryFromTempDir(t)

		MustWriteFile(t, "go.md", "# Go")

		// Create a pack file from a parsed file
		parsedFile := ParseFileFromRelativePath(t, "go.md")
		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)
		assert.Len(t, packFile.BlobRefs, 1)
		blob := packFile.BlobRefs[0]

		// Reread the same file must not trigger a new pack file
		parsedFile = ParseFileFromRelativePath(t, "go.md")
		newPackFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)
		assert.Equal(t, packFile.OID, newPackFile.OID) // Same pack file
		assert.Len(t, newPackFile.BlobRefs, 1)
		newBlob := newPackFile.BlobRefs[0]
		assert.Equal(t, blob.OID, newBlob.OID) // Same blobs

		// Edit the Markdown file
		MustWriteFile(t, "go.md", "# Golang")

		// An updated file must trigger a new pack file
		parsedFile = ParseFileFromRelativePath(t, "go.md")
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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")
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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		CurrentConfig().DryRun = true

		parsedFile := ParseFileFromRelativePath(t, "go.md")
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
		root := SetUpRepositoryFromTempDir(t)
		FreezeNow(t)

		pathGif := filepath.Join(root, "smallest.gif")
		err := os.WriteFile(pathGif, smallestGIF, 0644)
		require.NoError(t, err)

		convertionDone := false
		CurrentConfig().Converter().OnPreGeneration(func(cmd string, args ...string) {
			convertionDone = true
		})
		parsedMedia := ParseMedia(root, pathGif)
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
		invalidGif := []byte("invalid gif")
		err = os.WriteFile(pathGif, invalidGif, 0644)
		require.NoError(t, err)
		convertionDone = false
		newPackFile, err = NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)
		assert.NotEqual(t, packFile.OID, newPackFile.OID) // Different files, different pack files
		assert.True(t, convertionDone)                    // Reconvertion
	})

	t.Run("LoadPackFileFromPath", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

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
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

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
      data: eJzckU+L2zwQxu/6FENyyC6sY1lx/lin95YXCqWUPbUUryyNZRFHYyRlt4F++OJssmnLsiz0Vt0ez2+emWdMzkiYLPj73oTF/mAlWMo8Jcws9crbrHMxUTiyQeld63qsT66lXq43uBTrql03HJVpl0IXbbvETVmJxjSVKsWqRXZtmRTvXiQdB5TwkRKy5FKPEmajkLA97QT/P+80Yz15W5+RLUH+B8BiRyFdgd9qAXuV3CPWg0rdGHu+N+zJ7Vzv/E7CzNL09aEqpeCaQ8IoGQBApEPQKKFLaYgyz9HPR5sBjVNzCjYfVb6l+mYIZIPa75239eh5UBZvTx5J2bPb+DKw9Iu4/IIr9AJci73zKGHDNPmEPkn4kZ2I6RRei8FOxYfpuf3hrP/7yzBnn6/Pw77dXGwszQ0+5oZ0DpOppdzS5BaeVASD0VmPBpojfKYGQ4JtQBdxj+Fu/AKf3A7vQHkDH9DDfUf7IZIHlWBLZHsE50Fwvp5fstedip2EJSpstC4rbHW1qBSWWFWrddEWQpQrVWleNkKvkDVkji/3+tdOElAlNLVKEgQXi4wXGS/uCyEXXHL+hR0G8zbgvMHvbwE/AwAA//8+vVyp
    - oid: "4000000000000000000000000000000000000000"
      kind: link
      ctime: 2023-01-01T12:30:00Z
      desc: link "https://go.dev/doc/" [4000000000000000000000000000000000000000]
      data: eJyMzUFugzAQheG9T2F5nzAYKGEu0At01Q0aPGNi1cEWnUQ5ftWq64i3/j+9khit6+HYnKkUvmLKMv/BPgzjRQY/TnFcQIjj4EMb4yCXfvILLxP1/i2K2Yr+E9cd/tolk6aHzJX0inYt5xsblaeifS+ZttXc94z2qlq/sWnWcmZ5NFxCYzRpFrTOmbXMG93kl5uwC6nwTIrWg+9O0J6g/Wg9doAAn+Ze+XWQNpbnq+AnAAD//8vXWvg=
    - oid: "5000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Flashcard: Golang Logo" [5000000000000000000000000000000000000000]'
      data: eJy8kUGP0zAQhe/+FUN7KERy66R10/iCuLAXjkhIIBSN47Fj1RtHsbuwEj8eNe1uOcBqT8xp8t7LzDdy9EbBQorX1YKlcHIKXOQ2YOo7nAx3MeDgeIgushG7o/WB2nnurpP1gWRVN7bWgtBYWXWltZIOu6bSRje4q/aW2O2XRflqlPw4koKPTxws+xxIwepZUXA3o8Gn6OKKhTi49hq6i7D502Wpj1O+uTdjooDZP1A7Yu7Pl6/vDfvhjz744ahg5eLyXwsx58nrU6akGABARnftzsXBRXaT5s/gB1JQ1qyLQ6YhK/jFZ3e5hL9vYbP9pccMJlKC3BMUxTVwfpKigInGiRIN+f0lzTm/NB+gKFwce5qKYn2R3nw7T/3+dr25J+MxbVxcpwf37omo7TH1CuRWdrXWjSi10EZQbRvUjTR4qPfS0J4q0dgdWqajeXy+4n9gToSZTItZQSWqLRclF+XnslJboYT4yk6jeTngB0M/Xwr8DgAA///8t+uP
    - oid: "6000000000000000000000000000000000000000"
      kind: flashcard
      ctime: 2023-01-01T12:30:00Z
      desc: flashcard "Golang Logo" [6000000000000000000000000000000000000000]
      data: eJyMjT9PwzAQxXd/iqMTWHLquE3/eEFMLIxISCAUXeKzE9WNI9utGPjwKBGIrcpNp/fe773QGw2rnVx2KzZie7K9p3oGt221P1Cl9ke7byShsZVqS2srOmyPqjHNEbdqZ4n9I6ty8dYQ8h9ULYYiecz9leoRc6fBheJsWPIXN/3Cekxdi9EIFzwOTvjgAktdiLnOffak4Xk24GUyMrqkGQCAABeYjWHIGt46zGACJcgdAee/xFTFOUQaIyUa8iNrsD1p+BZzwxNw7sLYUeS8YLN09zGtfN4X6zOZHtPahSJd3QNrI2EmU2PWoKTaCFkKWb6WSm+klvKdXUZzO9APhr5uBX4CAAD//8EWirI=
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
