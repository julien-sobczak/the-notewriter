package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/copier"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
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
			SetUpCollectionFromTempDir(t)
			file := NewFileFromAttributes(nil, "", tt.input)
			actual, err := file.FrontMatterString()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}

func TestNewFile(t *testing.T) {
	SetUpCollectionFromTempDir(t)
	f := NewEmptyFile("")
	f.SetAttribute("tags", []interface{}{"toto"})

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

	t.Run("Extract content", func(t *testing.T) {
		var tests = []struct {
			name       string // name
			rawContent string // input
			actualBody string // output
		}{

			{
				name: "File without Front Matter",
				rawContent: `# Hello

Hello World!
`,
				actualBody: `# Hello

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
				actualBody: `# Hello

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
				actualBody: `# Hello

## Flashcard: Demo

What is the question?
---
The answer
`,
			},

			{
				name: "File without Front Matter but with a nested Markdown document",
				rawContent: "" +
					"# Hello\n" +
					"\n" +
					"## Cheatsheet: Markdown + Front Matter\n" +
					"\n" +
					"```md\n" +
					"---\n" +
					"layout: post-read\n" +
					"---\n" +
					"\n" +
					"# A Nested Document\n" +
					"\n" +
					"A nested content\n" +
					"```\n",
				actualBody: "" +
					"# Hello\n" +
					"\n" +
					"## Cheatsheet: Markdown + Front Matter\n" +
					"\n" +
					"```md\n" +
					"---\n" +
					"layout: post-read\n" +
					"---\n" +
					"\n" +
					"# A Nested Document\n" +
					"\n" +
					"A nested content\n" +
					"```\n",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filename := SetUpCollectionFromFileContent(t, "test.md", tt.rawContent)
				f, err := NewFileFromPath(nil, filename)
				require.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.actualBody), strings.TrimSpace(f.Body))
			})
		}
	})

	t.Run("Extract notes", func(t *testing.T) { // FIXME debug and implement
		tests := []struct {
			name       string
			rawContent string                      // input
			check      func(t *testing.T, f *File) // output
		}{

			// A file can contains a single note.
			// The free kind is used by default.
			{
				name: "A file used as a free note",
				rawContent: `
# Free Note

This is a free note.
`,
				check: func(t *testing.T, f *File) {
					// A single note representing the whole file must be found.
					notes := f.GetNotes()
					require.Len(t, notes, 1)
					note := notes[0]
					require.Equal(t, KindFree, note.NoteKind)
				},
			},

			// A prefix can be used
			{
				name: "A file used as a reference note",
				rawContent: `
# Cheatsheet: Using a file as a single note

This is a note.
`,
				check: func(t *testing.T, f *File) {
					// A single note representing the whole file must be found.
					notes := f.GetNotes()
					require.Len(t, notes, 1)
					note := notes[0]
					require.Equal(t, KindCheatsheet, note.NoteKind)
				},
			},

			// A file can contains free notes (sometimes due to typo)
			{
				name: "A file contaning free notes",
				rawContent: `
# Example

## Note: A typed note

This note is strongly typed.

## A free note

This note is not typed.

## Flaschard: A untyped note due to a typo

What is the kind for flashcard?

---

Flashcard

`,
				check: func(t *testing.T, f *File) {
					// Free notes must be found even if untyped
					notes := f.GetNotes()
					require.Len(t, notes, 3)
					note1 := notes[0]
					note2 := notes[1]
					note3 := notes[2]
					require.Equal(t, KindNote, note1.NoteKind)
					require.Equal(t, KindFree, note2.NoteKind)
					require.Equal(t, KindFree, note3.NoteKind)
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filename := SetUpCollectionFromFileContent(t, "test.md", tt.rawContent)
				f, err := NewFileFromPath(nil, filename)
				require.NoError(t, err)
				tt.check(t, f)
			})
		}
	})

}

func TestEditFileFrontMatter(t *testing.T) {
	filename := SetUpCollectionFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(nil, filename)
	require.NoError(t, err)
	assert.Equal(t, int64(46), f.Size)
	assert.Equal(t, "7bd2eeb34151be89fad00c85274dd1a42c6e87b0", f.Hash)

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
	assert.Equal(t, "3acf3dd5f831dc123fb79034580ae08cf44c5667", f.Hash)
}

func TestPreserveCommentsInFrontMatter(t *testing.T) {
	filename := SetUpCollectionFromFileContent(t, "sample.md", `---
# Front-Matter
tags: [favorite, inspiration] # Custom tags
# published: true
---
`)

	// Init the file
	f, err := NewFileFromPath(nil, filename)
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
	f, err := NewFileFromPath(nil, filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	require.Len(t, notes, 4)

	assert.Equal(t, KindFlashcard, notes[0].NoteKind)
	assert.Nil(t, notes[0].ParentNote)
	assert.Equal(t, 10, notes[0].Line)
	assert.Equal(t, "Flashcard: About _The NoteWriter_", notes[0].Title)
	assert.Equal(t, notes[0].ContentRaw, "**What** is _The NoteWriter_?\n\n---\n\n_The NoteWriter_ is an unobstrusive application to organize all kinds of notes.")

	assert.Equal(t, KindQuote, notes[1].NoteKind)
	assert.Nil(t, notes[1].ParentNote)
	assert.Equal(t, 19, notes[1].Line)
	assert.Equal(t, "Quote: Gustave Flaubert on Order", notes[1].Title)
	assert.Equal(t, notes[1].ContentRaw, "`#favorite` `#life-changing`\n\n`@name: Gustave Flaubert`\n`@references: https://fortelabs.com/blog/tiagos-favorite-second-brain-quotes/`\n\nBe regular and orderly in your life so that you may be violent and original in your work.")

	assert.Equal(t, KindFlashcard, notes[2].NoteKind)
	assert.Equal(t, notes[1], notes[2].ParentNote)
	assert.Equal(t, 29, notes[2].Line)
	assert.Equal(t, "Flashcard: Gustave Flaubert on Order", notes[2].Title)
	assert.Equal(t, notes[2].ContentRaw, "`#creativity`\n\n**Why** order is required for creativity?\n\n---\n\n> Be regular and orderly in your life **so that you may be violent and original in your work**.\n> -- Gustave Flaubert")

	assert.Equal(t, KindTodo, notes[3].NoteKind)
	assert.Nil(t, notes[3].ParentNote)
	assert.Equal(t, 41, notes[3].Line)
	assert.Equal(t, "TODO: Backlog", notes[3].Title)
	assert.Equal(t, notes[3].ContentRaw, "* [*] Complete examples\n* [ ] Write `README.md`")
}

func TestFileInheritance(t *testing.T) {
	// TOOD Complete using fixture TestInheritance/
	filename := SetUpCollectionFromGoldenFile(t)

	// Init the file
	f, err := NewFileFromPath(nil, filename)
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
	f, err := NewFileFromPath(nil, filename)
	require.NoError(t, err)

	notes := f.GetNotes()
	flashcards := f.GetFlashcards()
	require.Len(t, flashcards, 2)

	// Check relations
	assert.Equal(t, f, flashcards[0].File)
	assert.Equal(t, notes[0], flashcards[0].Note)
	assert.Equal(t, notes[2], flashcards[1].Note)
	// Check content
	assert.Equal(t, `**What** is _The NoteWriter_?`, flashcards[0].FrontMarkdown)
	assert.Equal(t, `_The NoteWriter_ is an unobstrusive application to organize all kinds of notes.`, flashcards[0].BackMarkdown)
	assert.Equal(t, `<p><strong>What</strong> is <em>The NoteWriter</em>?</p>`, flashcards[0].FrontHTML)
	assert.Equal(t, `<p><em>The NoteWriter</em> is an unobstrusive application to organize all kinds of notes.</p>`, flashcards[0].BackHTML)
	assert.Equal(t, `What is The NoteWriter?`, flashcards[0].FrontText)
	assert.Equal(t, `The NoteWriter is an unobstrusive application to organize all kinds of notes.`, flashcards[0].BackText)
}

func TestGetMedias(t *testing.T) {
	root := SetUpCollectionFromGoldenDir(t)

	// Init the file
	f, err := NewFileFromPath(nil, filepath.Join(root, "medias.md"))
	require.NoError(t, err)

	// Step 1: Check medias on a file
	medias := f.GetMedias()
	require.Len(t, medias, 5)

	// Dead links must be detected
	assert.False(t, medias[0].Dangling)
	assert.True(t, medias[1].Dangling) // Link is broken

	// Relative path must be store
	assert.Equal(t, "medias/leitner_system.svg", medias[0].RelativePath)

	// File-specific information about each existing media must be collected
	assert.Equal(t, "4bfdde386e5c63e7f1b31c77574e0cc9c25aab69", medias[0].Hash)
	assert.NotZero(t, medias[0].MTime)
	assert.Equal(t, int64(13177), medias[0].Size)

	// Step 2: Check medias on a note
	note := f.FindNoteByKindAndShortTitle(KindReference, "Animation")
	require.NotNil(t, note)
	medias = note.GetMedias()
	assert.Len(t, medias, 1)
	assert.Equal(t, "medias/leitner_system_animation.gif", medias[0].RelativePath)

	// Step 3: Check medias on a flashcard
	flashcard := f.FindFlashcardByTitle("Fishes")
	require.NotNil(t, flashcard)
	note = f.FindNoteByKindAndShortTitle(KindFlashcard, "Fishes")
	require.NotNil(t, flashcard)
	medias = note.GetMedias()
	assert.Len(t, medias, 2)
	assert.Equal(t, "medias/jellyfish.ogm", medias[0].RelativePath)
	assert.Equal(t, "medias/aquarium.webm", medias[1].RelativePath)
}

func TestFileSave(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")
	FreezeNow(t)
	assertNoFiles(t)
	assertNoNotes(t)
	assertNoFlashcards(t)
	assertNoLinks(t)
	assertNoReminders(t)
	assertNoMedias(t)

	// Init the file
	f, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
	require.NoError(t, err)

	err = CurrentDB().BeginTransaction()
	require.NoError(t, err)
	err = f.Save()
	require.NoError(t, err)
	for _, object := range f.SubObjects() {
		err := object.Save()
		require.NoError(t, err)
	}
	err = CurrentDB().CommitTransaction()
	require.NoError(t, err)
	require.Equal(t, 1, mustCountFiles(t))
	require.Equal(t, 3, mustCountNotes(t))
	require.Equal(t, 1, mustCountMedias(t))
	require.Equal(t, 1, mustCountFlashcards(t))
	require.Equal(t, 1, mustCountLinks(t))
	require.Equal(t, 1, mustCountReminders(t))

	// Check the file
	actual, err := CurrentCollection().LoadFileByPath(f.RelativePath)
	require.NoError(t, err)
	assert.NotEqual(t, "", actual.OID)
	assert.Equal(t, "go.md", actual.RelativePath)
	assert.Equal(t, "go", actual.Wikilink)
	expectedFrontMatter, err := f.FrontMatterString()
	assert.NoError(t, err)
	actualFrontMatter, err := actual.FrontMatterString()
	assert.NoError(t, err)
	assert.Equal(t, expectedFrontMatter, actualFrontMatter)
	assert.Equal(t, []string{"go"}, actual.GetTags())

	assert.Contains(t, actual.Body, "# Go", actual.Body)
	assert.Equal(t, 6, actual.BodyLine)
	assert.Equal(t, f.Mode, actual.Mode)
	assert.Equal(t, f.Size, actual.Size)
	assert.Equal(t, f.Hash, actual.Hash)
	assert.Equal(t, f.MTime, actual.MTime)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)

	// Check a note
	note, err := CurrentCollection().FindNoteByTitle("Reference: Golang History")
	require.NoError(t, err)
	assert.NotEqual(t, "", note.OID)
	assert.Equal(t, actual.OID, note.FileOID)
	assert.EqualValues(t, "", note.ParentNoteOID) // 0 = new, -1 = nil
	assert.Equal(t, KindReference, note.NoteKind)
	assert.Equal(t, "Reference: Golang History", note.Title)
	assert.Equal(t, "Golang History", note.LongTitle)
	assert.Equal(t, "Golang History", note.ShortTitle)
	assert.Equal(t, actual.RelativePath, note.RelativePath)
	assert.Equal(t, "go#Reference: Golang History", note.Wikilink)
	assert.Equal(t, map[string]interface{}{
		"source": "https://en.wikipedia.org/wiki/Go_(programming_language)",
		"tags":   []interface{}{"go", "history"},
		"title":  "Golang History",
	}, note.Attributes)
	assert.Equal(t, []string{"go", "history"}, note.Tags)
	assert.Equal(t, 8, note.Line)
	assert.Equal(t, "`#history`\n\n`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`\n\n[Golang](https://go.dev/doc/ \"#go/go\") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.", note.ContentRaw)
	assert.Equal(t, "0eba86c8b008c0222869ef5358d48ab8241ffc8e", note.Hash)
	assert.Equal(t, `[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.`, note.ContentMarkdown)
	assert.Equal(t, "<p><a href=\"https://go.dev/doc/\" target=\"_blank\" title=\"#go/go\">Golang</a> was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.</p>", note.ContentHTML)
	assert.Equal(t, "Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.", note.ContentText)
	assert.NotEmpty(t, note.CreatedAt)
	assert.NotEmpty(t, note.UpdatedAt)
	assert.Empty(t, note.DeletedAt)
	assert.NotEmpty(t, note.LastCheckedAt)

	// Check the flashcard
	flashcardNote, err := CurrentCollection().FindNoteByTitle("Flashcard: Golang Logo")
	require.NoError(t, err)
	flashcard, err := CurrentCollection().FindFlashcardByShortTitle("Golang Logo")
	require.NoError(t, err)
	assert.NotEqual(t, "", flashcard.OID)
	assert.Equal(t, "Golang Logo", flashcard.ShortTitle)
	assert.EqualValues(t, actual.OID, flashcard.FileOID)
	assert.Equal(t, flashcardNote.OID, flashcard.NoteOID)
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
	assert.Equal(t, "A **gopher**.\n\n![Logo](oid:4044044044044044044044044044044044044040)", flashcard.BackMarkdown) // Must Refresh
	assert.Equal(t, "<p>What does the <strong>Golang logo</strong> represent?</p>", flashcard.FrontHTML)
	assert.Equal(t, "<p>A <strong>gopher</strong>.</p>\n\n<p><media oid=\"4044044044044044044044044044044044044040\" alt=\"Logo\" /></p>", flashcard.BackHTML)
	assert.Equal(t, "What does the Golang logo represent?", flashcard.FrontText)
	assert.Equal(t, "A gopher.\n\n![Logo](oid:4044044044044044044044044044044044044040)", flashcard.BackText)
	assert.NotEmpty(t, flashcard.CreatedAt)
	assert.NotEmpty(t, flashcard.UpdatedAt)
	assert.Empty(t, flashcard.DeletedAt)
	assert.NotEmpty(t, flashcard.LastCheckedAt)

	// Check the media
	media, err := CurrentCollection().FindMediaByRelativePath("medias/go.svg")
	require.NoError(t, err)
	assert.NotEqual(t, "", media.OID)
	assert.Equal(t, "medias/go.svg", media.RelativePath)
	assert.Equal(t, KindPicture, media.MediaKind)
	assert.Equal(t, false, media.Dangling)
	assert.Equal(t, ".svg", media.Extension)
	assert.NotEmpty(t, media.MTime)
	assert.Equal(t, "0cd82f33352563c9cf918d9f4fa0504cc6b84526", media.Hash)
	assert.EqualValues(t, 2288, media.Size)
	assert.NotEmpty(t, media.Mode)
	assert.NotEmpty(t, media.CreatedAt)
	assert.NotEmpty(t, media.UpdatedAt)
	assert.Empty(t, media.DeletedAt)
	assert.NotEmpty(t, media.LastCheckedAt)

	// Check the link
	links, err := CurrentCollection().FindLinksByText("Golang")
	require.NoError(t, err)
	require.Len(t, links, 1)
	link := links[0]
	linkNote, err := CurrentCollection().FindNoteByTitle("Reference: Golang History")
	require.NoError(t, err)
	assert.NotEqual(t, "", link.OID)
	assert.Equal(t, linkNote.OID, link.NoteOID)
	assert.Equal(t, "Golang", link.Text)
	assert.Equal(t, "https://go.dev/doc/", link.URL)
	assert.Equal(t, "", link.Title)
	assert.Equal(t, "go", link.GoName)
	assert.NotEmpty(t, link.CreatedAt)
	assert.NotEmpty(t, link.UpdatedAt)
	assert.Empty(t, link.DeletedAt)
	assert.NotEmpty(t, link.LastCheckedAt)

	// Check the reminder
	reminders, err := CurrentCollection().FindReminders()
	require.NoError(t, err)
	require.Len(t, reminders, 1)
	reminder := reminders[0]
	reminderNote, err := CurrentCollection().FindNoteByTitle("TODO: Conferences")
	require.NoError(t, err)
	assert.NotEqual(t, "", reminder.OID)
	assert.Equal(t, reminderNote.OID, reminder.NoteOID)
	assert.Equal(t, reminderNote.FileOID, reminder.FileOID)
	assert.Equal(t, "[Gophercon Europe](https://gophercon.eu/)", reminder.DescriptionRaw)
	assert.Equal(t, "[Gophercon Europe](https://gophercon.eu/)", reminder.DescriptionMarkdown)
	assert.Equal(t, "<p><a href=\"https://gophercon.eu/\" target=\"_blank\">Gophercon Europe</a></p>", reminder.DescriptionHTML)
	assert.Equal(t, "Gophercon Europe", reminder.DescriptionText)
	assert.Equal(t, "#reminder-2023-06-26", reminder.Tag)
	assert.Empty(t, reminder.LastPerformedAt)
	assert.EqualValues(t, time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC), reminder.NextPerformedAt)
	assert.NotEmpty(t, reminder.CreatedAt)
	assert.NotEmpty(t, reminder.UpdatedAt)
	assert.Empty(t, reminder.DeletedAt)
	assert.NotEmpty(t, reminder.LastCheckedAt)
}

func TestFile(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		SetUpCollectionFromTempDir(t)

		// Preconditions
		require.False(t, CurrentConfig().LintFile.IsInheritableAttribute("source", "index.md"))

		parent := NewFileFromAttributes(nil, "index.md", []Attribute{
			{Key: "tags", Value: "go"},
			{Key: "source", Value: "go.dev"}, // not inheritable
		})
		child := NewFileFromAttributes(parent, "go.md", []Attribute{
			{Key: "tags", Value: "language"},
		})
		actual := child.GetAttributes()
		expected := map[string]interface{}{
			"source": "go.dev", // File attributes are inheritable between files.
			"tags":   []interface{}{"go", "language"},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("YAML", func(t *testing.T) {
		// Make tests reproductible
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		root := SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

		fileSrc, err := NewFileFromPath(nil, filepath.Join(root, "go.md"))
		require.NoError(t, err)
		fileSrc.MTime = clock.Now()

		// Marshall
		buf := new(bytes.Buffer)
		err = fileSrc.Write(buf)
		require.NoError(t, err)
		noteYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: go.md
wikilink: go
front_matter:
    tags:
        - go
attributes:
    tags:
        - go
body: |-
    # Go

    ## Reference: Golang History

    `+"`"+`#history`+"`"+`

    `+"`"+`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`+"`"+`

    [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.


    ## Flashcard: Golang Logo

    What does the **Golang logo** represent?

    ---

    A **gopher**.

    ![Logo](./medias/go.svg)


    ## TODO: Conferences

    * [Gophercon Europe](https://gophercon.eu/) `+"`"+`#reminder-2023-06-26`+"`"+`
body_line: 6
mode: 420
size: 463
hash: 23334328153429ce5ba99acd83181b06c44f30af
mtime: 2023-01-01T01:12:30Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(noteYAML))

		// Unmarshall
		fileDest := new(File)
		err = fileDest.Read(buf)
		require.NoError(t, err)

		// Compare ignoreing a few attributes
		fileSrc.FrontMatter = nil
		fileDest.FrontMatter = nil
		fileSrc.new = false
		fileSrc.stale = false
		assert.EqualValues(t, fileSrc, fileDest)
	})

}

func TestInheritance(t *testing.T) {
	SetUpCollectionFromGoldenDir(t)

	err := CurrentCollection().Add(".")
	require.NoError(t, err)

	// Check how attributes are inherited in files
	fileIndex, err := CurrentCollection().LoadFileByPath("index.md")
	require.NoError(t, err)
	require.NotNil(t, fileIndex)
	fileGoIndex, err := CurrentCollection().LoadFileByPath("skills/go/index.md")
	require.NoError(t, err)
	require.NotNil(t, fileGoIndex)
	fileGoGeneral, err := CurrentCollection().LoadFileByPath("skills/go/general.md")
	require.NoError(t, err)
	require.NotNil(t, fileGoGeneral)
	fileGoGoroutines, err := CurrentCollection().LoadFileByPath("skills/go/goroutines.md")
	require.NoError(t, err)
	require.NotNil(t, fileGoGoroutines)

	assert.EqualValues(t, map[string]interface{}{
		"source": "https://github.com/julien-sobczak/the-notewriter",
		"tags":   []interface{}{"test"},
	}, fileIndex.GetAttributes())
	assert.EqualValues(t, map[string]interface{}{
		"tags": []interface{}{"go"},
	}, fileGoIndex.GetAttributes())
	assert.EqualValues(t, map[string]interface{}{
		"subject": "language",
		"tags":    []interface{}{"go", "programming"},
	}, fileGoGeneral.GetAttributes())
	assert.EqualValues(t, map[string]interface{}{
		"tags": []interface{}{"go"},
	}, fileGoGoroutines.GetAttributes())

	// Check how attributes and tags are inherited in notes
	generalNotes, err := CurrentCollection().SearchNotes(`path:"skills/go/general.md"`)
	require.NoError(t, err)
	require.Len(t, generalNotes, 2)
	goroutinesNotes, err := CurrentCollection().SearchNotes(`path:"skills/go/goroutines.md"`)
	require.NoError(t, err)
	require.Len(t, goroutinesNotes, 1)

	historyNote := generalNotes[0]
	require.Equal(t, "History", historyNote.ShortTitle)
	assert.EqualValues(t, []string{"go", "programming", "history"}, historyNote.GetTags())
	assert.EqualValues(t, map[string]interface{}{
		"subject": "language", // Inherited from file
		"source":  "https://en.wikipedia.org/wiki/Go_(programming_language)",
		"tags":    []interface{}{"go", "programming", "history"},
		"title":   "History", // Copied from note's title
	}, historyNote.GetAttributes())

	creationNote := generalNotes[1]
	require.Equal(t, "Golang creation", creationNote.ShortTitle)
	assert.EqualValues(t, []string{"go", "programming", "history"}, creationNote.GetTags())
	assert.EqualValues(t, map[string]interface{}{
		"subject": "language",
		// source is not inheritable
		"tags":  []interface{}{"go", "programming", "history"}, // Inherited from parent note
		"title": "Golang Creation",                             // Specific attribute is preserved if defined explicitely
	}, creationNote.GetAttributes())

	goroutineNote := goroutinesNotes[0]
	require.Equal(t, "Start a goroutine", goroutineNote.ShortTitle)
	assert.EqualValues(t, []string{"go"}, goroutineNote.GetTags())
	assert.EqualValues(t, map[string]interface{}{
		"example": "https://go.dev/tour/concurrency/1",
		"tags":    []interface{}{"go"},
		"title":   "Start a goroutine", // Specific attribute is preserved if defined explicitely
	}, goroutineNote.GetAttributes())
}

func TestFeatures(t *testing.T) {

	t.Run("Relations", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestRelations")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		fileA, err := CurrentCollection().FindFileByWikilink("a")
		require.NoError(t, err)
		assert.NotNil(t, fileA)
		fileB, err := CurrentCollection().FindFileByWikilink("b")
		require.NoError(t, err)
		assert.NotNil(t, fileB)
		fileC, err := CurrentCollection().FindFileByWikilink("c")
		require.NoError(t, err)
		assert.NotNil(t, fileC)

		notesA, err := CurrentCollection().SearchNotes(`path:"a.md"`)
		require.NoError(t, err)
		require.Len(t, notesA, 1)
		noteA := notesA[0]
		notesB, err := CurrentCollection().SearchNotes(`path:"b.md"`)
		require.NoError(t, err)
		require.Len(t, notesB, 1)
		noteB := notesB[0]
		notesC, err := CurrentCollection().SearchNotes(`path:"c.md"`)
		require.NoError(t, err)
		require.Len(t, notesC, 1)
		noteC := notesC[0]

		relationsA := noteA.Relations()
		relationsB := noteB.Relations() // FIXME now DEBUG why only one attribute value for `@inspiration`
		relationsC := noteC.Relations()

		expectedA := []*Relation{
			{
				SourceOID:  noteA.OID,
				SourceKind: "note",
				TargetOID:  noteB.OID,
				TargetKind: "note",
				Type:       "referenced_by",
			},
		}
		expectedB := []*Relation{
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  noteA.OID,
				TargetKind: "note",
				Type:       "inspired_by",
			},
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  fileC.OID,
				TargetKind: "file",
				Type:       "inspired_by",
			},
		}
		expectedC := []*Relation{
			{
				SourceOID:  noteC.OID,
				SourceKind: "note",
				TargetOID:  fileA.OID,
				TargetKind: "file",
				Type:       "references",
			},
		}

		assert.Equal(t, expectedA, relationsA)
		assert.Equal(t, expectedB, relationsB)
		assert.Equal(t, expectedC, relationsC)

		// Check at least one relation exists to detect complete failure
		count, err := CurrentCollection().CountRelations()
		require.NoError(t, err)
		require.Greater(t, count, 0)

		// Check relations between objects
		relationsFromFileA, err := CurrentCollection().FindRelationsFrom(fileA.OID)
		require.NoError(t, err)
		relationsToFileA, err := CurrentCollection().FindRelationsTo(fileA.OID)
		require.NoError(t, err)
		relationsFromNoteA, err := CurrentCollection().FindRelationsFrom(noteA.OID)
		require.NoError(t, err)
		relationsToNoteA, err := CurrentCollection().FindRelationsTo(noteA.OID)
		require.NoError(t, err)

		relationsFromFileB, err := CurrentCollection().FindRelationsFrom(fileB.OID)
		require.NoError(t, err)
		relationsToFileB, err := CurrentCollection().FindRelationsTo(fileB.OID)
		require.NoError(t, err)
		relationsFromNoteB, err := CurrentCollection().FindRelationsFrom(noteB.OID)
		require.NoError(t, err)
		relationsToNoteB, err := CurrentCollection().FindRelationsTo(noteB.OID)
		require.NoError(t, err)

		relationsFromFileC, err := CurrentCollection().FindRelationsFrom(fileC.OID)
		require.NoError(t, err)
		relationsToFileC, err := CurrentCollection().FindRelationsTo(fileC.OID)
		require.NoError(t, err)
		relationsFromNoteC, err := CurrentCollection().FindRelationsFrom(noteC.OID)
		require.NoError(t, err)
		relationsToNoteC, err := CurrentCollection().FindRelationsTo(noteC.OID)
		require.NoError(t, err)

		expectedFromFileA := []*Relation{}
		expectedToFileA := []*Relation{
			{
				SourceOID:  noteC.OID,
				SourceKind: "note",
				TargetOID:  fileA.OID,
				TargetKind: "file",
				Type:       "references",
			},
		}
		expectedFromNoteA := []*Relation{
			{
				SourceOID:  noteA.OID,
				SourceKind: "note",
				TargetOID:  noteB.OID,
				TargetKind: "note",
				Type:       "referenced_by",
			},
		}
		expectedToNoteA := []*Relation{
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  noteA.OID,
				TargetKind: "note",
				Type:       "inspired_by",
			},
		}
		assert.ElementsMatch(t, expectedFromFileA, relationsFromFileA)
		assert.ElementsMatch(t, expectedToFileA, relationsToFileA)
		assert.ElementsMatch(t, expectedFromNoteA, relationsFromNoteA)
		assert.ElementsMatch(t, expectedToNoteA, relationsToNoteA)

		expectedFromFileB := []*Relation{}
		expectedToFileB := []*Relation{}
		expectedFromNoteB := []*Relation{
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  noteA.OID,
				TargetKind: "note",
				Type:       "inspired_by",
			},
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  fileC.OID,
				TargetKind: "file",
				Type:       "inspired_by",
			},
		}
		expectedToNoteB := []*Relation{
			{
				SourceOID:  noteA.OID,
				SourceKind: "note",
				TargetOID:  noteB.OID,
				TargetKind: "note",
				Type:       "referenced_by",
			},
		}
		assert.ElementsMatch(t, expectedFromFileB, relationsFromFileB)
		assert.ElementsMatch(t, expectedToFileB, relationsToFileB)
		assert.ElementsMatch(t, expectedFromNoteB, relationsFromNoteB)
		assert.ElementsMatch(t, expectedToNoteB, relationsToNoteB)

		expectedFromFileC := []*Relation{}
		expectedToFileC := []*Relation{
			{
				SourceOID:  noteB.OID,
				SourceKind: "note",
				TargetOID:  fileC.OID,
				TargetKind: "file",
				Type:       "inspired_by",
			},
		}
		expectedFromNoteC := []*Relation{
			{
				SourceOID:  noteC.OID,
				SourceKind: "note",
				TargetOID:  fileA.OID,
				TargetKind: "file",
				Type:       "references",
			},
		}
		expectedToNoteC := []*Relation{}
		assert.ElementsMatch(t, expectedFromFileC, relationsFromFileC)
		assert.ElementsMatch(t, expectedToFileC, relationsToFileC)
		assert.ElementsMatch(t, expectedFromNoteC, relationsFromNoteC)
		assert.ElementsMatch(t, expectedToNoteC, relationsToNoteC)
	})

	t.Run("Ignore", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestIgnore")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		fileInclude, err := CurrentCollection().FindFileByWikilink("include")
		require.NoError(t, err)
		require.NotNil(t, fileInclude)
		fileIgnore, err := CurrentCollection().FindFileByWikilink("ignore")
		require.NoError(t, err)
		require.NotNil(t, fileIgnore)
		fileIncludeIgnore, err := CurrentCollection().FindFileByWikilink("include-ignore")
		require.NoError(t, err)
		require.NotNil(t, fileIncludeIgnore)

		notesInclude, err := CurrentCollection().SearchNotes(`path:"include.md"`)
		require.NoError(t, err)
		require.Len(t, notesInclude, 1)
		assert.Equal(t, "Include", notesInclude[0].ShortTitle)

		notesIncludeIgnore, err := CurrentCollection().SearchNotes(`path:"include-ignore.md"`)
		require.NoError(t, err)
		require.Len(t, notesIncludeIgnore, 2)
		assert.Equal(t, "Include", notesIncludeIgnore[0].ShortTitle)
		assert.Equal(t, "Include", notesIncludeIgnore[1].ShortTitle)
	})

}

func TestPostProcessing(t *testing.T) {
	SetUpCollectionFromGoldenDir(t)
	UseSequenceOID(t)

	err := CurrentCollection().Add(".")
	require.NoError(t, err)

	t.Run("HTML Comments", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`path:"quotes/walt-disney.md"`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		assert.Equal(t, `<h1>On Doing</h1>`, note.TitleHTML)
		assert.Equal(t, `<figure>
	<blockquote>
		<p>The way to get started is to quit talking and begin doing.</p>
	</blockquote>
	<figcaption>— Walt Disney <cite>undefined</cite></figcaption>
</figure>`, note.ContentHTML)
	})

	t.Run("Quotes Formatting", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`path:"quotes/walt-disney.md"`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		assert.Equal(t, `<h1>On Doing</h1>`, note.TitleHTML)
		assert.Equal(t, `<figure>
	<blockquote>
		<p>The way to get started is to quit talking and begin doing.</p>
	</blockquote>
	<figcaption>— Walt Disney <cite>undefined</cite></figcaption>
</figure>`, note.ContentHTML)
	})

	t.Run("Attributed Quotes Formatting", func(t *testing.T) {
		// See https://developer.mozilla.org/en-US/docs/Web/HTML/Element/cite
		notes, err := CurrentCollection().SearchNotes(`kind:quote @title:"J.R.R. Tolkein on Life"`)
		require.NoError(t, err)
		note := notes[0]
		assert.Equal(t, `<h1>J.R.R. Tolkein on Life</h1>`, note.TitleHTML)
		assert.Equal(t, `<figure>
	<blockquote>
		<p>All we have to decide is what to do with the time that is given us.</p>
	</blockquote>
	<figcaption>— J.R.R. Tolkein <cite><em>The Fellowship of the Ring</em></cite></figcaption>
</figure>`, note.ContentHTML)
	})

	t.Run("Note Inclusion", func(t *testing.T) {
		// File.GetNotes() <= Dependent Files are not saved inside this method
		// so if one note depends on another in the same file, the file will not exist in DB...
		// TODO save relations in Note.Save() to trigger Refresh()
		notes, err := CurrentCollection().SearchNotes(`@title:"Commonplace book"`)
		require.NoError(t, err)
		note := notes[0]
		assert.Equal(t, `<h1>Commonplace book</h1>`, note.TitleHTML)
		assert.Equal(t, `<p>Commonplace books compile knowledge by writing information into books.</p>

<p><em>Hypomnema</em> is a Greek word with several translations into English including a reminder, a note, a public record, a commentary, an anecdotal record, a draft, a copy, and other variations on those terms.</p>`, note.ContentHTML)
	})

	t.Run("Quote Inclusion", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`kind:note @title:"On Doing"`)
		require.NoError(t, err)
		note := notes[0]
		assert.Equal(t, `<h1>On Doing</h1>`, note.TitleHTML)
		assert.Equal(t, `<figure>
	<blockquote>
		<p>The way to get started is to quit talking and begin doing.</p>
	</blockquote>
	<figcaption>— Walt Disney <cite>undefined</cite></figcaption>
</figure>

<p>Only motion can get you closer to your goal.</p>`, note.ContentHTML)
	})

	t.Run("Media URL Replacement", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`@title:Gopher`)
		require.NoError(t, err)
		note := notes[0]
		assert.Equal(t, `Gophers are rodents of the family **Geomyidae.**

![A Gopher](oid:4044044044044044044044044044044044044040)

The Golang programming language uses the image of a gopher as logo:

![Golang Logo](oid:0000000000000000000000000000000000000012)`, note.ContentMarkdown)
	})

	t.Run("Comment Formatting", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`@title:"Allen Saunders on Life"`)
		require.NoError(t, err)
		note := notes[0]

		// Check Markdown
		assert.Equal(t, `> Life is what happens when you're busy making other plans.
> — Allen Saunders`, note.ContentMarkdown)
		assert.Equal(t, `Life is about doing.`, note.CommentMarkdown)

		// Check HTML
		assert.Equal(t, `<figure>
	<blockquote>
		<p>Life is what happens when you&rsquo;re busy making other plans.</p>
	</blockquote>
	<figcaption>— Allen Saunders</figcaption>
</figure>`, note.ContentHTML)
		assert.Equal(t, `Life is about doing.`, note.CommentHTML)

		// Check Text
		assert.Equal(t, `> Life is what happens when you're busy making other plans.
> — Allen Saunders`, note.ContentText)
		assert.Equal(t, `Life is about doing.`, note.CommentText)

	})

	t.Run("Asciidoc Text Replacements", func(t *testing.T) {
		notes, err := CurrentCollection().SearchNotes(`@title:"Asciidoc Text replacements"`)
		require.NoError(t, err)
		note := notes[0]
		assert.Equal(t, strings.TrimSpace(`
* Copyright: © ©
* Registered: ® ®
* Trademark: ™ ™
* Em dash: — —
* Ellipses: … …
* Single right arrow: → →
* Double right arrow: ⇒ ⇒
* Single left arrow: ← ←
* Double left arrow: ⇐ ⇐

Except when present in code block like `+"`i--`"+` or:

`+"```c"+`
i--
`+"```"+`
`), note.ContentMarkdown)
	})

}

func TestParseNotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string                          // input
		checkFn func(*testing.T, []*ParsedNote) // output
	}{

		{
			name: "Comments are not notes",
			input: `
# Blocks

## Note: First

This note contains a block containing a comment:

` + "```yaml" + `

# A basic document

key: value
` + "```" + `

## Note: Second

This note contains nothing interesting.
`,
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 2) // The comment inside the block code must not be considered like a free note
				assert.Equal(t, "First", notes[0].ShortTitle)
				assert.Equal(t, "Second", notes[1].ShortTitle)
			},
		},

		{
			name: "Free Note File",
			input: `# My Project

## TODO

* [ ] Add licence GNU GPL
`,
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 1) // No typed notes present inside the doc
				assert.Equal(t, "My Project", notes[0].ShortTitle)
				assert.Equal(t, KindFree, notes[0].Kind)
			},
		},

		{
			name: "Free Note Inside File",
			input: `# My Project

## TODO: A TODO Note

* [ ] Add licence GNU GPL

## A Free Note

### Subsection

This is a subsection of the free note.

## Reference: A Reference Note

### Subsection

This is a subsection of the reference note.
`,
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 3)
				assert.Equal(t, "A TODO Note", notes[0].ShortTitle)
				assert.Equal(t, "A Free Note", notes[1].ShortTitle)
				assert.Equal(t, "A Reference Note", notes[2].ShortTitle)
			},
		},

		{
			name: "Category sections are not free notes",
			input: `
# My Project

## TODO: Backlog

* [ ] Add license GNU GPL

## Features

### Snippet: Idea A

Presentation of idea A

### Snippet: Idea B

Presentation of idea B
`,
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 3)
				assert.Equal(t, "Backlog", notes[0].ShortTitle)
				assert.Equal(t, "Idea A", notes[1].ShortTitle)
				assert.Equal(t, "Idea B", notes[2].ShortTitle)
			},
		},

		{
			name: "Code comments are not notes",
			input: "# S3\n" +
				"\n" +
				"## Cheatsheet: Minio CLI `mc`\n" +
				"\n" +
				"```shell\n" +
				"# Create files under $HOME/.mc\n" +
				"$ mc config host add minio http://10.45.32.192/ <api_key> <secret_key> --api S3v4\n" +
				"\n" +
				"# List all buckets\n" +
				"$ mc ls minio\n" +
				"```\n",
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 1)
			},
		},

		{
			name: "Markdown code documents are ignored",
			input: "" +
				"## Note: Templates\n" +
				"\n" +
				"Flashcards supports Front/Back notes:\n" +
				"\n" +
				"```md\n" +
				"## Flashcard A\n" +
				"\n" +
				"What is A?\n" +
				"---\n" +
				"The first letter of the alphabet\n" +
				"\n" +
				"## Flashcard B\n" +
				"\n" +
				"What is B?\n" +
				"---\n" +
				"The second letter of the alphabet\n" +
				"```\n",
			checkFn: func(t *testing.T, notes []*ParsedNote) {
				require.Len(t, notes, 1)
				assert.Equal(t, "Templates", notes[0].ShortTitle)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetUpCollectionFromTempDir(t)
			notes := ParseNotes(tt.input)
			tt.checkFn(t, notes)
		})
	}
}

func TestParseFileComplex(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestComplex")
	file, err := ParseFile(filepath.Join(root, "syntax.md"))
	require.NoError(t, err)

	notes := ParseNotes(file.Body)

	// Check note by note
	var note *ParsedNote

	note = notes[0]
	assert.Equal(t, &ParsedNote{
		Kind:       "note",
		Level:      2,
		LongTitle:  "Note: Markdown in Markdown",
		ShortTitle: "Markdown in Markdown",
		Line:       27 - file.BodyLine + 1,
		NoteTags:   nil,
		NoteAttributes: map[string]interface{}{
			"source": "https://en.wikipedia.org/wiki/Markdown",
		},
	}, ignoreNoteBody(note))

	note = notes[1]
	assert.Equal(t, &ParsedNote{
		Kind:       "cheatsheet",
		Level:      2,
		LongTitle:  "Cheatsheet: How to include HTML in Markdown",
		ShortTitle: "How to include HTML in Markdown",
		Line:       44 - file.BodyLine + 1,
		NoteTags:   []string{"html"},
		NoteAttributes: map[string]interface{}{
			"tags": []interface{}{"html"},
		},
	}, ignoreNoteBody(note))

	note = notes[2]
	assert.Equal(t, &ParsedNote{
		Kind:       "note",
		Level:      2,
		LongTitle:  "Note: A",
		ShortTitle: "A",
		Line:       55 - file.BodyLine + 1,
		NoteTags:   []string{"tag-a"},
		NoteAttributes: map[string]interface{}{
			"tags":   []interface{}{"tag-a"},
			"source": "https://www.markdownguide.org/basic-syntax/#headings",
		},
	}, ignoreNoteBody(note))

	note = notes[3]
	assert.Equal(t, &ParsedNote{
		Kind:       "note",
		Level:      3,
		LongTitle:  "Note: B",
		ShortTitle: "B",
		Line:       63 - file.BodyLine + 1,
		NoteTags:   []string{"tag-b1", "tag-b2"},
		NoteAttributes: map[string]interface{}{
			"tags": []interface{}{"tag-b1", "tag-b2"},
		},
	}, ignoreNoteBody(note))

	note = notes[4]
	assert.Equal(t, &ParsedNote{
		Kind:       "note",
		Level:      4,
		LongTitle:  "Note: C",
		ShortTitle: "C",
		Line:       69 - file.BodyLine + 1,
		NoteTags:   nil,
		NoteAttributes: map[string]interface{}{
			"source": "https://www.markdownguide.org/basic-syntax/#headings",
		},
	}, ignoreNoteBody(note))

	note = notes[5]
	assert.Equal(t, &ParsedNote{
		Kind:       "note",
		Level:      5,
		LongTitle:  "Note: D",
		ShortTitle: "D",
		Line:       75 - file.BodyLine + 1,
		NoteTags:   []string{"tag-d"},
		NoteAttributes: map[string]interface{}{
			"tags": []interface{}{"tag-d"},
		},
	}, ignoreNoteBody(note))

	note = notes[6]
	assert.Equal(t, &ParsedNote{
		Kind:           "note",
		Level:          6,
		LongTitle:      "Note: E",
		ShortTitle:     "E",
		Line:           81 - file.BodyLine + 1,
		NoteTags:       nil,
		NoteAttributes: map[string]interface{}{},
	}, ignoreNoteBody(note))

	note = notes[7]
	assert.Equal(t, &ParsedNote{
		Kind:           "todo",
		Level:          2,
		LongTitle:      "TODO: List",
		ShortTitle:     "List",
		Line:           86 - file.BodyLine + 1,
		NoteTags:       nil,
		NoteAttributes: map[string]interface{}{},
	}, ignoreNoteBody(note))

	note = notes[8]
	assert.Equal(t, &ParsedNote{
		Kind:           "note",
		Level:          2,
		LongTitle:      "Note: Comments",
		ShortTitle:     "Comments",
		Line:           100 - file.BodyLine + 1,
		NoteTags:       nil,
		NoteAttributes: map[string]interface{}{},
	}, ignoreNoteBody(note))

	note = notes[9]
	assert.Equal(t, &ParsedNote{
		Kind:       "quote",
		Level:      2,
		LongTitle:  "Quote: Richly Annotated Quote",
		ShortTitle: "Richly Annotated Quote",
		Line:       116 - file.BodyLine + 1,
		NoteTags:   []string{"life", "doing", "life-changing", "courage"},
		NoteAttributes: map[string]interface{}{
			"name":        "Christine Mason Miller",
			"nationality": "American",
			"occupation":  "author",
			"tags":        []interface{}{"life", "doing", "life-changing", "courage"},
		},
	}, ignoreNoteBody(note))
}

/* Test Helpers */

func ignoreNoteBody(note *ParsedNote) *ParsedNote {
	var res ParsedNote
	copier.Copy(&res, note)
	res.Body = ""
	return &res
}
