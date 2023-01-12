package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/julien-sobczak/the-notetaker/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontMatterString(t *testing.T) {
	var tests = []struct {
		name     string      // name
		input    []Attribute // input
		expected string      // expected result
	}{
		{
			"Scalar values",
			[]Attribute{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key2",
					Value: 2,
				},
			},
			`
key1: value1
key2: 2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := NewFileFromAttributes(tt.input)
			actual, err := file.FrontMatterString()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}

func TestNewFile(t *testing.T) {
	f := NewEmptyFile()
	f.SetAttribute("tags", []string{"toto"})

	assert.Equal(t, []interface{}{"toto"}, f.GetAttribute("tags"))
	assert.Equal(t, []string{"toto"}, f.GetTags())

	actual, err := f.FrontMatterString()
	require.NoError(t, err)
	expected := `
tags:
- toto`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func TestNewFileFromPath(t *testing.T) {
	var tests = []struct {
		name          string // name
		rawContent    string // input
		actualContent string // output
	}{

		{
			name: "File without Front Matter",
			rawContent: `# Hello

Hello World!
`,
			actualContent: `# Hello

Hello World!
`,
		},

		{
			name: "File with Front Matter",
			rawContent: `---
tags: [test]
---

# Hello

Hello World!
`,
			actualContent: `# Hello

Hello World!
`,
		},

		{
			name: "File without Front Matter and Flashcard",
			rawContent: `# Hello

## Flashcard: Demo

What is the question?
---
The answer
`,
			actualContent: `# Hello

## Flashcard: Demo

What is the question?
---
The answer
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := SetUpCollectionFromFileContent(t, "test.md", tt.rawContent)
			f, err := NewFileFromPath(filename)
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.actualContent), strings.TrimSpace(f.Content))
		})
	}
}

func TestEditFileFrontMatter(t *testing.T) {
	filename := SetUpCollectionFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(filename)
	require.NoError(t, err)
	assert.Equal(t, int64(46), f.Size)
	assert.Equal(t, "d610783465f779858b00bc3f8133ebd5", f.Hash)

	// Check initial content
	assertFrontMatterEqual(t, `tags: [favorite, inspiration]`, f)
	assertContentEqual(t, `Blabla`, f)

	// Override an attribute
	f.SetAttribute("tags", []string{"ancient"})
	assertFrontMatterEqual(t, `tags: [ancient]`, f)

	// Add an attribute
	f.SetAttribute("extras", map[string]string{"key1": "value1", "key2": "value2"})
	assertFrontMatterEqual(t, `
tags: [ancient]
extras:
  key1: value1
  key2: value2
`, f)

	// Save the file
	f.Save()
	rawContent, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.Equal(t, `---
tags: [ancient]
extras:
  key1: value1
  key2: value2
---
Blabla`, strings.TrimSpace(string(rawContent)))

	// Check file-specific attributes has changed
	assert.Equal(t, int64(68), f.Size)
	assert.Equal(t, "a6ac86136a6ed70c213669b34491c92a", f.Hash)
}

func TestPreserveCommentsInFrontMatter(t *testing.T) {
	fc, err := os.CreateTemp("", "sample.md")
	require.NoError(t, err)
	defer os.Remove(fc.Name())

	_, err = fc.Write([]byte(`
---
# Front-Matter
tags: [favorite, inspiration] # Custom tags
# published: true
---
`))
	require.NoError(t, err)
	fc.Close()

	// Init the file
	f, err := NewFileFromPath(fc.Name())
	require.NoError(t, err)

	// Change attributes
	f.SetAttribute("tags", []string{"ancient"})
	f.SetAttribute("new", 10)
	assertFrontMatterEqual(t, `
# Front-Matter
tags: [ancient] # Custom tags
# published: true

new: 10
`, f)
	// FIXME debug why an additional newline
}

func TestGetNotes(t *testing.T) {
	filename := SetUpFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	require.Len(t, notes, 4)

	assert.Equal(t, KindFlashcard, notes[0].Kind)
	assert.Nil(t, notes[0].ParentNote)
	assert.Equal(t, 6, notes[0].Line)
	assert.Equal(t, "Flashcard: About _The NoteTaker_", notes[0].Title)
	t.Log(notes[0].RawContent)
	assert.Equal(t, notes[0].RawContent, "**What** is _The NoteTaker_?\n\n---\n\n_The NoteTaker_ is an unobstrusive application to organize all kinds of notes.")

	assert.Equal(t, KindQuote, notes[1].Kind)
	assert.Nil(t, notes[1].ParentNote)
	assert.Equal(t, 15, notes[1].Line)
	assert.Equal(t, "Quote: Gustave Flaubert on Order", notes[1].Title)
	assert.Equal(t, notes[1].RawContent, "`#favorite` `#life-changing`\n\n<!-- name: Gustave Flaubert -->\n<!-- references: https://fortelabs.com/blog/tiagos-favorite-second-brain-quotes/ -->\n\nBe regular and orderly in your life so that you may be violent and original in your work.")

	assert.Equal(t, KindFlashcard, notes[2].Kind)
	assert.Equal(t, notes[1], notes[2].ParentNote)
	assert.Equal(t, 25, notes[2].Line)
	assert.Equal(t, "Flashcard: Gustave Flaubert on Order", notes[2].Title)
	assert.Equal(t, notes[2].RawContent, "`#creativity`\n\n**Why** order is required for creativity?\n\n---\n\n> Be regular and orderly in your life **so that you may be violent and original in your work**.\n> -- Gustave Flaubert")

	assert.Equal(t, KindTodo, notes[3].Kind)
	assert.Nil(t, notes[3].ParentNote)
	assert.Equal(t, 40, notes[3].Line)
	assert.Equal(t, "TODO: Backlog", notes[3].Title)
	assert.Equal(t, notes[3].RawContent, "* [*] Complete examples\n* [ ] Write `README.md`")
}

func TestFileInheritance(t *testing.T) {
	filename := SetUpFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	require.Len(t, notes, 5)

	n := f.FindNoteByKindAndShortTitle(KindQuote, "Success Is Action")
	require.NotNil(t, n)
	assert.Equal(t, []string{"productivity", "favorite"}, n.GetTags())
}

func TestGetFlashcards(t *testing.T) {
	filename := SetUpFromGoldenFileNamed(t, "TestGetNotes.md")

	// Init the file
	f, err := NewFileFromPath(filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	flashcards := f.GetFlashcards()
	require.Len(t, flashcards, 2)

	// Check relations
	assert.Equal(t, f, flashcards[0].File)
	assert.Equal(t, notes[0], flashcards[0].Note)
	assert.Equal(t, notes[2], flashcards[1].Note)
	// Check content
	assert.Equal(t, `**What** is _The NoteTaker_?`, flashcards[0].FrontMarkdown)
	assert.Equal(t, `_The NoteTaker_ is an unobstrusive application to organize all kinds of notes.`, flashcards[0].BackMarkdown)
	assert.Equal(t, `<p><strong>What</strong> is <em>The NoteTaker</em>?</p>`, flashcards[0].FrontHTML)
	assert.Equal(t, `<p><em>The NoteTaker</em> is an unobstrusive application to organize all kinds of notes.</p>`, flashcards[0].BackHTML)
	assert.Equal(t, `What is The NoteTaker?`, flashcards[0].FrontText)
	assert.Equal(t, `The NoteTaker is an unobstrusive application to organize all kinds of notes.`, flashcards[0].BackText)
}

func TestGetMedias(t *testing.T) {
	dirname := SetUpCollectionFromGoldenDir(t)

	// Init the file
	f, err := NewFileFromPath(filepath.Join(dirname, "medias.md"))
	require.NoError(t, err)

	// Step 1: Check medias on a file
	medias, err := f.GetMedias()
	require.NoError(t, err)
	require.Len(t, medias, 5)

	// Dead links must be detected
	assert.False(t, medias[0].Dangling)
	assert.True(t, medias[1].Dangling) // Link is broken

	// Relative path must be store
	assert.Equal(t, "medias/leitner_system.svg", medias[0].Filepath)

	// File-specific information about each existing media must be collected
	assert.Equal(t, "fdfcf70a6207648fd5d54740f0ffa915", medias[0].Hash)
	assert.NotZero(t, medias[0].MTime)
	assert.Equal(t, int64(13177), medias[0].Size)

	// Step 2: Check medias on a note
	note := f.FindNoteByKindAndShortTitle(KindReference, "Animation")
	require.NotNil(t, note)
	medias, err = note.GetMedias()
	require.NoError(t, err)
	assert.Len(t, medias, 1)
	assert.Equal(t, "medias/leitner_system_animation.gif", medias[0].Filepath)

	// Step 3: Check medias on a flashcard
	flashcard := f.FindFlashcardByTitle("Fishes")
	require.NotNil(t, flashcard)
	medias, err = flashcard.GetMedias()
	require.NoError(t, err)
	assert.Len(t, medias, 2)
	assert.Equal(t, "medias/jellyfish.ogm", medias[0].Filepath)
	assert.Equal(t, "medias/aquarium.webm", medias[1].Filepath)
}

/* Test Helpers */

func assertFrontMatterEqual(t *testing.T, expected string, file *File) {
	actual, err := file.FrontMatterString()
	require.NoError(t, err)
	assertTrimEqual(t, expected, actual)
}

func assertContentEqual(t *testing.T, expected string, file *File) {
	actual := file.Content
	assertTrimEqual(t, expected, actual)
}

func assertTrimEqual(t *testing.T, expected string, actual string) {
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}
