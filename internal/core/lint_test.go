package core

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoDuplicateNoteTitle(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-duplicate-note-title.md"))
	require.NoError(t, err)

	violations, err := NoDuplicateNoteTitle(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "no-duplicate-note-title.md",
			Message:      `duplicated note with title "Long title must be unique inside a file"`,
			Line:         15,
		},
	}, violations)
}

func TestMinLinesBetweenNotes(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "min-lines-between-notes.md"))
	require.NoError(t, err)

	violations, err := MinLinesBetweenNotes(file, []string{"2"})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "min-lines-between-notes.md",
			Message:      `missing blank lines before note "Note: Two"`,
			Line:         7,
		},
		{
			RelativePath: "min-lines-between-notes.md",
			Message:      `missing blank lines before note "Note: Four"`,
			Line:         15,
		},
	}, violations)
}

func TestNoteTitleMatch(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "note-title-match.md"))
	require.NoError(t, err)

	violations, err := NoteTitleMatch(file, []string{`^(Note|Reference):\s\S.*$`})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "note-title-match.md",
			Message:      `note title "Reference : Example" does not match regex "^(Note|Reference):\\s\\S.*$"`,
			Line:         7,
		},
		{
			RelativePath: "note-title-match.md",
			Message:      `note title "reference: Example" does not match regex "^(Note|Reference):\\s\\S.*$"`,
			Line:         11,
		},
	}, violations)
}

func TestNoFreeNote(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-free-note.md"))
	require.NoError(t, err)

	violations, err := NoFreeNote(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "no-free-note.md",
			Message:      `XXX`,
			Line:         1,
		},
	}, violations)
}

func TestNoDanglingMedia(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-dangling-media.md"))
	require.NoError(t, err)

	violations, err := NoDanglingMedia(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "no-dangling-media.md",
			Message:      `dangling media pic.jpeg detected in no-dangling-media.md`,
			Line:         3,
		},
		{
			RelativePath: "no-dangling-media.md",
			Message:      `dangling media no-dangling-media/pic.jpg detected in no-dangling-media.md`,
			Line:         5,
		},
	}, violations)
}

func TestNoDeadWikilink(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-dead-wikilink.md"))
	require.NoError(t, err)

	violations, err := NoDeadWikilink(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "no-dead-wikilink.md",
			Message:      `XXX`,
			Line:         1,
		},
	}, violations)
}

func TestNoExtensionWikilink(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-extension-wikilink.md"))
	require.NoError(t, err)

	violations, err := NoExtensionWikilink(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			RelativePath: "no-extension-wikilink.md",
			Message:      `XXX`,
			Line:         1,
		},
	}, violations)
}
