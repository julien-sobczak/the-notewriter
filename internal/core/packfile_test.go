package core

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/testutil"
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
		tr := NewTestRepository(t,
			FromGoldenDirNamed("TestMinimal"),
			WithFreezeNow(),
			WithClockBasedFileInfoReader(),
			WithSequenceOIDs(),
		)

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
		tr := NewTestRepository(t,
			FromGoldenDirNamed("TestMinimal"),
			WithFreezeNow(),
			WithSequenceOIDs(),
			WithConfigOverride(func(c *Config) {
				c.DryRun = true
			}),
		)

		parsedFile := tr.ParseFile("go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("NewPackFileFromParsedFile_Retry", func(t *testing.T) {
		tr := NewTestRepository(t, WithFreezeNow())

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
		tr := NewTestRepository(t,
			FromGoldenDirNamed("TestMinimal"),
			WithFreezeNow(),
			WithSequenceOIDs(),
		)

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
		tr := NewTestRepository(t,
			FromGoldenDirNamed("TestMinimal"),
			WithFreezeNow(),
			WithSequenceOIDs(),
			WithConfigOverride(func(c *Config) {
				c.DryRun = true
			}),
		)

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
		// We use a boolean to determine if the conversion was done
		convertionDone := false
		tr := NewTestRepository(t,
			WithFreezeNow(),

			// Update the boolean using hooks
			WithConfigOverride(func(c *Config) {
				c.Converter().OnPreGeneration(func(cmd string, args ...string) {
					convertionDone = true
				})
			}),
		)

		tr.WriteFileRaw("smallest.gif", smallestGIF)

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
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"), WithFreezeNow(), WithSequenceOIDs())

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
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"), WithFreezeNow(), WithSequenceOIDs())

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
		tr := NewTestRepository(t,
			FromGoldenDirNamed("TestMinimal"),
			WithFreezeOn("2023-01-01 12:30"),
			WithClockBasedFileInfoReader(),
			WithSequenceOIDs())

		parsedFile := tr.ParseFile("go.md")

		packFileSrc, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Marshall YAML
		buf := new(bytes.Buffer)
		err = packFileSrc.Write(buf)
		require.NoError(t, err)
		// Save the YAML output
		yamlBytes := buf.Bytes()

		// Use regex for all data fields
		reBase64 := regexp.MustCompile(`^[A-Za-z0-9+/=\s]+$`)
		testutil.AssertYAMLMatches(t, map[string]any{
			"oid":                "4c578e5279f7b0eadf52c1ff5e8492bdb9a426fe",
			"file_relative_path": "go.md",
			"file_mtime":         "2023-01-01T12:30:00Z",
			"file_size":          1,
			"ctime":              "2023-01-01T12:30:00Z",
			"objects": []any{
				map[string]any{
					"oid":   "1000000000000000000000000000000000000000",
					"kind":  "file",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `file "go.md" [1000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "3000000000000000000000000000000000000000",
					"kind":  "note",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `note "Note: Golang History" [3000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "4000000000000000000000000000000000000000",
					"kind":  "link",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `link "https://go.dev/doc/" [4000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "5000000000000000000000000000000000000000",
					"kind":  "note",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `note "Flashcard: Golang Logo" [5000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "6000000000000000000000000000000000000000",
					"kind":  "flashcard",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `flashcard "Golang Logo" [6000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "7000000000000000000000000000000000000000",
					"kind":  "note",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `note "TODO: Conferences" [7000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
				map[string]any{
					"oid":   "8000000000000000000000000000000000000000",
					"kind":  "reminder",
					"ctime": "2023-01-01T12:30:00Z",
					"desc":  `reminder #reminder-2023-06-26 [8000000000000000000000000000000000000000]`,
					"data":  reBase64,
				},
			},
			"blobs": []any{
				map[string]any{
					"oid":        "2000000000000000000000000000000000000000",
					"mime":       "text/markdown",
					"attributes": map[string]any{},
					"tags":       []any{"original", "markdown"},
				},
			},
		}, bytes.NewReader(yamlBytes))

		// Unmarshall YAML
		packFileDest := new(PackFile)
		err = packFileDest.Read(bytes.NewReader(yamlBytes))
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
