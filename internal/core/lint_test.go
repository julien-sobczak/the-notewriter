package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoEmptyTitle(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-empty-title.md")

	violations, err := NoEmptyTitle(file, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-empty-title",
			RelativePath: "no-empty-title.md",
			Message:      `note with empty title`,
			Line:         1,
		},
		{
			Name:         "no-empty-title",
			RelativePath: "no-empty-title.md",
			Message:      `note with empty title`,
			Line:         3,
		},
	}, violations)
}

func TestNoDuplicateNoteTitle(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-duplicate-note-title.md")

	violations, err := NoDuplicateNoteTitle(file, nil, nil)
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

func TestNoDuplicateSlug(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	// File a.md is valid
	file := tr.ParseFile("no-duplicate-slug/a.md")

	violations, err := NoDuplicateSlug(file, nil, nil)
	require.NoError(t, err)
	require.Len(t, violations, 0)

	// File b.md contains non-unique slugs
	file = tr.ParseFile("no-duplicate-slug/b.md")

	violations, err = NoDuplicateSlug(file, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-duplicate-slug",
			RelativePath: "no-duplicate-slug/b.md",
			Message:      `duplicated slug "b-note-1"`,
			Line:         11,
		},
		{
			Name:         "no-duplicate-slug",
			RelativePath: "no-duplicate-slug/b.md",
			Message:      `duplicated slug "a-note-1"`,
			Line:         23,
		},
	}, violations)

	// File c.md contains unique slugs but using an invalid format
	file = tr.ParseFile("no-duplicate-slug/c.md")

	violations, err = NoDuplicateSlug(file, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "no-duplicate-slug",
			RelativePath: "no-duplicate-slug/c.md",
			Message:      `invalid slug format "no space allowed"`,
			Line:         7,
		},
		{
			Name:         "no-duplicate-slug",
			RelativePath: "no-duplicate-slug/c.md",
			Message:      `invalid slug format "éà"`,
			Line:         13,
		},
		{
			Name:         "no-duplicate-slug",
			RelativePath: "no-duplicate-slug/c.md",
			Message:      `invalid slug format "TitleCase"`,
			Line:         19,
		},
	}, violations)
}

func TestMinLinesBetweenNotes(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("min-lines-between-notes.md")

	violations, err := MinLinesBetweenNotes(file, nil, []any{2})
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
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("max-lines-between-notes.md")

	violations, err := MaxLinesBetweenNotes(file, nil, []any{2})
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
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("note-title-match.md")

	violations, err := NoteTitleMatch(file, nil, []any{`^(Note|Quote):\s\S.*$`})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "note-title-match",
			RelativePath: "note-title-match.md",
			Message:      `note title "note: Example" does not match regex "^(Note|Quote):\\s\\S.*$"`,
			Line:         7,
		},
	}, violations)
}

func TestRequireTag(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file1 := tr.ParseFile("require-tag/require-tag-1.md")
	file2 := tr.ParseFile("require-tag/require-tag-2.md")

	// Default pattern
	violations, err := RequireTag(file1, nil, []any{})
	require.NoError(t, err)
	require.Equal(t, []*Violation{
		{
			Name:         "require-tag",
			Message:      "note \"Quote: No Tag\" does not have tags",
			RelativePath: "require-tag/require-tag-1.md",
			Line:         3,
		},
	}, violations)
	violations, err = RequireTag(file2, nil, []any{})
	require.NoError(t, err)
	assert.Len(t, violations, 0)

	// Custom pattern
	violations, err = RequireTag(file1, nil, []any{`^(life|favorite)$`})
	require.NoError(t, err)
	assert.Equal(t, []*Violation{
		{
			Name:         "require-tag",
			Message:      "note \"Quote: No Tag\" does not have tags",
			RelativePath: "require-tag/require-tag-1.md",
			Line:         3,
		},
		{
			Name:         "require-tag",
			Message:      "note \"Quote: Tag\" does not have tags", // useless does not match
			RelativePath: "require-tag/require-tag-1.md",
			Line:         10,
		},
	}, violations)
	violations, err = RequireTag(file2, nil, []any{`^(life|favorite)$`})
	require.NoError(t, err)
	assert.Len(t, violations, 0)
}

func TestNoDanglingMedia(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-dangling-media.md")

	violations, err := NoDanglingMedia(file, nil, nil)
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
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-dead-wikilink.md")

	violations, err := NoDeadWikilink(file, nil, nil)
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
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-extension-wikilink.md")

	violations, err := NoExtensionWikilink(file, nil, nil)
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
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	file := tr.ParseFile("no-ambiguous-wikilink.md")

	violations, err := NoAmbiguousWikilink(file, nil, nil)
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

func TestCheckAttributes(t *testing.T) {
	tr := NewTestRepository(t, FromGoldenDirNamed("TestLint"))

	fileRoot := tr.ParseFile("check-attributes.md")
	fileSub := tr.ParseFile("check-attributes/check-attributes.md")

	violations, err := CheckAttributes(fileRoot)
	require.NoError(t, err)
	require.Len(t, violations, 2)
	require.ElementsMatch(t, []*Violation{
		{
			Name:         "check-attributes",
			Message:      `attribute "name" missing on note "Quote: Steve Jobs on Passion" in file "check-attributes.md"`,
			RelativePath: "check-attributes.md",
			Line:         3,
		},
		{
			Name:         "check-attributes",
			Message:      `attribute "isbn" on note "Note: _Steve Jobs_ by Walter Isaacson" in file "check-attributes.md" does not match pattern "^([0-9-]{10}|[0-9]{3}-[0-9]{10})$"`,
			RelativePath: "check-attributes.md",
			Line:         14,
		},
	}, violations)

	violations, err = CheckAttributes(fileSub)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.ElementsMatch(t, []*Violation{
		{
			Name:         "check-attributes",
			Message:      `attribute "name" missing on note "Quote: Steve Jobs on Life" in file "check-attributes/check-attributes.md"`,
			RelativePath: "check-attributes/check-attributes.md",
			Line:         7,
		},
	}, violations)
}
