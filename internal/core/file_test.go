package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
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
	f.SaveOnDisk()
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
	filename := SetUpCollectionFromFileContent(t, "sample.md", `---
# Front-Matter
tags: [favorite, inspiration] # Custom tags
# published: true
---
`)

	// Init the file
	f, err := NewFileFromPath(filename)
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
	filename := SetUpCollectionFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	require.Len(t, notes, 4)

	assert.Equal(t, KindFlashcard, notes[0].Kind)
	assert.Nil(t, notes[0].ParentNote)
	assert.Equal(t, 6, notes[0].Line)
	assert.Equal(t, "Flashcard: About _The NoteTaker_", notes[0].Title)
	t.Log(notes[0].ContentRaw)
	assert.Equal(t, notes[0].ContentRaw, "**What** is _The NoteTaker_?\n\n---\n\n_The NoteTaker_ is an unobstrusive application to organize all kinds of notes.")

	assert.Equal(t, KindQuote, notes[1].Kind)
	assert.Nil(t, notes[1].ParentNote)
	assert.Equal(t, 15, notes[1].Line)
	assert.Equal(t, "Quote: Gustave Flaubert on Order", notes[1].Title)
	assert.Equal(t, notes[1].ContentRaw, "`#favorite` `#life-changing`\n\n<!-- name: Gustave Flaubert -->\n<!-- references: https://fortelabs.com/blog/tiagos-favorite-second-brain-quotes/ -->\n\nBe regular and orderly in your life so that you may be violent and original in your work.")

	assert.Equal(t, KindFlashcard, notes[2].Kind)
	assert.Equal(t, notes[1], notes[2].ParentNote)
	assert.Equal(t, 25, notes[2].Line)
	assert.Equal(t, "Flashcard: Gustave Flaubert on Order", notes[2].Title)
	assert.Equal(t, notes[2].ContentRaw, "`#creativity`\n\n**Why** order is required for creativity?\n\n---\n\n> Be regular and orderly in your life **so that you may be violent and original in your work**.\n> -- Gustave Flaubert")

	assert.Equal(t, KindTodo, notes[3].Kind)
	assert.Nil(t, notes[3].ParentNote)
	assert.Equal(t, 40, notes[3].Line)
	assert.Equal(t, "TODO: Backlog", notes[3].Title)
	assert.Equal(t, notes[3].ContentRaw, "* [*] Complete examples\n* [ ] Write `README.md`")
}

func TestFileInheritance(t *testing.T) {
	filename := SetUpCollectionFromGoldenFile(t)

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
	filename := SetUpCollectionFromGoldenFileNamed(t, "TestGetNotes.md")

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
	assert.Equal(t, "medias/leitner_system.svg", medias[0].RelativePath)

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
	assert.Equal(t, "medias/leitner_system_animation.gif", medias[0].RelativePath)

	// Step 3: Check medias on a flashcard
	flashcard := f.FindFlashcardByTitle("Fishes")
	require.NotNil(t, flashcard)
	medias, err = flashcard.GetMedias()
	require.NoError(t, err)
	assert.Len(t, medias, 2)
	assert.Equal(t, "medias/jellyfish.ogm", medias[0].RelativePath)
	assert.Equal(t, "medias/aquarium.webm", medias[1].RelativePath)
}

func TestFileSave(t *testing.T) {
	dirname := SetUpCollectionFromGoldenDir(t)

	// Init the file
	f, err := NewFileFromPath(filepath.Join(dirname, "go.md"))
	require.NoError(t, err)

	assertNoFiles(t)
	clock.Freeze()
	err = f.Save()
	require.NoError(t, err)

	require.Equal(t, 1, mustCountFiles(t))
	require.Equal(t, 3, mustCountNotes(t))
	require.Equal(t, 1, mustCountMedias(t))
	require.Equal(t, 1, mustCountFlashcards(t))
	require.Equal(t, 1, mustCountLinks(t))
	require.Equal(t, 1, mustCountReminders(t))

	// Check the file
	actual, err := LoadFileByPath(f.RelativePath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, actual.ID)
	assert.Equal(t, "go.md", actual.RelativePath)
	assert.Equal(t, "go", actual.Wikilink)
	expectedFrontMatter, err := f.FrontMatterString()
	assert.NoError(t, err)
	actualFrontMatter, err := actual.FrontMatterString()
	assert.NoError(t, err)
	assert.Equal(t, expectedFrontMatter, actualFrontMatter)
	assert.Equal(t, []string{"go"}, actual.GetTags())
	assert.Contains(t, actual.Content, "# Go", actual.Content)
	assert.Equal(t, f.Mode, actual.Mode)
	assert.Equal(t, f.Size, actual.Size)
	assert.Equal(t, f.Hash, actual.Hash)
	assert.Equal(t, f.MTime, actual.MTime)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)

	// Check a note
	note, err := FindNoteByTitle("Reference: Golang History")
	require.NoError(t, err)
	assert.NotEqual(t, 0, note.ID)
	assert.Equal(t, actual.ID, note.FileID)
	assert.EqualValues(t, -1, note.ParentNoteID) // 0 = new, -1 = nil
	assert.Equal(t, KindReference, note.Kind)
	assert.Equal(t, "Reference: Golang History", note.Title)
	assert.Equal(t, "Golang History", note.ShortTitle)
	assert.Equal(t, actual.RelativePath, note.RelativePath)
	assert.Equal(t, "go#Reference: Golang History", note.Wikilink)
	assert.Equal(t, map[string]interface{}{
		"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
	}, note.Attributes)
	assert.Equal(t, []string{"go", "history"}, note.GetTags())
	assert.Equal(t, 3, note.Line)
	assert.Equal(t, "`#history`\n\n<!-- source: https://en.wikipedia.org/wiki/Go_(programming_language) -->\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.", note.ContentRaw)
	assert.Equal(t, "bb406ddcc9f0b212e2329a1e093aa21d", note.Hash)
	assert.Equal(t, `[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.`, note.ContentMarkdown)
	assert.Equal(t, "<p><a href=\"https://go.dev/doc/\" title=\"#go/go\">Golang</a> was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.</p>", note.ContentHTML)
	assert.Equal(t, "Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.", note.ContentText)
	assert.NotEmpty(t, note.CreatedAt)
	assert.NotEmpty(t, note.UpdatedAt)
	assert.Empty(t, note.DeletedAt)
	assert.NotEmpty(t, note.LastCheckedAt)

	// Check the flashcard
	flashcardNote, err := FindNoteByTitle("Flashcard: Golang Logo")
	require.NoError(t, err)
	flashcard, err := FindFlashcardByShortTitle("Golang Logo")
	require.NoError(t, err)
	assert.NotEqual(t, 0, flashcard.ID)
	assert.Equal(t, "Golang Logo", flashcard.ShortTitle)
	assert.EqualValues(t, actual.ID, flashcard.FileID)
	assert.Equal(t, flashcardNote.ID, flashcard.NoteID)
	assert.Equal(t, []string{"go"}, flashcard.Tags)
	assert.Equal(t, CardNew, flashcard.Type)
	assert.Equal(t, QueueNew, flashcard.Queue)
	assert.EqualValues(t, 0, flashcard.Due)
	assert.EqualValues(t, 1, flashcard.Interval)
	assert.Equal(t, 2500, flashcard.EaseFactor)
	assert.Equal(t, 0, flashcard.Repetitions)
	assert.Equal(t, 0, flashcard.Lapses)
	assert.Equal(t, 0, flashcard.Left)
	assert.Equal(t, "What does the **Golang logo** represent?", flashcard.FrontMarkdown)
	assert.Equal(t, "A **gopher**.\n\n![Logo](./medias/go.svg)", flashcard.BackMarkdown)
	assert.Equal(t, "<p>What does the <strong>Golang logo</strong> represent?</p>", flashcard.FrontHTML)
	assert.Equal(t, "<p>A <strong>gopher</strong>.</p>\n\n<p><img src=\"./medias/go.svg\" alt=\"Logo\" /></p>", flashcard.BackHTML)
	assert.Equal(t, "What does the Golang logo represent?", flashcard.FrontText)
	assert.Equal(t, "A gopher.\n\n![Logo](./medias/go.svg)", flashcard.BackText)
	assert.NotEmpty(t, flashcard.CreatedAt)
	assert.NotEmpty(t, flashcard.UpdatedAt)
	assert.Empty(t, flashcard.DeletedAt)
	assert.NotEmpty(t, flashcard.LastCheckedAt)

	// Check the media
	media, err := FindMediaByRelativePath("medias/go.svg")
	require.NoError(t, err)
	assert.NotEqual(t, 0, media.ID)
	assert.Equal(t, "medias/go.svg", media.RelativePath)
	assert.Equal(t, KindPicture, media.Kind)
	assert.Equal(t, false, media.Dangling)
	assert.Equal(t, 0, media.Links)
	assert.Equal(t, ".svg", media.Extension)
	assert.NotEmpty(t, media.MTime)
	assert.Equal(t, "974a75814a1339c82cb497ea1ab56383", media.Hash)
	assert.EqualValues(t, 2288, media.Size)
	assert.NotEmpty(t, media.Mode)
	assert.NotEmpty(t, media.CreatedAt)
	assert.NotEmpty(t, media.UpdatedAt)
	assert.Empty(t, media.DeletedAt)
	assert.NotEmpty(t, media.LastCheckedAt)

	// Check the link
	links, err := FindLinksByText("Golang")
	require.NoError(t, err)
	require.Len(t, links, 1)
	link := links[0]
	linkNote, err := FindNoteByTitle("Reference: Golang History")
	require.NoError(t, err)
	assert.NotEqual(t, 0, link.ID)
	assert.Equal(t, linkNote.ID, link.NoteID)
	assert.Equal(t, "Golang", link.Text)
	assert.Equal(t, "https://go.dev/doc/", link.URL)
	assert.Equal(t, "", link.Title)
	assert.Equal(t, "go", link.GoName)
	assert.NotEmpty(t, link.CreatedAt)
	assert.NotEmpty(t, link.UpdatedAt)
	assert.Empty(t, link.DeletedAt)
	assert.NotEmpty(t, link.LastCheckedAt)

	// Check the reminder
	reminders, err := FindReminders()
	require.NoError(t, err)
	require.Len(t, reminders, 1)
	reminder := reminders[0]
	reminderNote, err := FindNoteByTitle("TODO: Conferences")
	require.NoError(t, err)
	assert.NotEqual(t, 0, reminder.ID)
	assert.Equal(t, reminderNote.ID, reminder.NoteID)
	assert.Equal(t, reminderNote.FileID, reminder.FileID)
	assert.Equal(t, "[Gophercon Europe](https://gophercon.eu/)", reminder.DescriptionRaw)
	assert.Equal(t, "[Gophercon Europe](https://gophercon.eu/)", reminder.DescriptionMarkdown)
	assert.Equal(t, "<p><a href=\"https://gophercon.eu/\">Gophercon Europe</a></p>", reminder.DescriptionHTML)
	assert.Equal(t, "Gophercon Europe", reminder.DescriptionText)
	assert.Equal(t, "#reminder-2023-06-26", reminder.Tag)
	assert.Empty(t, reminder.LastPerformedAt)
	assert.EqualValues(t, time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC), reminder.NextPerformedAt) // FIXME
	assert.NotEmpty(t, reminder.CreatedAt)
	assert.NotEmpty(t, reminder.UpdatedAt)
	assert.Empty(t, reminder.DeletedAt)
	assert.NotEmpty(t, reminder.LastCheckedAt)
}

/* Test Helpers */

func mustCountFiles(t *testing.T) int {
	count, err := CountFiles()
	require.NoError(t, err)
	return count
}

func mustCountMedias(t *testing.T) int {
	count, err := CountMedias()
	require.NoError(t, err)
	return count
}

func mustCountNotes(t *testing.T) int {
	count, err := CountNotes()
	require.NoError(t, err)
	return count
}

func mustCountLinks(t *testing.T) int {
	count, err := CountLinks()
	require.NoError(t, err)
	return count
}

func mustCountFlashcards(t *testing.T) int {
	count, err := CountFlashcards()
	require.NoError(t, err)
	return count
}

func mustCountReminders(t *testing.T) int {
	count, err := CountReminders()
	require.NoError(t, err)
	return count
}

func assertNoFiles(t *testing.T) {
	count, err := CountFiles()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

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