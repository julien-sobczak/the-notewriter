package e2e_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities

	"github.com/julien-sobczak/the-notewriter/pkg/anki"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/fixtures.apkg
var fixturesAPKG []byte

func TestAnkiImport(t *testing.T) {

	t.Run("ExtractCollection", func(t *testing.T) {
		collection := NewTestAnki(t)

		// Update this test if the fixtures are updated
		require.Len(t, collection.Notes, 6)
		require.Len(t, collection.Cards, 11)

		// Check the models
		require.Len(t, collection.Models, 5)
		modelBasic := collection.Models[int64(0)]
		modelBasicReversed := collection.Models[int64(1)]
		modelBasicId := collection.Models[int64(2)]
		modelCloze := collection.Models[int64(3)]
		modelCustom := collection.Models[int64(4)]

		assert.Equal(t, &anki.Model{
			ID:   1761228936233,
			Name: "Basic",
			Fields: []anki.Field{
				{Name: "Front", Ord: 0},
				{Name: "Back", Ord: 1},
			},
			Templates: []anki.Template{
				{Name: "Card 1", Ord: 0, Qfmt: "{{Front}}", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n{{Back}}"},
			},
		}, modelBasic)
		assert.Equal(t, &anki.Model{
			ID:   1761227904554,
			Name: "Basic (and reversed card)",
			Fields: []anki.Field{
				{Name: "Front", Ord: 0},
				{Name: "Back", Ord: 1},
			},
			Templates: []anki.Template{
				{Name: "Card 1", Ord: 0, Qfmt: "{{Front}}", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n{{Back}}"},
				{Name: "Card 2", Ord: 1, Qfmt: "{{Back}}", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n{{Front}}"},
			},
		}, modelBasicReversed)
		assert.Equal(t, &anki.Model{
			ID:   1761227904553,
			Name: "Basic-1e202",
			Fields: []anki.Field{
				{Name: "Front", Ord: 0},
				{Name: "Back", Ord: 1},
			},
			Templates: []anki.Template{
				{Name: "Card 1", Ord: 0, Qfmt: "{{Front}}", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n{{Back}}"},
			},
		}, modelBasicId)
		assert.Equal(t, &anki.Model{
			ID:   1761227904557,
			Name: "Cloze",
			Fields: []anki.Field{
				{Name: "Text", Ord: 0},
				{Name: "Back Extra", Ord: 1},
			},
			Templates: []anki.Template{
				{Name: "Cloze", Ord: 0, Qfmt: "{{cloze:Text}}", Afmt: "{{cloze:Text}}<br>\n{{Back Extra}}"},
			},
		}, modelCloze)
		assert.Equal(t, &anki.Model{
			ID:   1761227937885,
			Name: "CustomType",
			Fields: []anki.Field{
				{Name: "Word", Ord: 0},
				{Name: "Definition", Ord: 1},
				{Name: "Synonym", Ord: 2},
			},
			Templates: []anki.Template{
				{Name: "Card 1", Ord: 0, Qfmt: "{{Word}}", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n{{Definition}}"},
				{Name: "Card 2", Ord: 1, Qfmt: "Synomym of <i>{{Word}}</i>?", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n<strong>{{Synonym}}</strong>"},
				{Name: "Card 3", Ord: 2, Qfmt: "<b>Found</b> the word\n\n<em class=\"english\">{{Definition}}</em>", Afmt: "{{FrontSide}}\n\n<hr id=answer>\n\n<strong style=\"font-color: red\">{{Word}}</strong>"},
			},
		}, modelCustom)

		// Check some notes

		note1 := collection.Notes[0]
		note2 := collection.Notes[1]
		note3 := collection.Notes[2]
		note4 := collection.Notes[3]
		note5 := collection.Notes[4]
		note6 := collection.Notes[5]

		assert.NotZero(t, note1.ID)
		assert.Equal(t, []string{"car", "A 4-wheel vehicle to drive on the road", "vehicle"}, note1.Fields)
		assert.Equal(t, modelCustom.ID, note1.MID)

		assert.NotZero(t, note2.ID)
		assert.Equal(t, []string{"kid", "a small person that always plays", "child"}, note2.Fields)
		assert.Equal(t, modelCustom.ID, note2.MID)

		assert.NotZero(t, note3.ID)
		assert.Equal(t, []string{"Find the <b>application name</b>:<br><br><img src=\"Anki-icon.svg\">", "<b>Anki</b> application"}, note3.Fields)
		assert.Equal(t, modelBasicId.ID, note3.MID)

		assert.NotZero(t, note4.ID)
		assert.Equal(t, []string{"[sound:En-us-happiness.ogg]", "Happiness"}, note4.Fields)
		assert.Equal(t, modelBasicId.ID, note4.MID)

		assert.NotZero(t, note5.ID)
		assert.Equal(t, []string{"Canberra was founded in {{c1::1913}}", "Found the year"}, note5.Fields)
		assert.Equal(t, modelCloze.ID, note5.MID)

		assert.NotZero(t, note6.ID)
		assert.Equal(t, []string{"Voiture", "<font color=\"#0000ff\">car</font>"}, note6.Fields)
		assert.Equal(t, modelBasicReversed.ID, note6.MID)

		card1 := collection.Cards[0]

		assert.Equal(t, &anki.Card{
			ID:   card1.ID,
			NID:  note1.ID,
			Mod:  1761228873,
			Ord:  0,
			Type: 2,
			Due:  4,
			Ivl:  4,
			Reps: 1,
		}, card1)
	})

	t.Run("BasicImport", func(t *testing.T) {
		collection := NewTestAnki(t)

		tr := NewTestRepository(t)

		outputFile := filepath.Join(tr.Root, "skills/random/index.md")
		packFile, err := collection.Import(outputFile)
		require.NoError(t, err)

		// Check output file was created
		assert.FileExists(t, outputFile)
		// Read and verify content
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		expectedContent := text.UnescapeTestContent(`
# Untitled

## Untitled ‛@slug: anki-1761228454049‛

### Flashcard: Untitled ‛@cid: 1761228454049‛ ‛@slug: anki-1761228454049‛ ‛#english‛ ‛#word‛

car

---

A 4-wheel vehicle to drive on the road

### Flashcard: Untitled ‛@cid: 1761228454050‛ ‛@slug: anki-1761228454050‛ ‛#english‛ ‛#word‛

Synomym of _car_?

---

**vehicle**

### Flashcard: Untitled ‛@cid: 1761228454051‛ ‛@slug: anki-1761228454051‛ ‛#english‛ ‛#word‛

**Found** the word

<em class="english">A 4-wheel vehicle to drive on the road</em>

---

<strong style="font-color: red">car</strong>

## Untitled ‛@slug: anki-1761228487386‛

### Flashcard: Untitled ‛@cid: 1761228487386‛ ‛@slug: anki-1761228487386‛ ‛#english‛ ‛#word‛

kid

---

a small person that always plays

### Flashcard: Untitled ‛@cid: 1761228487387‛ ‛@slug: anki-1761228487387‛ ‛#english‛ ‛#word‛

Synomym of _kid_?

---

**child**

### Flashcard: Untitled ‛@cid: 1761228487388‛ ‛@slug: anki-1761228487388‛ ‛#english‛ ‛#word‛

**Found** the word

<em class="english">a small person that always plays</em>

---

<strong style="font-color: red">kid</strong>

## Flashcard: Untitled ‛@cid: 1761228593304‛ ‛@slug: anki-1761228593304‛ ‛#software‛

Find the **application name**:



![Anki-icon.svg](./Anki-icon.svg)

---

**Anki** application

## Flashcard: Untitled ‛@cid: 1761228745053‛ ‛@slug: anki-1761228745053‛ ‛#software‛

![En-us-happiness.ogg](./En-us-happiness.ogg)

---

Happiness

## Flashcard: Untitled ‛@cid: 1761228816781‛ ‛@slug: anki-1761228816781‛ ‛#geography‛ ‛#trivia‛

Canberra was founded in [{c1::1913}]

## Untitled ‛@slug: anki-1761228859664‛

### Flashcard: Untitled ‛@cid: 1761228859664‛ ‛@slug: anki-1761228859664‛ ‛#french‛ ‛#vocabulary‛

Voiture

---

<font color="#0000ff">car</font>

### Flashcard: Untitled ‛@cid: 1761228859665‛ ‛@slug: anki-1761228859665‛ ‛#french‛ ‛#vocabulary‛

<font color="#0000ff">car</font>

---

Voiture
`)
		assert.Equal(t, strings.TrimSpace(expectedContent), strings.TrimSpace(string(content)))

		// Check medias were extracted
		mediaDir := filepath.Dir(outputFile)
		assert.FileExists(t, filepath.Join(mediaDir, "Anki-icon.svg"))
		assert.FileExists(t, filepath.Join(mediaDir, "En-us-happiness.ogg"))

		// Check operations were saved on disk
		assert.FileExists(t, packFile)
		// No other files must exist
		objects := ListFilesInDir(t, filepath.Join(tr.Root, ".nt/objects"))
		assert.Len(t, objects, 1)
	})

	t.Run("ImportWithMediaDir", func(t *testing.T) {
		collection := NewTestAnki(t)

		tr := NewTestRepository(t)

		outputFile := filepath.Join(tr.Root, "skills/index.md")
		_, err := collection.Import(
			outputFile,
			anki.WithMediaDir("medias"),
			anki.WithStaged(true),
		)
		require.NoError(t, err)

		// Check output files
		assert.FileExists(t, outputFile)

		// Verify media files
		mediaDir := filepath.Join(filepath.Dir(outputFile), "medias")
		assert.FileExists(t, filepath.Join(mediaDir, "Anki-icon.svg"))
		assert.FileExists(t, filepath.Join(mediaDir, "En-us-happiness.ogg"))

		objectsDir := filepath.Join(tr.Root, ".nt/objects")
		filenames := ListFilesInDir(t, objectsDir)
		require.Len(t, filenames, 1) // 1 Markdown file = 1 pack file

		// Verify the pack file
		packFile, err := LoadPackFileFromPath(filepath.Join(objectsDir, filenames[0]))
		require.NoError(t, err)
		assert.Greater(t, len(packFile.PackObjects), 0)
		packObject := packFile.PackObjects[0]
		assert.Equal(t, "operation", packObject.Kind)
		operation := packObject.Read().(*Operation)
		assert.Equal(t, "review-flashcard", operation.Name)
		assert.Contains(t, operation.Extras, "review")
		review := operation.Extras["review"].(map[string]any)
		require.Contains(t, review, "algorithm")
		require.Contains(t, review, "settings")
		assert.Equal(t, "anki-sm-2", review["algorithm"])
		assert.Contains(t, review["settings"], "factor")
		assert.Contains(t, review["settings"], "ivl")
		assert.Contains(t, review["settings"], "lastIvl")
		assert.Contains(t, review["settings"], "time")
		assert.Contains(t, review["settings"], "type")
	})

	t.Run("AppendToExistingFile", func(t *testing.T) {
		collection := NewTestAnki(t)

		tr := NewTestRepository(t)
		tr.WriteFile("existing.md", "# Existing\n\n## Note: An existing note\n\nHere is some text.\n")

		packFile, err := collection.Import(
			tr.AbsolutePath("existing.md"),
			anki.WithIgnoreScheduling(true),
		)
		require.NoError(t, err)

		// Read file and verify both old and new content
		content := tr.ReadFile("existing.md")

		// Check existing content is preserved
		assert.Contains(t, content, "## Note: An existing note")
		assert.Contains(t, content, "Here is some text.")
		// Check new content was appended
		assert.Contains(t, content, "## Flashcard:")

		// Check no objects were created (no packfiles needed)
		assert.Empty(t, packFile)
		filenames := ListFilesInDir(t, filepath.Join(tr.Root, ".nt/objects"))
		assert.Len(t, filenames, 0)
	})

}

/* Helpers */

// NewTestAnki extracts the embedded fixtures APKG and returns the Collection for testing.
func NewTestAnki(t *testing.T) *anki.Collection {
	t.Helper()
	apkgSource := copyFixturesTemp(t)
	collection, err := anki.ExtractCollection(apkgSource)
	require.NoError(t, err)
	t.Cleanup(func() {
		collection.Close()
	})
	return collection
}

// copyFixturesTemp writes the embedded fixtures APKG to a temporary file and returns its path.
func copyFixturesTemp(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "fixtures-*.apkg")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	_, err = tmpFile.Write(fixturesAPKG)
	require.NoError(t, err)
	tmpFile.Close()

	return tmpFile.Name()
}

// ListFilesInDir lists all files in the given directory and its subdirectories.
func ListFilesInDir(t *testing.T, dir string) []string {
	t.Helper()
	// Return nothing if the directory does not exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []string{}
	}
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	require.NoError(t, err)
	return files
}
