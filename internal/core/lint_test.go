package core

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchemaAttributes(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "check-attribute/check-attribute.md"))
	require.NoError(t, err)
	notes := ParseNotes(file.Body)
	require.Equal(t, "Quote: Steve Jobs on Life", notes[0].LongTitle)
	definitions := GetSchemaAttributes(file.RelativePath, notes[0].Kind)
	assert.Equal(t, []*ConfigLintSchemaAttribute{
		{
			Name:     "isbn",
			Type:     "string",
			Pattern:  "^([0-9-]{10}|[0-9]{3}-[0-9]{10})$",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "name",
			Type:     "string",
			Required: BoolPointer(true),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "references",
			Type:     "array",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "source",
			Type:     "string",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "tags",
			Type:     "array",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
	}, definitions)

	file, err = ParseFile(filepath.Join(root, "check-attribute.md"))
	require.NoError(t, err)
	notes = ParseNotes(file.Body)
	require.Equal(t, "Note: _Steve Jobs_ by Walter Isaacson", notes[1].LongTitle)
	definitions = GetSchemaAttributes(file.RelativePath, notes[1].Kind)
	assert.Equal(t, []*ConfigLintSchemaAttribute{
		{
			Name:     "isbn",
			Type:     "string",
			Pattern:  "^([0-9-]{10}|[0-9]{3}-[0-9]{10})$",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		// Name does not match
		{
			Name:     "references",
			Type:     "array",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "source",
			Type:     "string",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
		{
			Name:     "tags",
			Type:     "array",
			Required: BoolPointer(false),
			Inherit:  BoolPointer(true),
		},
	}, definitions)
}

func TestNoDuplicateNoteTitle(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-duplicate-note-title.md"))
	require.NoError(t, err)

	violations, err := NoDuplicateNoteTitle(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-duplicate-note-title",
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
			Name:         "min-lines-between-notes",
			RelativePath: "min-lines-between-notes.md",
			Message:      `missing blank lines before note "Note: Two"`,
			Line:         7,
		},
		{
			Name:         "min-lines-between-notes",
			RelativePath: "min-lines-between-notes.md",
			Message:      `missing blank lines before note "Note: Four"`,
			Line:         15,
		},
	}, violations)
}

func TestMaxLinesBetweenNotes(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "max-lines-between-notes.md"))
	require.NoError(t, err)

	violations, err := MaxLinesBetweenNotes(file, []string{"2"})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "max-lines-between-notes",
			RelativePath: "max-lines-between-notes.md",
			Message:      `too many blank lines before note "Note: One"`,
			Line:         6,
		},
		{
			Name:         "max-lines-between-notes",
			RelativePath: "max-lines-between-notes.md",
			Message:      `too many blank lines before note "Note: Three"`,
			Line:         16,
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
			Name:         "note-title-match",
			RelativePath: "note-title-match.md",
			Message:      `note title "reference: Example" does not match regex "^(Note|Reference):\\s\\S.*$"`,
			Line:         7,
		},
	}, violations)
}

func TestRequireQuoteTag(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file1, err := ParseFile(filepath.Join(root, "require-quote-tag/require-quote-tag-1.md"))
	require.NoError(t, err)
	file2, err := ParseFile(filepath.Join(root, "require-quote-tag/require-quote-tag-2.md"))
	require.NoError(t, err)

	// Default pattern
	violations, err := RequireQuoteTag(file1, []string{})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "require-quote-tag",
			Message:      "quote \"Quote: No Tag\" does not have tags",
			RelativePath: "require-quote-tag/require-quote-tag-1.md",
			Line:         7,
		},
	}, violations)
	violations, err = RequireQuoteTag(file2, []string{})
	require.NoError(t, err)
	assert.Len(t, violations, 0)

	// Custom pattern
	violations, err = RequireQuoteTag(file1, []string{`^(life|favorite)$`})
	require.NoError(t, err)
	assert.Equal(t, []*Violation{
		{
			Name:         "require-quote-tag",
			Message:      "quote \"Quote: No Tag\" does not have tags",
			RelativePath: "require-quote-tag/require-quote-tag-1.md",
			Line:         7,
		},
		{
			Name:         "require-quote-tag",
			Message:      "quote \"Quote: Tag\" does not have tags", // useless does not match
			RelativePath: "require-quote-tag/require-quote-tag-1.md",
			Line:         14,
		},
	}, violations)
	violations, err = RequireQuoteTag(file2, []string{`^(life|favorite)$`})
	require.NoError(t, err)
	assert.Len(t, violations, 0)
}

func TestNoFreeNote(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-free-note.md"))
	require.NoError(t, err)

	violations, err := NoFreeNote(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-free-note",
			RelativePath: "no-free-note.md",
			Message:      `free note "A free note" not allowed`,
			Line:         3,
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
			Name:         "no-dangling-media",
			RelativePath: "no-dangling-media.md",
			Message:      `dangling media pic.jpeg detected in no-dangling-media.md`,
			Line:         3,
		},
		{
			Name:         "no-dangling-media",
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
			Name:         "no-dead-wikilink",
			RelativePath: "no-dead-wikilink.md",
			Message:      "section not found for wikilink [[#B]]",
			Line:         5,
		},
		{
			Name:         "no-dead-wikilink",
			RelativePath: "no-dead-wikilink.md",
			Message:      "section not found for wikilink [[no-dead-wikilink/sub/file#An Unknown Note]]",
			Line:         12,
		},
		{
			Name:         "no-dead-wikilink",
			RelativePath: "no-dead-wikilink.md",
			Message:      "file not found for wikilink [[no-dead-wikilink/sub/unknown]]",
			Line:         13,
		},
		{
			Name:         "no-dead-wikilink",
			RelativePath: "no-dead-wikilink.md",
			Message:      "file not found for wikilink [[sub/unknown]]",
			Line:         14,
		},
		{
			Name:         "no-dead-wikilink",
			RelativePath: "no-dead-wikilink.md",
			Message:      "file not found for wikilink [[unknown.md]]",
			Line:         15,
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
			Name:         "no-extension-wikilink",
			RelativePath: "no-extension-wikilink.md",
			Message:      `extension found in wikilink [[no-extension-wikilink.md#Note: Link 1]]`,
			Line:         13,
		},
		{
			Name:         "no-extension-wikilink",
			RelativePath: "no-extension-wikilink.md",
			Message:      `extension found in wikilink [[no-extension-wikilink.md]]`,
			Line:         21,
		},
		{
			Name:         "no-extension-wikilink",
			RelativePath: "no-extension-wikilink.md",
			Message:      `extension found in wikilink [[dir/dangling/file.md]]`,
			Line:         25,
		},
	}, violations)
}

func TestNoAmbiguousWikilink(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	file, err := ParseFile(filepath.Join(root, "no-ambiguous-wikilink.md"))
	require.NoError(t, err)

	violations, err := NoAmbiguousWikilink(file, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-ambiguous-wikilink",
			RelativePath: "no-ambiguous-wikilink.md",
			Message:      `ambiguous reference for wikilink [[books.md]]`,
			Line:         3,
		},
		{
			Name:         "no-ambiguous-wikilink",
			RelativePath: "no-ambiguous-wikilink.md",
			Message:      `ambiguous reference for wikilink [[books.md#Treasure Island by Robert Louis Stevenson]]`,
			Line:         6,
		},
	}, violations)
}

func TestCheckAttribute(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	fileRoot, err := ParseFile(filepath.Join(root, "check-attribute.md"))
	require.NoError(t, err)
	fileSub, err := ParseFile(filepath.Join(root, "check-attribute/check-attribute.md"))
	require.NoError(t, err)

	violations, err := CheckAttribute(fileRoot, nil)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Equal(t, []*Violation{
		{
			Name:         "check-attribute",
			Message:      `attribute "isbn" in note "Note: _Steve Jobs_ by Walter Isaacson" in file "check-attribute.md" does not match pattern "^([0-9-]{10}|[0-9]{3}-[0-9]{10})$"`,
			RelativePath: "check-attribute.md",
			Line:         14,
		},
	}, violations)

	violations, err = CheckAttribute(fileSub, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "check-attribute",
			Message:      `attribute "name" missing on note "Quote: Steve Jobs on Life" in file "check-attribute/check-attribute.md"`,
			RelativePath: "check-attribute/check-attribute.md",
			Line:         0,
		},
	}, violations)
}
