package core_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileWithTestdata(t *testing.T) {
	core.FreezeNow(t)

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
					"rating": 5,
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
				assert.Equal(t, core.KindNote, noteNote.Kind)
				assert.Equal(t, "basic-notetaking-note-a-note", noteNote.Slug)
				assert.Equal(t, markdown.Document("Note: A Note"), noteNote.Title)
				assert.Equal(t, markdown.Document("A Note"), noteNote.ShortTitle)
				assert.Equal(t, 11, noteNote.Line)
				assert.Equal(t, "## Note: A Note\n\nNotes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Content.String())
				assert.Equal(t, "Notes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Body.String())
				assert.Empty(t, nil, noteNote.Attributes)
				// No subobjects
				assert.Nil(t, noteNote.Flashcard)
				assert.Len(t, noteNote.GoLinks, 0)
				assert.Len(t, noteNote.Reminders, 0)

				// Check "Quote: Tim Ferris on Note-Taking"
				noteTimFerris, ok := file.FindNoteByShortTitle("Tim Ferris on Note-Taking")
				require.True(t, ok)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Tim Ferris",
				}), noteTimFerris.NoteAttributes)
				require.Equal(t, core.AttributeSet(map[string]any{
					"title":  "Basic Note-Taking",
					"author": "Tim Ferris",
					"rating": 5,
					"slug":   "basic-notetaking",
					"tags":   []string{"thinking"},
				}), noteTimFerris.Attributes)

				// Check "Flashcard: Commonplace Book"
				noteCommomplace, ok := file.FindNoteByShortTitle("Commonplace Book")
				require.True(t, ok)
				require.NotNil(t, noteCommomplace.Flashcard)
				flashcardCommonplace := noteCommomplace.Flashcard
				assert.Equal(t, "Commonplace Book", flashcardCommonplace.ShortTitle.String())
				assert.Equal(t, "(Thinking) What are **commonplace books**?", flashcardCommonplace.Front.String())
				assert.Equal(t, "A tool to compile knowledge, usually by writing information into books.", flashcardCommonplace.Back.String())

				// Check "Reference: Leonardo da Vinci's Notebooks"
				noteDaVinci, ok := file.FindNoteByShortTitle("Leonardo da Vinci's Notebooks")
				require.True(t, ok)
				require.Equal(t, core.AttributeSet(map[string]any{
					"author": "Leonardo da Vinci",
					"year":   "~1510",
				}), noteDaVinci.NoteAttributes)
				require.Equal(t, core.AttributeSet(map[string]any{
					"title":  "Basic Note-Taking",
					"author": "Leonardo da Vinci",
					"rating": 5,
					"slug":   "basic-notetaking",
					"tags":   []string{"thinking"},
					"year":   "~1510",
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
				assert.Contains(t, note.Body.String(), "#### Motivations")
				assert.Contains(t, note.Body.String(), "#### Introduction")
				assert.Contains(t, note.Body.String(), "#### Demo")
				// BUT
				note, ok = file.FindNoteByTitle("Reference: First Notebooks")
				require.True(t, ok)
				assert.NotContains(t, note.Body.String(), "Flashcard: First Notebooks")

				// TODO complete
			},
		},

		{
			name:   "Generator",
			golden: "generator",
			test: func(t *testing.T, file *core.ParsedFile) {
				require.NotNil(t, file)

				// TODO complete
			},
		},

		// Add more test cases here to enrich Markdown support
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			root := core.SetUpRepositoryFromGoldenDirNamed(t, "TestParser")
			md, err := markdown.ParseFile(filepath.Join(root, testcase.golden+".md"))
			require.NoError(t, err)
			file, err := core.ParseFile(md, nil)
			require.NoError(t, err)
			testcase.test(t, file)
		})
	}
}

func TestParseFileWithTempdir(t *testing.T) {

	t.Run("Slug", func(t *testing.T) {
		core.FreezeNow(t)

		root := core.SetUpRepositoryFromTempDir(t)
		core.MustWriteFile(t, "dira/index.md", `
# Index A

## Note: Note in index A

This is a note in index A.
`)
		core.MustWriteFile(t, "dira/a.md", `
# File A

## Note: First note in file A

This is a note in file A.

## Note: Second note in file A

‛@slug: note-a‛

This is a note in file A.

`)
		core.MustWriteFile(t, "dirb/index.md", `
---
slug: b
---
# Index B

## Note: Note in Index B

This is a note in index B.`)
		core.MustWriteFile(t, "dirb/b.md", `
---
slug: b
---
# File B

## Note: Note in file B

This is a note in file B.`)

		mdIndexA := markdown.MustParseFile(filepath.Join(root, "dira/index.md"))
		mdA := markdown.MustParseFile(filepath.Join(root, "dira/a.md"))
		mdIndexB := markdown.MustParseFile(filepath.Join(root, "dirb/index.md"))
		mdB := markdown.MustParseFile(filepath.Join(root, "dirb/b.md"))

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
		core.FreezeNow(t)

		root := core.SetUpRepositoryFromTempDir(t)
		core.MustWriteFile(t, "a.md", `
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

		md := markdown.MustParseFile(filepath.Join(root, "a.md"))
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
		core.MustWriteFile(t, "b.md", `
# Go

## Note: Golang

This is a note.

### Note: Goroutines

This is a sub-note
`)

		md = markdown.MustParseFile(filepath.Join(root, "b.md"))
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
