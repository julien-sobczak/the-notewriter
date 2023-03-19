package core

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoteQuote(t *testing.T) {
	f := NewEmptyFile("example.md")
	f.SetAttribute("tags", []string{"favorite"})
	f.SetAttribute("name", "Austin Kleon")

	note := NewNote(f, "Quote: On Advices",
		"`#creativity`\n\n<!-- source: Steal Like an Artist -->\n\nWhen people give you advice, they’re really just talking to themselves in the past.", 10)

	assert.Equal(t, "> When people give you advice, they’re really just talking to themselves in the past.\n> -- Austin Kleon", note.ContentMarkdown)
	assert.Equal(t, "<blockquote>\n<p>When people give you advice, they’re really just talking to themselves in the past.\n&ndash; Austin Kleon</p>\n</blockquote>", note.ContentHTML)
	// FIXME https://stackoverflow.com/a/49212532 put the author in <p> outside the blockquote
	assert.Equal(t, "\"When people give you advice, they’re really just talking to themselves in the past.\n-- Austin Kleon\"", note.ContentText)
	// FIXME put the author output the quotation marks
}

func TestMergeTags(t *testing.T) {
	var tests = []struct {
		name     string // name
		inputA   []string
		inputB   []string
		expected []string
	}{
		{
			"empty slices",
			nil,
			nil,
			nil,
		},
		{
			"empty slice",
			[]string{"favorite"},
			nil,
			[]string{"favorite"},
		},
		{
			"single value",
			[]string{"favorite"},
			[]string{"life-changing"},
			[]string{"favorite", "life-changing"},
		},
		{
			"multiple values",
			[]string{"a", "b"},
			[]string{"c", "d"},
			[]string{"a", "b", "c", "d"},
		},
		{
			"duplicates",
			[]string{"a", "b"},
			[]string{"b", "c"},
			[]string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mergeTags(tt.inputA, tt.inputB)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMergeAttributes(t *testing.T) {
	var tests = []struct {
		name     string // name
		inputA   map[string]interface{}
		inputB   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			"nil maps",
			nil,
			nil,
			nil,
		},
		{
			"append in slices",
			map[string]interface{}{
				"tags": []string{"a", "b"},
			},
			map[string]interface{}{
				"tags": "c",
			},
			map[string]interface{}{
				"tags": []string{"a", "b", "c"},
			},
		},
		{
			"append in slices",
			map[string]interface{}{
				"tags": []string{"a", "b"},
			},
			map[string]interface{}{
				"tags": []string{"c", "d"},
			},
			map[string]interface{}{
				"tags": []string{"a", "b", "c", "d"},
			},
		},
		{
			"override basic value",
			map[string]interface{}{
				"tags": "a",
			},
			map[string]interface{}{
				"tags": "b",
			},
			map[string]interface{}{
				"tags": "b",
			},
		},
		{
			"add new keys",
			map[string]interface{}{
				"a": "a",
			},
			map[string]interface{}{
				"b": "b",
			},
			map[string]interface{}{
				"a": "a",
				"b": "b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mergeAttributes(tt.inputA, tt.inputB)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewNoteParsing(t *testing.T) {
	var tests = []struct {
		name    string
		input   string
		content string
		strict  bool
	}{

		{
			name: "Note: Basic",
			input: `
This is a **basic, multiline note**
that says nothing _interesting_.`,
			// Nothing to do
			content: `
This is a **basic, multiline note**
that says nothing _interesting_.`,
		},

		{
			name: "Quote: without author",
			input: `
We are all born ignorant, but one must work
hard to remain stupid.`,
			// Prefix line with quotation syntax
			content: `
> We are all born ignorant, but one must work
> hard to remain stupid.`,
		},

		{
			name: "Quote: with author",
			input: `
<!-- name: Henry Ford -->
Quality means doing it right when no one is looking.`,
			// Append the author after the quotation
			content: `
> Quality means doing it right when no one is looking.
> -- Henry Ford`,
		},

		{
			name: "Quote: with tags",
			input: "\n" +
				"`#favorite` `#life`\n" +
				"Action is a necessary part of success.\n" +
				"\n" +
				"`#todo`\n",
			// Strip tags
			content: "> Action is a necessary part of success.\n",
			strict:  true,
		},

		{
			name: "Quote: With attribute",
			input: `
<!-- source: https://jamesclear.com/3-2-1/july-14-2022 -->
<!-- author: James Clear -->

Knowledge is making the right choice with all the information.
Wisdom is making the right choice without all the information.
`,
			// Strip attributes
			content: "> Knowledge is making the right choice with all the information.\n> Wisdom is making the right choice without all the information.\n> -- James Clear\n",
			strict:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := NewEmptyFile("example.md")
			actual := NewNote(file, tt.name, tt.input, 10)
			content, _, _ := actual.parseContentRaw()
			if tt.strict {
				assert.Equal(t, tt.content, content)
			} else {
				assert.Equal(t, strings.TrimSpace(tt.content), strings.TrimSpace(content))
			}
		})
	}
}

func TestGetLinks(t *testing.T) {
	var tests = []struct {
		name     string  // name
		title    string  // input
		content  string  // input
		expected []*Link // output
	}{
		{
			name:  "Different link syntaxes",
			title: "Cheatsheet: How to create a module?",
			content: `
[Link 1](https://docs.npmjs.com "Tutorial to creating Node.js modules")
[Link 2](https://docs.npmjs.com "Tutorial to creating Node.js modules #go/node/module")
[Link 3](https://docs.npmjs.com "#go/node/module")
[Link 4](https://docs.npmjs.com)`,
			expected: []*Link{
				{
					Text:   "Link 2",
					URL:    "https://docs.npmjs.com",
					GoName: "node/module",
					Title:  "Tutorial to creating Node.js modules",
				},
				{
					Text:   "Link 3",
					URL:    "https://docs.npmjs.com",
					GoName: "node/module",
					Title:  "",
				},
			},
		},
	}

	SetUpCollectionFromTempDir(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile("example.md"), tt.title, tt.content, 1)
			links := note.GetLinks()
			require.Len(t, links, len(tt.expected))
			for i, actualLink := range links {
				expectedLink := tt.expected[i]
				assert.Equal(t, expectedLink.Text, actualLink.Text)
				assert.Equal(t, expectedLink.GoName, actualLink.GoName)
				assert.Equal(t, expectedLink.URL, actualLink.URL)
				assert.Equal(t, expectedLink.Title, actualLink.Title)
			}
		})
	}
}

func TestGetReminders(t *testing.T) {
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	var tests = []struct {
		name     string      // name
		title    string      // input
		content  string      // input
		expected []*Reminder // output
	}{
		{
			name:  "Different reminder syntaxes",
			title: "TODO: Activities",
			content: "\n" +
				"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\n",
			expected: []*Reminder{
				{
					DescriptionRaw:  "Buy **Lego Christmas** sets to create a village",
					Tag:             "#reminder-2025-09",
					NextPerformedAt: time.Date(2025, time.Month(9), 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	SetUpCollectionFromTempDir(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile("example.md"), tt.title, tt.content, 1)
			reminders := note.GetReminders()
			require.Len(t, reminders, len(tt.expected))
			for i, actualReminder := range reminders {
				expectedReminder := tt.expected[i]
				if expectedReminder.DescriptionRaw != "" {
					assert.Equal(t, expectedReminder.DescriptionRaw, actualReminder.DescriptionRaw)
				}
				if expectedReminder.Tag != "" {
					assert.Equal(t, expectedReminder.Tag, actualReminder.Tag)
				}
				if !expectedReminder.NextPerformedAt.IsZero() {
					assert.EqualValues(t, expectedReminder.NextPerformedAt, actualReminder.NextPerformedAt)
				}
			}
		})
	}
}

func TestNoteFormat(t *testing.T) {
	UseFixedOID(t, "16252dafd6355e678bf8ae44b127f657cd3cdd0e")

	var tests = []struct {
		name             string // name
		title            string // input
		content          string // input
		expectedJSON     string // output
		expectedMarkdown string // output
		expectedHTML     string // output
		expectedText     string // output
	}{
		{
			name:  "Basic note",
			title: "TODO: **Activities**",
			content: "\n" +
				"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\n",
			expectedJSON:     "{\n \"oid\": \"16252dafd6355e678bf8ae44b127f657cd3cdd0e\",\n \"relativePath\": \"\",\n \"wikilink\": \"#TODO: **Activities**\",\n \"frontMatter\": null,\n \"tags\": [\n  \"reminder-2025-09\"\n ],\n \"contentRaw\": \"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\",\n \"contentMarkdown\": \"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\",\n \"contentHTML\": \"\\u003cul\\u003e\\n\\u003cli\\u003e[ ] Buy \\u003cstrong\\u003eLego Christmas\\u003c/strong\\u003e sets to create a village \\u003ccode\\u003e#reminder-2025-09\\u003c/code\\u003e\\u003c/li\\u003e\\n\\u003c/ul\\u003e\",\n \"contentText\": \"* [ ] Buy Lego Christmas sets to create a village `#reminder-2025-09`\"\n}",
			expectedMarkdown: "# TODO: **Activities**\n\n* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`",
			expectedHTML:     "<h1><p>TODO: <strong>Activities</strong></p></h1>\n\n<ul>\n<li>[ ] Buy <strong>Lego Christmas</strong> sets to create a village <code>#reminder-2025-09</code></li>\n</ul>",
			expectedText:     "TODO: Activities\n\n* [ ] Buy Lego Christmas sets to create a village `#reminder-2025-09`",
			// TODO use backtip for more a readable test?
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile(""), tt.title, tt.content, 1)
			actualJSON := note.FormatToJSON()
			actualMarkdown := note.FormatToMarkdown()
			actualHTML := note.FormatToHTML()
			actualText := note.FormatToText()
			assert.Equal(t, tt.expectedJSON, actualJSON)
			assert.Equal(t, tt.expectedMarkdown, actualMarkdown)
			assert.Equal(t, tt.expectedHTML, actualHTML)
			assert.Equal(t, tt.expectedText, actualText)
		})
	}
}

func TestSearchNotes(t *testing.T) {
	SetUpCollectionFromGoldenDirNamed(t, "TestNoteFTS")

	db := CurrentDB().Client()
	CurrentLogger().SetVerboseLevel(VerboseTrace)

	// Insert a note
	note := NewNote(NewEmptyFile("example.md"), "Reference: FTS5", "TODO", 2)
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	err = note.InsertWithTx(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Search the note using a full-text query
	notes, err := SearchNotes("kind:reference fts5")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Update the note content
	note.updateContent("full-text")
	tx, err = db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	err = note.UpdateWithTx(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Search the note using a full-text query
	notes, err = SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Delete the note
	tx, err = db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	err = note.DeleteWithTx(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Check the note is no longer
	notes, err = SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}

func TestNote(t *testing.T) {
	// Make tests reproductible
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("YAML", func(t *testing.T) {
		noteSrc := NewNote(NewEmptyFile("example.md"), "TODO: Backlog", "* [ ] Test", 2)

		// Marshall
		buf := new(bytes.Buffer)
		err := noteSrc.Write(buf)
		require.NoError(t, err)
		noteYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
parent_note_oid: ""
kind: todo
title: 'TODO: Backlog'
short_title: Backlog
relative_path: example.md
wikilink: 'example.md#TODO: Backlog'
line: 2
content_raw: '* [ ] Test'
content_hash: 40c0dbcb392522d74c890ff92bcb3fec
content_markdown: '* [ ] Test'
content_html: |-
    <ul>
    <li>[ ] Test</li>
    </ul>
content_text: '* [ ] Test'
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(noteYAML))

		// Unmarshall
		noteDest := new(Note)
		err = noteDest.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, cleanNote(noteSrc), cleanNote(noteDest))
	})

}

/* Test Helpers */

// cleanNote ignore some values as EqualValues is very strict.
func cleanNote(n *Note) *Note {
	// Do not compare state management attributes
	n.File = nil // Do not recurse on file
	n.new = false
	n.stale = false
	if len(n.Attributes) == 0 {
		n.Attributes = nil // nil or empty is the same
	}
	return n
}
