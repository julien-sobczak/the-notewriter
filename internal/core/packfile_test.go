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
      data: eJzckNFr2zAQxt/1VxzOQ1uoY1lxklpPe8tgMMbo08ZwZeksizg6IyntAvvjh9Ok2aCUwt6mt0/33e/uO3JGQrbg73sZi8PeSrCUe0qYWxqUt3nvYqJwYKPS284N2ByplV6u73Ap1nW3bjkq0y2FLrtuiXdVLVrT1qoSqw7ZpSUr371IOowo4TMlZMmlASVcTULC5rgTfHze6YoN5G1zsmQZiz2FdNZ/e1nAQSX3iM2oUj+lnO8Me3JbNzi/lXBlafb6DJVScO0+YZQMACDSPmiU0Kc0RlkU6OcTZkTj1JyCLSZVbKi5HgPZoHY7520zMffK4s2RkZQ90aaXg6U/xPniF9OL4VIcnEcJd0yTT+iThF/50TGbwWsx2LH4MDu1P5z0h38Mc+J8fx724/qMsTQ3+FgY0gVkM0uFpewGnlQEg9FZjwbaA3ylFkOCTUAXcYfhdvqBL26Lt6C8gU/o4b6n3RjJg0qwIbIDgvMgOF/Pz9mbXsVewhIVtlpXNXa6XtQKK6zr1brsSiGqlao1r1qhV8haMoeXe/1vJwmoEppGJQmCi0XOy5yX96WQCy45/8b2o3nb4LzBn28ZfgcAAP//L/RWXg==
    - oid: "4000000000000000000000000000000000000000"
      kind: link
      ctime: 2023-01-01T12:30:00Z
      desc: link "https://go.dev/doc/" [4000000000000000000000000000000000000000]
      data: eJyMzUFugzAQheG9T2F5nzAYKGEu0At01Q0aPGNi1cEWnUQ5ftWq64i3/j+9khit6+HYnKkUvmLKMv/BPgzjRQY/TnFcQIjj4EMb4yCXfvILLxP1/i2K2Yr+E9cd/tolk6aHzJX0inYt5xsblaeifS+ZttXc94z2qlq/sWnWcmZ5NFxCYzRpFrTOmbXMG93kl5uwC6nwTIrWg+9O0J6g/Wg9doAAn+Ze+XWQNpbnq+AnAAD//8vXWvg=
    - oid: "5000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Flashcard: Golang Logo" [5000000000000000000000000000000000000000]'
      data: eJy8kUGP0zwQhu/+FfOlh35EctdJ66bxBXGBC0ckJBCKxvHYseqNo9hdWIkfj5q2uxxgtSfmNPPO45l35OiNgkKK10XBUjg5BS5yGzANPc6GuxhwdDxEF9mE/dH6QN0yd9fL5kCyblrbaEForKz7ylpJh11ba6Nb3NV7S+z5SVG92kp+nEjB+5sPln0OpGD9pCj4sFiDj9HFNQtxdN0VKgqWhjjnW/0byGYKmP0DdRPm4Xzr5t6w7/7ogx+PCtYurv62AnOevT5lSooBAGR01+wcHFxkz9JSBj+SgqphfRwzjVnBT750Vyv48xa2tD8PmMFESpAHgrK8AudPKEuYaZop0ZjfXmjO+SV5B2Xp4jTQXJabi/Tf1/PUb/9v7u7JeEx3Lm7Sg3tzc9QNmAYFciv7RutWVFpoI6ixLepWGjw0e2loT7Vo7Q4t09E8Pl3xL2zOhJlMh1lBLeotFxUX1aeqVluhhPjCTpN5GfCjoR8vAb8CAAD//1TS5qU=
    - oid: "6000000000000000000000000000000000000000"
      kind: flashcard
      ctime: 2023-01-01T12:30:00Z
      desc: flashcard "Golang Logo" [6000000000000000000000000000000000000000]
      data: eJyMjT9PwzAQxXd/iqMTWHLquE3/eEFMLIxISCAUXeKzE9WNI9utGPjwKBGIrcpNp/fe773QGw2rnVx2KzZie7K9p3oGt221P1Cl9ke7byShsZVqS2srOmyPqjHNEbdqZ4n9I6ty8dYQ8h9ULYYiecz9leoRc6fBheJsWPIXN/3Cekxdi9EIFzwOTvjgAktdiLnOffak4Xk24GUyMrqkGQCAABeYjWHIGt46zGACJcgdAee/xFTFOUQaIyUa8iNrsD1p+BZzwxNw7sLYUeS8YLN09zGtfN4X6zOZHtPahSJd3QNrI2EmU2PWoKTaCFkKWb6WSm+klvKdXUZzO9APhr5uBX4CAAD//8EWirI=
    - oid: "7000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "TODO: Conferences" [7000000000000000000000000000000000000000]'
      data: eJykj0GP0zAQhe/+FaP0EEDK1nabpPYVEMe99ARCWcceJ1azduRMgJX48ahlqyKBVisxJ4+/5/eeU3Aaipa/bgq2TOugYUgVJZcqm6LHjNHiwmZjTz5M2F0s97ZuD1jLVvm252icr6UV3td42CvZu16ZvWw8stuTQry6BT3NqOGYXGIUaEIN5fH+w72G97dCJZtSHLpnXhRsGVOm6/6HkGWcDIVv2M2GxvPn7h4d+x5OYQrxpKEc0uYf7oYoh34lXDQDACAzPJ/OU8GQ2O3qsk4hogZ5YDZFwkgaflYXutnAXwHsQt7Bl09pHjHbFOHjmtOMX9+MRPOit9vhSu5w3b6Fh03GxxAd5kpyuat4U8nm4RrWjWYZNVihrLNt74ywXHneCC4boXbq0IgeRYOtUbVvd6xP7klD+f8FSmYzGkLXGdLwG4iKi6OQesc155/ZOruXBWfTHy8JfgUAAP//xrTPgg==
    - oid: "8000000000000000000000000000000000000000"
      kind: reminder
      ctime: 2023-01-01T12:30:00Z
      desc: 'reminder #reminder-2023-06-26 [8000000000000000000000000000000000000000]'
      data: eJyMjr1urDAQhXs/hcUtuClYjJdf91FeIFWiCBl7DFbAtswQ7eNHIVptkmLF6HTzfWfGWy1o0rJjk5Ag1buxM/S7WKqqaaHiTWeagYHUpuKqMKaCtuz4oIdOlrw2QG5KUhy+5TxepeawFGGWaD+gDxInQUd/WjTRsKpoA1rvBE1fn3yYICrv6OMWfYC3/xNiWEWej9fNCbb8ISUoR0HTfxEW6zTEjDN+zlid8Tols1yxDxCNjwvoXqKgjLEi2/PMmNjzQhxc/nK3mh+ciiDxF7AXFVycv4Et6PvA14+Xe8BnAAAA///TlIXH
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
