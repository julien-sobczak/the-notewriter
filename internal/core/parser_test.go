package core_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileWithTestdata(t *testing.T) {
	testutil.FreezeNow(t)
	testutil.FreezeFileInfoReader(t)

	testcases := []struct {
		name   string
		golden string
		test   func(t *testing.T, file *core.ParsedFile)
	}{
		{
			name:   "Basic",
			golden: "basic",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				// We check everything in this basic file
				// so that following tests can focus on specificities

				// Check file
				assert.NotEmpty(t, file.RepositoryPath)
				assert.NotEmpty(t, file.AbsolutePath)
				assert.NotEmpty(t, file.RelativePath)
				assert.True(t, strings.HasPrefix(file.AbsolutePath, file.RepositoryPath))
				assert.True(t, strings.HasSuffix(file.AbsolutePath, file.RelativePath))
				assert.Equal(t, "basic-notetaking", file.Slug)
				assert.Equal(t, "Basic Note-Taking", file.Title.String())
				assert.Equal(t, "Basic Note-Taking", file.ShortTitle.String())

				// File attributes extracted from the Front Matter
				assert.Equal(t, core.AttributeSet(map[string]any{
					"title":  "Basic Note-Taking",
					"rating": int64(5),
					"slug":   "basic-notetaking",
					"tags":   []string{"thinking"},
				}), file.FileAttributes)

				// Check subobjects
				assert.Len(t, file.Medias, 1)
				assert.Len(t, file.Notes, 4)

				// Check media "da-vinci-notebook.png"
				mediaDaVinci, ok := file.FindMediaByFilename("da-vinci-notebook.png")
				require.True(t, ok)
				expectedDaVinci := &core.ParsedMedia{
					RawPath:      "medias/da-vinci-notebook.png",
					AbsolutePath: filepath.Join(filepath.Dir(file.Markdown.AbsolutePath), "medias/da-vinci-notebook.png"),
					RelativePath: "medias/da-vinci-notebook.png",
					Extension:    ".png",
					MediaKind:    core.KindPicture,
					// File existence must also be checked
					Dangling: false,
					MTime:    clock.Now(),
					Size:     1,
					Line:     33,
				}
				require.EqualExportedValues(t, *expectedDaVinci, *mediaDaVinci)
				assert.WithinDuration(t, time.Now(), mediaDaVinci.FileMTime(), 1*time.Minute) // test cases are copied in a temp directory
				assert.Greater(t, mediaDaVinci.FileSize(), int64(0))

				// Check "Note: A Note"
				noteNote, ok := file.FindNoteByShortTitle("A Note")
				require.True(t, ok)
				assert.Equal(t, 2, noteNote.Level)
				assert.Equal(t, "Note", noteNote.Type)
				assert.Equal(t, "basic-notetaking-note-a-note", noteNote.Slug)
				assert.Equal(t, markdown.Document("Note: A Note"), noteNote.Title)
				assert.Equal(t, markdown.Document("A Note"), noteNote.ShortTitle)
				assert.Equal(t, 11, noteNote.Line)
				assert.Equal(t, "## Note: A Note\n\nNotes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Content.String())
				assert.Equal(t, "Notes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Body.String())
				assert.Equal(t, core.AttributeSet{
					"rating": int64(5),
					"tags":   []string{"thinking"},
					"title":  "A Note",
				}, noteNote.Attributes)
				// No subobjects
				assert.Len(t, noteNote.Flashcards, 0)
				assert.Len(t, noteNote.GoLinks, 0)
				assert.Len(t, noteNote.Reminders, 0)

				// Check "Quote: Tim Ferris on Note-Taking"
				noteTimFerris, ok := file.FindNoteByShortTitle("Tim Ferris on Note-Taking")
				require.True(t, ok)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Tim Ferris",
				}), noteTimFerris.NoteAttributes)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Tim Ferris",
					"rating": int64(5),
					"tags":   []string{"thinking"},
					"title":  "Tim Ferris on Note-Taking",
				}), noteTimFerris.Attributes)

				// Check "Flashcard: Commonplace Book"
				noteCommomplace, ok := file.FindNoteByShortTitle("Commonplace Book")
				require.True(t, ok)
				require.Len(t, noteCommomplace.Flashcards, 1)
				flashcardCommonplace := noteCommomplace.Flashcards[0]
				assert.Equal(t, "Commonplace Book", flashcardCommonplace.ShortTitle.String())
				assert.Equal(t, "(Thinking) What are **commonplace books**?", flashcardCommonplace.Front.String())
				assert.Equal(t, "A tool to compile knowledge, usually by writing information into books.", flashcardCommonplace.Back.String())

				// Check "Note: Leonardo da Vinci's Notebooks"
				noteDaVinci, ok := file.FindNoteByShortTitle("Leonardo da Vinci's Notebooks")
				require.True(t, ok)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Leonardo da Vinci",
					"year":   "~1510",
				}), noteDaVinci.NoteAttributes)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Leonardo da Vinci",
					"rating": int64(5),
					"tags":   []string{"thinking"},
					"year":   "~1510",
					"title":  "Leonardo da Vinci's Notebooks",
				}), noteDaVinci.Attributes)
			},
		},

		{
			name:   "Characters Replacement",
			golden: "characters-replacement",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)
				noteAsciidoc, ok := file.FindNoteByShortTitle("Asciidoc Text replacements")
				require.True(t, ok)

				// Original text is preserved in original content only
				assert.Contains(t, noteAsciidoc.Content, `(C)`)
				assert.NotContains(t, noteAsciidoc.Body, `(C)`)

				assert.Contains(t, noteAsciidoc.Body, strings.TrimSpace(`
* Copyright: © ©
* Registered: ® ®
* Trademark: ™ ™
* Em dash: — —
* Ellipses: … …
* Single right arrow: → →
* Double right arrow: ⇒ ⇒
* Single left arrow: ← ←
* Double left arrow: ⇐ ⇐`))
				// But code blocks must not have been modified
				assert.Contains(t, noteAsciidoc.Body, "`i--`")
				assert.Contains(t, noteAsciidoc.Body, "```c\ni--\n```")
			},
		},

		{
			name:   "Comment",
			golden: "comment",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				noteA, ok := file.FindNoteByShortTitle("A")
				require.True(t, ok)
				noteB, ok := file.FindNoteByShortTitle("B")
				require.True(t, ok)

				assert.Equal(t, `Some text inside the note.`, noteA.Body.String())
				assert.Equal(t, `Text`, noteB.Body.String())
			},
		},

		{
			name:   "Ignore",
			golden: "ignore",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.Nil(t, file)
				// Nothing more to check
			},
		},

		{
			name:   "Minimal",
			golden: "minimal",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				// Sub-headings must only be included when untyped
				// Ex:
				note, ok := file.FindNoteByTitle("Note: Blog Post Outline")
				require.True(t, ok)
				assert.Contains(t, note.Body.String(), "## Motivations")
				assert.Contains(t, note.Body.String(), "## Introduction")
				assert.Contains(t, note.Body.String(), "## Demo")
				// BUT
				note, ok = file.FindNoteByTitle("Note: First Notebooks")
				require.True(t, ok)
				assert.NotContains(t, note.Body.String(), "Flashcard: First Notebooks")

				// TODO complete
			},
		},

		{
			name:   "Shorthands",
			golden: "shorthands",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				// Check notes for modified titles and extracted attributes
				assert.Len(t, file.Notes, 2)
				note1 := file.Notes[0]
				note2 := file.Notes[1]

				assert.Equal(t, "Shorthands Demo / Support emojis in title as attributes", note1.LongTitle.String())
				assert.Equal(t, "Task: Support emojis in title as attributes", note1.Title.String())
				assert.Equal(t, "Support emojis in title as attributes", note1.ShortTitle.String())
				assert.Equal(t, core.AttributeSet(map[string]any{
					"status":   "in-progress",
					"priority": "high",
					"title":    "Support emojis in title as attributes",
				}), note1.Attributes)

				assert.Equal(t, "Shorthands Demo / Review _Building a Second Brain_ ★★★★", note2.LongTitle.String())
				assert.Equal(t, "Note: Review _Building a Second Brain_ ★★★★", note2.Title.String())
				assert.Equal(t, "Review _Building a Second Brain_ ★★★★", note2.ShortTitle.String())
				assert.Equal(t, core.AttributeSet(map[string]any{
					"rating": int64(8),
					"title":  "Review _Building a Second Brain_ ★★★★",
				}), note2.Attributes)
			},
		},

		{
			name:   "TOC",
			golden: "toc",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				// Should have a TOC note plus the actual content notes
				assert.Len(t, file.Notes, 6) // 1 TOC + 5 notes

				// First note should be the generated TOC
				tocNote := file.Notes[0]
				assert.Equal(t, "Table of Content", tocNote.Title.String())
				assert.Equal(t, "Table of Content", tocNote.ShortTitle.String())
				assert.Equal(t, "Table of Content", tocNote.LongTitle.String())
				assert.Equal(t, file.Slug+"-toc", tocNote.Slug)
				assert.Equal(t, 0, tocNote.Line) // Generated note has line number 0

				// Check TOC content structure
				actual := tocNote.Body.String()
				expected := `
* [[#Note: Main Concept]]
  * [[#Note: Sub-concept]]
  * [[#Flashcard: Definition]]
* Resources
  * [[#Quote: Famous Quote]]
  * [[#Note: Resource Note]]
`
				assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))

				// TOC should have no comment, no attributes, no tags
				assert.Empty(t, tocNote.Comment.String())
				assert.Empty(t, tocNote.NoteAttributes)
				assert.Empty(t, tocNote.NoteTags)
			},
		},

		// Add more test cases here to enrich Markdown support
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			tr := core.NewTestRepository(t, core.FromGoldenDirNamed("TestParser"))
			md, err := markdown.ParseFile(filepath.Join(tr.Root, testcase.golden+".md"))
			require.NoError(t, err)
			file, err := core.ParseFile(md, nil)
			require.NoError(t, err)
			testcase.test(t, file)
		})
	}
}

func TestParseFileWithTempdir(t *testing.T) {

	t.Run("Nested Flashcards", func(t *testing.T) {
		tr := core.NewTestRepository(t)

		// Create a file with nested flashcards at different levels.
		// We will check that the flashcards are correctly extracted without including sub-notes and siblings sections.
		tr.WriteFile("learning.md", `
---
tags: learning
---

# Learning

## Note: Rote Memorization

Rote memorization is a technique of learning by repetition.

### Flashcard: Rote Memorization

(Learning) What is **rote memorization**?

---

A technique of **learning by repetition**.

### Advantages

* It is a simple technique.
* It is effective for short-term memory.

### Drawbacks

* It is not effective for long-term memory.
* It does not promote understanding of the material.

#### Flashcard: Rote Memorization Limitations

(Learning) What are the **limitations of rote memorization**?

---

It is not effective for **long-term memory** and does not promote **understanding of the material**.

## Note: Spaced Repetition

Spaced repetition is a technique of learning by reviewing material at increasing intervals.

### Flashcard: Spaced Repetition

(Learning) What is **spaced repetition**?

---

A technique of **learning by reviewing material at increasing intervals**.

### Flashcard: Spaced Repetition History

(Learning) [{c1::Hermann Ebbinghaus::person}] invented **spaced repetition**
`)

		md := markdown.MustParseFile(filepath.Join(tr.Root, "learning.md"))
		index, err := core.ParseFile(md, nil)
		require.NoError(t, err)

		require.Len(t, index.Notes, 6)

		note1 := index.Notes[0]
		note2 := index.Notes[1]
		note3 := index.Notes[2]
		note4 := index.Notes[3]
		note5 := index.Notes[4]
		note6 := index.Notes[5]

		assert.Equal(t, "Note: Rote Memorization", note1.Title.String())
		assert.Equal(t, "Flashcard: Rote Memorization", note2.Title.String())
		assert.Equal(t, "Flashcard: Rote Memorization Limitations", note3.Title.String())
		assert.Equal(t, "Note: Spaced Repetition", note4.Title.String())
		assert.Equal(t, "Flashcard: Spaced Repetition", note5.Title.String())
		assert.Equal(t, "Flashcard: Spaced Repetition History", note6.Title.String())

		assert.Equal(t, "Rote memorization is a technique of learning by repetition.", note1.Body.String())
		assert.Equal(t, "(Learning) What is **rote memorization**?\n\n---\n\nA technique of **learning by repetition**.", note2.Body.String())
		assert.Equal(t, "(Learning) What are the **limitations of rote memorization**?\n\n---\n\nIt is not effective for **long-term memory** and does not promote **understanding of the material**.", note3.Body.String())
		assert.Equal(t, "Spaced repetition is a technique of learning by reviewing material at increasing intervals.", note4.Body.String())
		assert.Equal(t, "(Learning) What is **spaced repetition**?\n\n---\n\nA technique of **learning by reviewing material at increasing intervals**.", note5.Body.String())
		assert.Equal(t, "(Learning) [{c1::Hermann Ebbinghaus::person}] invented **spaced repetition**", note6.Body.String())
	})

	t.Run("Attributes & Tags", func(t *testing.T) {
		// Test attributes and tags defined in Front Matter and in notes
		// are correctly parsed and merged.
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)
		tr.WriteFile("index.md", `
---
name: Voltaire
occupation: writer, philosopher
nationality: French
tags: [philosophy]
---

# Voltaire


## Quote: On Appreciation

‛#being‛ ‛@source: Unknown‛

Appreciation is a wonderful thing: It makes what is excellent in others belong to us as well.


`)
		md := markdown.MustParseFile(filepath.Join(tr.Root, "index.md"))
		index, err := core.ParseFile(md, nil)
		require.NoError(t, err)

		require.Len(t, index.Notes, 1)

		// Check attributes and notes
		note := index.Notes[0]
		assert.Equal(t, core.AttributeSet(map[string]any{
			"name":        "Voltaire",
			"nationality": "French",
			"occupation":  "writer, philosopher",
			"source":      "Unknown",
			"tags":        []string{"philosophy", "being"},
			"title":       "On Appreciation",
		}), note.Attributes)
		assert.Equal(t, core.AttributeSet(map[string]any{
			"source": "Unknown",
			"tags":   []string{"being"},
		}), note.NoteAttributes)
		assert.Equal(t, core.TagSet{"being"}, note.NoteTags)
		assert.Equal(t, core.TagSet{"philosophy", "being"}, note.Attributes.Tags())
	})

	t.Run("Slug", func(t *testing.T) {
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)
		tr.WriteFile("dira/index.md", `
# Index A

## Note: Note in index A

This is a note in index A.
`)
		tr.WriteFile("dira/a.md", `
# File A

## Note: First note in file A

This is a note in file A.

## Note: Second note in file A

‛@slug: note-a‛

This is a note in file A.

`)
		tr.WriteFile("dirb/index.md", `
---
slug: b
---
# Index B

## Note: Note in Index B

This is a note in index B.`)
		tr.WriteFile("dirb/b.md", `
---
slug: b
---
# File B

## Note: Note in file B

This is a note in file B.`)

		mdIndexA := markdown.MustParseFile(filepath.Join(tr.Root, "dira/index.md"))
		mdA := markdown.MustParseFile(filepath.Join(tr.Root, "dira/a.md"))
		mdIndexB := markdown.MustParseFile(filepath.Join(tr.Root, "dirb/index.md"))
		mdB := markdown.MustParseFile(filepath.Join(tr.Root, "dirb/b.md"))

		indexA, err := core.ParseFile(mdIndexA, nil)
		require.NoError(t, err)
		indexB, err := core.ParseFile(mdIndexB, nil)
		require.NoError(t, err)
		fileA, err := core.ParseFile(mdA, mdIndexA)
		require.NoError(t, err)
		fileB, err := core.ParseFile(mdB, mdIndexB)
		require.NoError(t, err)

		require.Len(t, indexA.Notes, 1)
		require.Len(t, fileA.Notes, 2)
		require.Len(t, indexB.Notes, 1)
		require.Len(t, fileB.Notes, 1)

		// Check file slugs
		assert.Equal(t, "dira", indexA.Slug)
		assert.Equal(t, "dira-a", fileA.Slug)
		assert.Equal(t, "b", indexB.Slug)
		assert.Equal(t, "b", fileB.Slug)

		// Check note slugs
		assert.Equal(t, "dira-note-note-in-index-a", indexA.Notes[0].Slug)
		assert.Equal(t, "dira-a-note-first-note-in-file-a", fileA.Notes[0].Slug)
		assert.Equal(t, "note-a", fileA.Notes[1].Slug)
		assert.Equal(t, "b-note-note-in-index-b", indexB.Notes[0].Slug)
		assert.Equal(t, "b-note-note-in-file-b", fileB.Notes[0].Slug)
	})

	t.Run("LongTitle", func(t *testing.T) {
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)
		tr.WriteFile("a.md", `
# File A

## Note: Short title

This is a note.

### Flashcard: Quiz time

Titles are concatenated with [...].

---

Titles are concatenated with **the parent note**.

## Note: Title with a long name

This is a note.

### Flashcard: Title with a long name

Except when [...].

---

Except when **identical to the parent note**.
`)

		md := markdown.MustParseFile(filepath.Join(tr.Root, "a.md"))
		file, err := core.ParseFile(md, nil)
		require.NoError(t, err)

		notes := file.Notes
		require.Len(t, notes, 4)

		note := notes[0]
		assert.Equal(t, "Note: Short title", note.Title.String())
		assert.Equal(t, "Short title", note.ShortTitle.String())
		assert.Equal(t, "File A / Short title", note.LongTitle.String())

		note = notes[1]
		assert.Equal(t, "Flashcard: Quiz time", notes[1].Title.String())
		assert.Equal(t, "Quiz time", note.ShortTitle.String())
		assert.Equal(t, "File A / Short title / Quiz time", note.LongTitle.String())

		note = notes[2]
		assert.Equal(t, "Note: Title with a long name", note.Title.String())
		assert.Equal(t, "Title with a long name", note.ShortTitle.String())
		assert.Equal(t, "File A / Title with a long name", note.LongTitle.String())

		note = notes[3]
		assert.Equal(t, "Flashcard: Title with a long name", note.Title.String())
		assert.Equal(t, "Title with a long name", note.ShortTitle.String())
		assert.Equal(t, "File A / Title with a long name", note.LongTitle.String())

		// Let's try with a more subtle example where titles have a common prefix
		tr.WriteFile("b.md", `
# Go

## Note: Golang

This is a note.

### Note: Goroutines

This is a sub-note
`)

		md = markdown.MustParseFile(filepath.Join(tr.Root, "b.md"))
		file, err = core.ParseFile(md, nil)
		require.NoError(t, err)

		notes = file.Notes
		require.Len(t, notes, 2)

		note = notes[0]
		assert.Equal(t, "Note: Golang", note.Title.String())
		assert.Equal(t, "Golang", note.ShortTitle.String())
		assert.Equal(t, "Go / Golang", note.LongTitle.String())

		note = notes[1]
		assert.Equal(t, "Note: Goroutines", note.Title.String())
		assert.Equal(t, "Goroutines", note.ShortTitle.String())
		assert.Equal(t, "Go / Golang / Goroutines", note.LongTitle.String())
	})

	t.Run("Markdown in Markdown", func(t *testing.T) {
		// Markdown document can includes code blocks with lines starting with # characters.
		// These lines must not be parsed as heading (and thus as notes).
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)
		tr.WriteFile("md.md", `
# File

## Note: Markdown Example 1

‛‛‛md
## Note: A Markdown Heading

This note is not a note but a code block inside a note.
‛‛‛

## Note: Markdown Example 2

Another note without a code block.
`)

		md := markdown.MustParseFile(filepath.Join(tr.Root, "md.md"))

		// Check Markdown.File correctly interprets the headings
		sections, err := md.GetSections()
		require.NoError(t, err)
		require.Len(t, sections, 3)
		assert.Equal(t, "File", sections[0].HeadingText.String())
		assert.Equal(t, "Note: Markdown Example 1", sections[1].HeadingText.String())
		assert.Equal(t, "## Note: Markdown Example 1\n\n```md\n## Note: A Markdown Heading\n\nThis note is not a note but a code block inside a note.\n```", sections[1].ContentText.String())
		assert.Equal(t, "Note: Markdown Example 2", sections[2].HeadingText.String())
		assert.Equal(t, "## Note: Markdown Example 2\n\nAnother note without a code block.", sections[2].ContentText.String())

		// Parse the file
		file, err := core.ParseFile(md, nil)
		require.NoError(t, err)
		// Check that two notes have been found and ensure the first one contains the code block
		require.Len(t, file.Notes, 2)
		firstNote := file.Notes[0]
		require.Equal(t, "```md\n## Note: A Markdown Heading\n\nThis note is not a note but a code block inside a note.\n```", firstNote.Body.String())
	})

	t.Run("Journal", func(t *testing.T) {
		// Journal note titles are parsed to extract the date.
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)

		// Assert date preprocessor is configured
		require.Contains(t, core.CurrentConfigFile().MustGetType("Journal").Processors, "date-extractor")

		tr.WriteFile("2024-12-05.md", `
# Journal: 2024-12-05

## Work

* Completed some work.
`)
		tr.WriteFile("2024-12-06.md", `
# Journal: 2024-12-05

‛@date: 2024-12-06‛

## Work

* Completed some work.
`)

		md1 := markdown.MustParseFile(filepath.Join(tr.Root, "2024-12-05.md"))
		file1, err := core.ParseFile(md1, nil)
		require.NoError(t, err)

		md2 := markdown.MustParseFile(filepath.Join(tr.Root, "2024-12-06.md"))
		file2, err := core.ParseFile(md2, nil)
		require.NoError(t, err)

		require.Len(t, file1.Notes, 1)
		require.Len(t, file2.Notes, 1)

		// An attribute "date" must have been added to the note in the first file
		note := file1.Notes[0]
		require.Equal(t, "Journal", note.Type)
		require.Contains(t, note.Attributes, "date")
		assert.Equal(t, "2024-12-05", note.Attributes["date"])

		// The attribute "date" must have been preserved on the note in the second file
		note = file2.Notes[0]
		require.Equal(t, "Journal", note.Type)
		require.Contains(t, note.Attributes, "date")
		assert.Equal(t, "2024-12-06", note.Attributes["date"]) // Do not override existing attributes
	})

	t.Run("Medias Rewriting", func(t *testing.T) {
		// Test attributes and tags defined in Front Matter and in notes
		// are correctly parsed and merged.
		testutil.FreezeNow(t)

		tr := core.NewTestRepository(t)
		tr.WriteFile("medias/mona-lisa.png", "This is the worth reproduction.")
		tr.WriteFile("paintings.md", `
# Paintings

## Artwork: Mona Lisa

‛@painter: Leonardo da Vinci‛ ‛@year: ~1503‛ ‛@source: Louvre Museum‛

‛#masterpiece‛

![Mona Lisa](medias/mona-lisa.png)

> The painting was stolen in 1911 and recovered in 1913.
`)
		md := markdown.MustParseFile(filepath.Join(tr.Root, "paintings.md"))
		index, err := core.ParseFile(md, nil)
		require.NoError(t, err)

		require.Len(t, index.Notes, 1)

		// Check attributes and notes
		note := index.Notes[0]
		assert.Contains(t, note.Content.String(), "![Mona Lisa](medias/mona-lisa.png")
		assert.Contains(t, note.Body.String(), "<media relative-path=\"medias/mona-lisa.png\" alt=\"Mona Lisa\" />")
		// Only the body is post-processed. Therefore, medias are replaced by <media> tags only inside it.
	})

	t.Run("Title Attributes", func(t *testing.T) {
		// Test the new functionality for extracting tags and attributes from titles
		tr := core.NewTestRepository(t)
		tr.WriteFile("test.md", `
# My Notes ‛@project: topsecret‛ ‛#low-motivation‛ ❗️

## Note: Important Topic ‛#critical‛ ‛@priority: high‛ ★★★

This is a critical note with high priority.

## Quote: Famous Quote ‛#inspirational‛

‛#life-changing‛ ‛@name: Someone Famous‛

> This is an inspirational quote.
`)

		// Parse the file
		parsedFile := tr.ParseFile("test.md")

		// Check tags/attributes were extracted from the file title
		require.True(t, parsedFile.FileAttributes.Includes("project"))
		assert.Equal(t, core.AttributeSet(map[string]any{
			"priority": "high",
			"project":  "topsecret",
			"tags":     []string{"low-motivation"},
		}), parsedFile.FileAttributes)
		assert.True(t, parsedFile.FileAttributes.Tags().Includes("low-motivation"))

		// Check tags/attributes were stripped from file titles
		assert.Equal(t, "My Notes", parsedFile.Title.String())
		assert.Equal(t, "My Notes", parsedFile.ShortTitle.String())

		require.Len(t, parsedFile.Notes, 2)
		parsedNote1 := parsedFile.Notes[0]
		parsedNote2 := parsedFile.Notes[1]

		// Check tags/attributes were extracted from the first note title
		assert.ElementsMatch(t, []string{"priority", "project", "rating", "tags", "title"}, parsedNote1.Attributes.Keys())
		assert.ElementsMatch(t, []string{"name", "priority", "project", "tags", "title"}, parsedNote2.Attributes.Keys())

		// Check tags/attributes were stripped from note titles
		assert.Equal(t, "Note: Important Topic ★★★", parsedNote1.Title.String())
		assert.Equal(t, "Important Topic ★★★", parsedNote1.ShortTitle.String())
		assert.Equal(t, "Quote: Famous Quote", parsedNote2.Title.String())
		assert.Equal(t, "Famous Quote", parsedNote2.ShortTitle.String())
	})

}

func TestDetermineFileSlug(t *testing.T) {
	tests := []struct {
		path string // input
		slug string // output
	}{
		{
			path: "go/syntax.md",
			slug: "go-syntax",
		},
		{
			path: "go/index.md",
			slug: "go",
		},
		{
			path: "go/go/syntax.md",
			slug: "go-syntax",
		},
		{
			path: "go/go.md",
			slug: "go",
		},
		// File at root does not include the dir prefix
		{
			path: "go.md",
			slug: "go",
		},
	}
	for _, tt := range tests {
		actual := core.DetermineFileSlug(tt.path)
		assert.Equal(t, tt.slug, actual)
	}
}

func TestMarkdownTransformers(t *testing.T) {

	t.Run("StripSubNotesTransformer", func(t *testing.T) {
		core.NewTestRepository(t)

		tests := []struct {
			name     string
			input    markdown.Document // input
			expected markdown.Document // output
		}{

			{
				name: "No sub-notes",
				input: `
## Note: A note

A simple note
`,
				// Nothing must be stripped
				expected: `
## Note: A note

A simple note
`,
			},

			{
				name: "Untyped sub-notes",
				input: `
## Note: A note

A simple note

### Subheading

Some more text
`,
				// Sub-sections must be present as they are not typed notes
				expected: `
## Note: A note

A simple note

### Subheading

Some more text
`,
			},

			{
				name: "With sub-notes",
				input: `
## Note: A note

A simple note

### Note: A sub note

Some more text
`,
				// Sub-notes must be trimmed
				expected: `
## Note: A note

A simple note
`,
			},

			{
				name: "With code blocks containing an heading",
				input: markdown.Document(text.UnescapeTestContent(`
## Note: A note

A simple note

‛‛‛md
### Note: Not a sub note

This is a code block.
‛‛‛
`)),
				// The code block must be ignored
				expected: markdown.Document(text.UnescapeTestContent(`
## Note: A note

A simple note

‛‛‛md
### Note: Not a sub note

This is a code block.
‛‛‛
`)),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := tt.input.Transform(core.StripSubNotesTransformer)
				require.NoError(t, err)
				assert.Equal(t, tt.expected.TrimSpace(), actual.TrimSpace())
			})
		}

	})

}

func TestFormatLongTitle(t *testing.T) {
	tests := []struct {
		name      string
		titles    []markdown.Document // input
		longTitle markdown.Document   // output
	}{
		{
			name:      "Basic",
			titles:    []markdown.Document{"Go", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Empty titles",
			titles:    []markdown.Document{"", "History"},
			longTitle: "History",
		},
		{
			name:      "Duplicate titles",
			titles:    []markdown.Document{"Go", "History", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Common prefix",
			titles:    []markdown.Document{"Go", "Go History"},
			longTitle: "Go History",
		},
		{
			name:      "Not common prefix",
			titles:    []markdown.Document{"Go", "Goroutines"},
			longTitle: "Go / Goroutines",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := core.FormatLongTitle(tt.longTitle)
			assert.Equal(t, tt.longTitle, actual)
		})
	}
}

func TestCustomNoteTypes(t *testing.T) {

	t.Run("New Type", func(t *testing.T) {
		tr := core.NewTestRepository(t)

		// Edit config to declare a new custom type
		tr.WriteFile(".nt/config.jsonnet", `
local nt = import 'nt.libsonnet';
{
    Attributes: {
		// New attributes for the BookReview type
		draft: {
            name: "draft",
            type: "bool",
            inherit: false,
        },
		isbn: {
			name: "isbn",
			type: "string",
            format: "isbn",
		},
        review_rating: {
            name: "review_rating",
            type: "integer",
            min: 0,
            max: 20,
            inherit: true,
        },
        review_stars: {
            name: "review_stars",
            type: "integer",
            min: 0,
            max: 5,
            inherit: true,
        },
        read_date: {
            name: "read_date",
            type: "date",
            format: "yyyy-mm-dd",
            inherit: true,
            memory: true,
        },
	},
	Types: nt.DefaultTypes + {

		// A new type similar to existing ones
		BookReview: nt.DefaultTypes.Note + {
			name: "BookReview",
			attributes: [
				{
					name: "isbn",
					required: true,
				},
			],
		},

		// A new type with a custom pattern
		Idea: nt.DefaultTypes.Note + {
			name: "Idea",
			pattern: "^💡 (.*)$",
		},
	}
}
		`)
		core.CurrentConfig().Reload()

		tr.WriteFile("the-midnight-library.md", `
---
title: The Midnight Library
isbn: 978-0525559474
---

# The Midnight Library

## BookReview: The Midnight Library

‛@read_date: 2025-04-01‛
‛@review_rating: 20‛ ‛@review_stars: 5‛
‛@draft: false‛

Definitely a book to read while your are still young to act before growing your regrets.

## 💡 Read again

Reread the book in 10 years to see how my perspective has changed.
`)
		md := markdown.MustParseFile(filepath.Join(tr.Root, "the-midnight-library.md"))
		file, err := core.ParseFile(md, nil)
		require.NoError(t, err)
		require.Len(t, file.Notes, 2)

		note, ok := file.FindNoteByTitle("BookReview: The Midnight Library")
		require.True(t, ok)
		assert.Equal(t, "BookReview", note.Type)
		assert.Equal(t, "BookReview: The Midnight Library", note.Title.String())
		assert.Equal(t, "The Midnight Library", note.ShortTitle.String())
		assert.Equal(t, core.AttributeSet(map[string]any{
			"isbn":          "978-0525559474",
			"review_rating": int64(20),
			"review_stars":  int64(5),
			"read_date":     "2025-04-01",
			"draft":         false,
			"title":         "The Midnight Library",
		}), note.Attributes)

		note, ok = file.FindNoteByTitle("💡 Read again")
		require.True(t, ok)
		assert.Equal(t, "Idea", note.Type)
		assert.Equal(t, "💡 Read again", note.Title.String())
		assert.Equal(t, "Read again", note.ShortTitle.String())
	})
}

func TestReplaceMedias(t *testing.T) {
	tests := []struct {
		name     string
		medias   []*core.ParsedMedia
		input    string
		expected string
	}{
		{
			name: "Replace single media",
			medias: []*core.ParsedMedia{
				{
					RawPath:      "images/pic.png",
					RelativePath: "images/pic.png",
					Dangling:     false,
				},
			},
			input:    `Here is an image: ![](images/pic.png)`,
			expected: `Here is an image: <media relative-path="images/pic.png" />`,
		},
		{
			name: "Replace multiple medias",
			medias: []*core.ParsedMedia{
				{
					RawPath:      "images/pic1.png",
					RelativePath: "images/pic1.png",
					Dangling:     false,
				},
				{
					RawPath:      "images/pic2.png",
					RelativePath: "images/pic2.png",
					Dangling:     false,
				},
			},
			input:    `![img1](images/pic1.png) and ![img2](images/pic2.png)`,
			expected: `<media relative-path="images/pic1.png" alt="img1" /> and <media relative-path="images/pic2.png" alt="img2" />`,
		},
		{
			name: "Do not replace dangling media",
			medias: []*core.ParsedMedia{
				{
					RawPath:      "images/pic.png",
					RelativePath: "images/pic.png",
					Dangling:     true,
				},
			},
			input:    `![](images/pic.png)`,
			expected: `![](images/pic.png)`,
		},
		{
			name:     "Do not replace external media",
			medias:   nil,
			input:    `![alt](https://example.com/pic.png)`,
			expected: `![alt](https://example.com/pic.png)`,
		},
		{
			name: "Replace media with relative path containing special characters",
			medias: []*core.ParsedMedia{
				{
					RawPath:      "images/my (special).png",
					RelativePath: "images/my (special).png",
					Dangling:     false,
				},
			},
			input:    `![](images/my (special).png)`,
			expected: `<media relative-path="images/my (special).png" />`,
		},
		{
			name: "Preserve alt and title",
			medias: []*core.ParsedMedia{
				{
					RawPath:      "images/pic.png",
					RelativePath: "images/pic.png",
					Dangling:     false,
				},
			},
			input:    `Here is an image: ![Alt](images/pic.png "A title")`,
			expected: `Here is an image: <media relative-path="images/pic.png" alt="Alt" title="A title" />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := markdown.Document(tt.input)
			transformer := core.ReplaceMedias(tt.medias)
			result, err := doc.Transform(transformer)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}
