package e2e_test

import (
	"testing"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnrichingWithMetadata verifies that inline tags and attributes defined in note titles
// and note bodies are correctly parsed, merged with frontmatter attributes, and stored.
func TestEnrichingWithMetadata(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("notes.md", `
---
tags: [programming]
---

# Programming Notes

## Note: Inline Tags

‛#go‛ ‛#beginner‛

Inline tags are defined at the start of the note body.

## Note: Inline Attributes ‛@author: John Doe‛

Inline attributes can be defined directly in the note heading.

## Note: Combined

‛#python‛ ‛@year: 1991‛

Both tags and attributes can appear together in the note body.
`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	// Verify that 3 notes were indexed
	require.Equal(t, 3, tr.CountNotes())

	// Note with inline tags in body: tags from frontmatter + note-level tags must be merged
	noteTags, err := CurrentRepository().FindNoteByPathAndTitle("notes.md", "Note: Inline Tags")
	require.NoError(t, err)
	require.NotNil(t, noteTags)
	assert.Equal(t, TagSet{"programming", "go", "beginner"}, noteTags.Tags)

	// Note with inline attribute in title: attribute must be accessible
	noteAttr, err := CurrentRepository().FindNoteByPathAndTitle("notes.md", "Note: Inline Attributes")
	require.NoError(t, err)
	require.NotNil(t, noteAttr)
	assert.Equal(t, "John Doe", noteAttr.Attributes["author"])
	// Frontmatter tags must still be present
	assert.Equal(t, TagSet{"programming"}, noteAttr.Tags)

	// Note with both tags and attributes in body
	noteCombined, err := CurrentRepository().FindNoteByPathAndTitle("notes.md", "Note: Combined")
	require.NoError(t, err)
	require.NotNil(t, noteCombined)
	assert.Equal(t, TagSet{"programming", "python"}, noteCombined.Tags)
	assert.Equal(t, "1991", noteCombined.Attributes["year"])
}

// TestWritingSyntaxSugar verifies that AsciiDoc character sequences are replaced
// in note bodies but preserved inside code blocks and inline code spans.
func TestWritingSyntaxSugar(t *testing.T) {
	NewTestRepository(t,
		WithFreezeNow(),
		WithFile("chars.md", `
# Character Replacements

## Note: Asciidoc Replacements

Supported replacements:

* Copyright: (C)
* Registered: (R)
* Trademark: (TM)
* Em dash: --
* Ellipsis: ...
* Right arrow: ->
* Double right arrow: =>
* Left arrow: <-
* Double left arrow: <=

Except when in inline code like ‛i--‛ or in a code block:

‛‛‛c
i--
‛‛‛
`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	note, err := CurrentRepository().FindNoteByPathAndTitle("chars.md", "Note: Asciidoc Replacements")
	require.NoError(t, err)
	require.NotNil(t, note)

	body := note.Body.String()

	// AsciiDoc character sequences must be replaced in the note body
	assert.Contains(t, body, "©")
	assert.Contains(t, body, "®")
	assert.Contains(t, body, "™")
	assert.Contains(t, body, "—")
	assert.Contains(t, body, "…")
	assert.Contains(t, body, "→")
	assert.Contains(t, body, "⇒")
	assert.Contains(t, body, "←")
	assert.Contains(t, body, "⇐")

	// Original sequences must no longer appear outside code spans/blocks
	assert.NotContains(t, body, "(C)")
	assert.NotContains(t, body, "(R)")
	assert.NotContains(t, body, "(TM)")

	// Inline code must be preserved as-is
	assert.Contains(t, body, "`i--`")

	// Content inside code blocks must NOT have replacements applied
	assert.Contains(t, body, "```c\ni--\n```")
}

// TestWritingUsingSnippets verifies that code blocks delimited by 3, 4, or 5
// backticks are correctly recognized, that AsciiDoc substitutions are not applied inside
// them, and that a block with more backticks can safely contain a fence with fewer backticks.
func TestWritingUsingSnippets(t *testing.T) {
	tr := NewTestRepository(t,
		WithFile("codeblocks.md", `
# Code Blocks

## Note: Three Backticks

A standard code block:

‛‛‛python
x = 1 -- 2  # (C) stays as-is inside code blocks
‛‛‛

## Note: Four Backticks

A four-backtick block can contain three-backtick fences:

‛‛‛‛markdown
Here is a code block inside:

‛‛‛python
if true:
    print("hello")
‛‛‛

And (C) stays as-is here too.
‛‛‛‛

## Note: Five Backticks

A five-backtick block can contain three- and four-backtick fences:

‛‛‛‛‛text
Four-backtick fence inside:
‛‛‛‛
nested
‛‛‛‛

Three-backtick fence inside:
‛‛‛
also nested
‛‛‛

AsciiDoc chars like --> and (TM) must not be replaced.
‛‛‛‛‛
`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	require.Equal(t, 3, tr.CountNotes())

	// Three-backtick code block
	noteThree, err := CurrentRepository().FindNoteByPathAndTitle("codeblocks.md", "Note: Three Backticks")
	require.NoError(t, err)
	require.NotNil(t, noteThree)
	bodyThree := noteThree.Body.String()
	// Code block delimiters and content are preserved
	assert.Contains(t, bodyThree, "```python")
	// AsciiDoc substitution must NOT have been applied inside the code block
	assert.Contains(t, bodyThree, "x = 1 -- 2")
	assert.Contains(t, bodyThree, "(C) stays as-is inside code blocks")

	// Four-backtick code block
	noteFour, err := CurrentRepository().FindNoteByPathAndTitle("codeblocks.md", "Note: Four Backticks")
	require.NoError(t, err)
	require.NotNil(t, noteFour)
	bodyFour := noteFour.Body.String()
	// The outer four-backtick fence is preserved
	assert.Contains(t, bodyFour, "````markdown")
	// The three-backtick fence nested inside is preserved verbatim
	assert.Contains(t, bodyFour, "```python")
	assert.Contains(t, bodyFour, `   print("hello")`) // Identation must be preserved
	// AsciiDoc substitution must NOT have been applied inside the outer code block
	assert.Contains(t, bodyFour, "(C) stays as-is here too.")

	// Five-backtick code block
	noteFive, err := CurrentRepository().FindNoteByPathAndTitle("codeblocks.md", "Note: Five Backticks")
	require.NoError(t, err)
	require.NotNil(t, noteFive)
	bodyFive := noteFive.Body.String()
	// The outer five-backtick fence is preserved
	assert.Contains(t, bodyFive, "`````text")
	// Both nested fences are preserved verbatim
	assert.Contains(t, bodyFive, "````")
	assert.Contains(t, bodyFive, "```")
	// AsciiDoc substitution must NOT have been applied inside the five-backtick block
	assert.Contains(t, bodyFive, "-->")
	assert.Contains(t, bodyFive, "(TM)")
}

// TestSavingImportantURLs verifies that goto links in different formats (simple, hierarchical, with title)
// are correctly extracted and stored in the database.
func TestSavingImportantURLs(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("resources.md", `
# Resources

## Note: Go Language

Useful links for the Go programming language:

* [Go Documentation](https://go.dev/doc/ "#go/go")
* [Go Playground](https://go.dev/play/ "#go/go/playground")
* [Go Blog](https://go.dev/blog/ "Blog #go/go/blog")

## Note: GitHub

Project links:

* [GitHub Repository](https://github.com/example/repo "#go/github-repo")
* [GitHub Issues](https://github.com/example/repo/issues "Issues Tracker #go/github-issues")
`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	// All five goto links must have been indexed
	require.Equal(t, 5, tr.CountGotos())

	// Simple goto: no title, simple name
	gotoSimple, err := CurrentRepository().FindGotoByName("go")
	require.NoError(t, err)
	require.NotNil(t, gotoSimple)
	assert.Equal(t, "Go Documentation", gotoSimple.Text.String())
	assert.Equal(t, "https://go.dev/doc/", string(gotoSimple.URL))
	assert.Empty(t, gotoSimple.Title)
	assert.Equal(t, "go", gotoSimple.Name)

	// Hierarchical goto: name with slash separator
	gotoPlay, err := CurrentRepository().FindGotoByName("go/playground")
	require.NoError(t, err)
	require.NotNil(t, gotoPlay)
	assert.Equal(t, "Go Playground", gotoPlay.Text.String())
	assert.Equal(t, "https://go.dev/play/", string(gotoPlay.URL))
	assert.Empty(t, gotoPlay.Title)

	// Goto with title: the part before "#go/" is the short title
	gotoBlog, err := CurrentRepository().FindGotoByName("go/blog")
	require.NoError(t, err)
	require.NotNil(t, gotoBlog)
	assert.Equal(t, "Go Blog", gotoBlog.Text.String())
	assert.Equal(t, "Blog", gotoBlog.Title)

	// Hierarchical goto with dash-separated name
	gotoRepo, err := CurrentRepository().FindGotoByName("github-repo")
	require.NoError(t, err)
	require.NotNil(t, gotoRepo)
	assert.Equal(t, "GitHub Repository", gotoRepo.Text.String())

	// Goto with title and dash-separated name
	gotoIssues, err := CurrentRepository().FindGotoByName("github-issues")
	require.NoError(t, err)
	require.NotNil(t, gotoIssues)
	assert.Equal(t, "GitHub Issues", gotoIssues.Text.String())
	assert.Equal(t, "Issues Tracker", gotoIssues.Title)
}

// TestWritingLongDoc verifies that a file tagged with "toc" produces an automatic table of
// contents note containing wikilinks to typed sections and titles for untyped parent
// sections, while excluding sections that have no typed child notes.
func TestWritingLongDoc(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("toc.md", `
---
tags: "toc"
---

# Document

## Introduction

This section has no typed child notes, so it should not appear in the TOC.

## Note: Core Concept

The core concept note.

### Note: Sub-concept

A sub-concept under the core concept.

### Flashcard: Sub-concept Definition

What is the sub-concept?

---

The sub-concept is a secondary idea.

## References

This untyped section has typed children, so it should appear in the TOC.

### Quote: Influential Quote

> The only way to do great work is to love what you do.

-- Steve Jobs

### Note: Reference Note

A reference note under the references section.

## Empty Section

This section has no typed children and should not appear in the TOC.
`))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	// There should be 6 notes: 1 TOC note + 5 content notes (Core Concept, Sub-concept,
	// Sub-concept Definition, Influential Quote, Reference Note)
	require.Equal(t, 6, tr.CountNotes())

	// Find the generated TOC note
	tocNote, err := CurrentRepository().FindNoteByPathAndTitle("toc.md", "Table of Content")
	require.NoError(t, err)
	require.NotNil(t, tocNote)

	tocBody := tocNote.Body.String()

	// Typed sections at the file level must appear as wikilinks
	assert.Contains(t, tocBody, "[[#Note: Core Concept]]")
	// Typed child sections must appear indented
	assert.Contains(t, tocBody, "[[#Note: Sub-concept]]")
	assert.Contains(t, tocBody, "[[#Flashcard: Sub-concept Definition]]")

	// Untyped parent sections with typed children appear as plain text (no wikilink)
	assert.Contains(t, tocBody, "References")
	assert.Contains(t, tocBody, "[[#Quote: Influential Quote]]")
	assert.Contains(t, tocBody, "[[#Note: Reference Note]]")

	// Sections without any typed children must NOT appear in the TOC
	assert.NotContains(t, tocBody, "Introduction", "Introduction section should not appear in TOC")
	assert.NotContains(t, tocBody, "Empty Section", "Empty Section should not appear in TOC")
}

func TestManagingList(t *testing.T) {
	/*
	 * This test migrates a note to a list and verifies that all items are correctly
	 * extracted with their attributes, then adds a new item to the list and verifies
	 * that it is correctly added.
	 */
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("readings.md", text.UnescapeTestContent(`
# My Readings

## Note: Read Books

* ‛@read_date: 2026-02-15‛ _Efficient Go_ ★★☆☆☆ ‛@author: Bartlomiej Plotka‛ ‛@isbn: 978-1098105716‛
  * Not as useful as _Systems Performance_ by Brendan Gregg.
* ‛@read_date: 2026-03-03‛: _The Product-Minded Engineer_ ★★★★★ ‛@author: Drew Hoskins‛
* ‛@read_date: 2026-03-10‛: _Working Effectively with Unit Tests_ ★★★☆☆ ‛@author: Jay Fields‛ ‛@isbn: 978-1503242708‛
  * If _XUnit Test Patterns_ gives you the full toolbox for mastering unit testing, _Working Effectively with Unit Tests_ provides an opiniated approach to use those tools in practice.
* ‛@read_date: 2026-03-16‛: _The Imagination Emporium_ ★★★☆☆ ‛@author: Duncan Wardle‛ ‛@isbn: 1637553617‛
* ‛@read_date: 2025-03-25‛: _On the Shortness of Life_ ★★★★☆ ‛@author: Seneca‛ ‛@isbn: 978-1985208728‛
  * It's always eye-opening to understand that people have always operated the same way and "modern" problems aren't so. A short read that's best appreciated if you are already familiar with Stoic philosophy.`)))

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// No items must have been extract when using the type "Note"
	note := tr.FindNoteByPathAndTitle("readings.md", "Note: Read Books")
	require.Empty(t, note.Items)

	// Change the type to "List" and verify that all items are correctly extracted with their attributes
	tr.ReplaceLine("readings.md", 4, "## Note: Read Books", "## List: Read Books")

	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// Reread the notes
	updatedNote := tr.FindNoteByPathAndTitle("readings.md", "List: Read Books")
	require.NotNil(t, updatedNote)
	require.Len(t, updatedNote.Items.Children, 5)
	oldNumberOfItems := len(updatedNote.Items.Children)

	// The old note must no longer exist
	oldNote, err := CurrentRepository().FindNoteByPathAndTitle("readings.md", "Note: Read Books")
	require.NoError(t, err) // But...
	require.Nil(t, oldNote)

	// Let's try to complete the list
	tr.AppendLines("readings.md", text.UnescapeTestContent(`
* ‛@read_date: 2025-12-20‛ _The Art of Spending Money_ ★★★★★ 👍 ‛@author: Morgan Housel‛ ‛@isbn: 9780593716632‛
  * One of my favorite books.
	`))

	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(true)
	require.NoError(t, err)

	// The new item must have been added to the list
	finalNote := tr.FindNoteByPathAndTitle("readings.md", "List: Read Books")
	require.NotNil(t, finalNote)
	require.Len(t, finalNote.Items.Children, oldNumberOfItems+1)
	assert.Equal(t, "_The Art of Spending Money_ ★★★★★ 👍", finalNote.Items.Children[oldNumberOfItems].Text.String())

	assert.Equal(t, 1, tr.CountNotes()) // We have only worked with a single note
}
