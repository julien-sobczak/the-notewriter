package core_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	testcases := []struct {
		name   string
		golden string
		test   func(*testing.T, *core.ParsedFileNew)
	}{
		{
			name:   "Basic",
			golden: "basic",
			test: func(t *testing.T, file *core.ParsedFileNew) {
				require.NotNil(t, file)

				// We check everything in this basic file
				// so that following tests can focus on specificities

				// Check file
				assert.Equal(t, "basic-notetaking", file.Slug)
				assert.Equal(t, "Basic Note-Taking", file.Title)
				assert.Equal(t, "Basic Note-Taking", file.ShortTitle)

				// File attributes extracted from the Front Matter
				assert.Equal(t, map[string]any{
					"title":  "Basic Note-Taking",
					"rating": 5,
					"slug":   "basic-notetaking",
					"tags":   []any{"thinking"},
				}, file.FileAttributes)

				// Check subobjects
				assert.Len(t, file.Medias, 1)
				assert.Len(t, file.Notes, 4)

				// Check media "da-vinci-notebook.png"
				mediaDaVinci, ok := file.FindMediaByFilename("da-vinci-notebook.png")
				require.True(t, ok)
				expectedDaVinci := &core.ParsedMediaNew{
					RawPath:      "medias/da-vinci-notebook.png",
					AbsolutePath: filepath.Join(filepath.Dir(file.Markdown.AbsolutePath), "medias/da-vinci-notebook.png"),
					Extension:    ".png",
					MediaKind:    core.KindPicture,
					Line:         41,
					// File existence must also be checked
					Dangling: false,
				}
				require.EqualExportedValues(t, *expectedDaVinci, *mediaDaVinci)
				assert.WithinDuration(t, time.Now(), mediaDaVinci.MTime(), 1*time.Minute) // test cases are copied in a temp directory
				assert.Greater(t, mediaDaVinci.Size(), int64(0))

				// Check "Note: A Note"
				noteNote, ok := file.FindNoteByShortTitle("A Note")
				require.True(t, ok)
				assert.Equal(t, 2, noteNote.Level)
				assert.Equal(t, core.KindNote, noteNote.Kind)
				assert.Equal(t, "basic-notetaking-note-a-note", noteNote.Slug)
				assert.Equal(t, "Note: A Note", noteNote.Title)
				assert.Equal(t, "A Note", noteNote.ShortTitle)
				assert.Equal(t, 11, noteNote.Line)
				assert.Equal(t, "## Note: A Note\n\nNotes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Content)
				assert.Equal(t, "Notes has many uses:\n\n* Journaling\n* To-Do list\n* Drawing\n* Diary\n* Flashcard\n* Reminder", noteNote.Body)
				assert.Empty(t, nil, noteNote.NoteAttributes)
				assert.Empty(t, nil, noteNote.NoteTags)
				// No subobjects
				assert.Nil(t, noteNote.Flashcard)
				assert.Len(t, noteNote.Links, 0)
				assert.Len(t, noteNote.Reminders, 0)

				// Check "Quote: Tim Ferris on Note-Taking"
				noteTimFerris, ok := file.FindNoteByShortTitle("Tim Ferris on Note-Taking")
				require.True(t, ok)
				require.Equal(t, map[string]any{
					"author": "Tim Ferris",
				}, noteTimFerris.NoteAttributes)

				// Check "Flashcard: Commonplace Book"
				noteCommomplace, ok := file.FindNoteByShortTitle("Commonplace Book")
				require.True(t, ok)
				require.NotNil(t, noteCommomplace.Flashcard)
				flashcardCommonplace := noteCommomplace.Flashcard
				assert.Equal(t, "Commonplace Book", flashcardCommonplace.ShortTitle)
				assert.Equal(t, "(Thinking) What are **commonplace books**?", flashcardCommonplace.Front)
				assert.Equal(t, "A tool to compile knowledge, usually by writing information into books.", flashcardCommonplace.Back)

				// Check "Reference: Leonardo da Vinci's Notebooks"
				noteDaVinci, ok := file.FindNoteByShortTitle("Leonardo da Vinci's Notebooks")
				require.True(t, ok)
				require.Equal(t, map[string]any{
					"author": "Leonardo da Vinci",
					"year":   "~1510",
				}, noteDaVinci.NoteAttributes)
			},
		},

		{
			name:   "Characters Replacement",
			golden: "characters-replacement",
			test: func(t *testing.T, file *core.ParsedFileNew) {
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
			test: func(t *testing.T, file *core.ParsedFileNew) {
				require.NotNil(t, file)

				noteA, ok := file.FindNoteByShortTitle("A")
				require.True(t, ok)
				noteB, ok := file.FindNoteByShortTitle("B")
				require.True(t, ok)

				assert.Equal(t, `Some text inside the note.`, noteA.Body)
				assert.Equal(t, `Text`, noteB.Body)
			},
		},

		{
			name:   "Ignore",
			golden: "ignore",
			test: func(t *testing.T, file *core.ParsedFileNew) {
				require.Nil(t, file)
				// Nothing more to check
			},
		},

		{
			name:   "Minimal",
			golden: "minimal",
			test: func(t *testing.T, file *core.ParsedFileNew) {
				require.NotNil(t, file)

				// TODO complete
			},
		},

		{
			name:   "Generator",
			golden: "generator",
			test: func(t *testing.T, file *core.ParsedFileNew) {
				require.NotNil(t, file)

			},
		},

		// Add more test cases here to enrich Markdown support
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			dirname := testutil.SetUpFromGoldenDirNamed(t, "TestParser")
			md, err := markdown.ParseFile(filepath.Join(dirname, testcase.golden+".md"))
			require.NoError(t, err)
			file, err := ParseFileFromMarkdownFile(md)
			require.NoError(t, err)
			testcase.test(t, file)
		})
	}
}
