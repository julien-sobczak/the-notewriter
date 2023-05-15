package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsing(t *testing.T) {

	t.Run("Tags & Attributes", func(t *testing.T) {
		var tests = []struct {
			name       string
			input      string                 // Input
			tags       []string               // Output
			attributes map[string]interface{} // Output
		}{
			{
				name: "Only tags",
				input: "\n" +
					"`#favorite` `#life-changing` `#0Aa-1Bb`\n" +
					"\n",
				tags: []string{"favorite", "life-changing", `0Aa-1Bb`},
				attributes: map[string]interface{}{
					"tags":  []interface{}{"favorite", "life-changing", `0Aa-1Bb`},
					"title": "Only tags",
				},
			},

			{
				name: "Only attributes",
				input: "\n" +
					"`@isbn: 9780807014271` `@name: Viktor Frankl` `@source: https://en.wikipedia.org/wiki/Man%27s_Search_for_Meaning`\n" +
					"\n",
				tags: nil,
				attributes: map[string]interface{}{
					"isbn":   "9780807014271",
					"name":   "Viktor Frankl",
					"source": "https://en.wikipedia.org/wiki/Man%27s_Search_for_Meaning",
					"title":  "Only attributes",
				},
			},

			{
				name: "Mixed on single lines",
				input: "\n" +
					"`#favorite` `@isbn: 9780807014271` `#life-changing` `@name: Viktor Frankl`\n" +
					"\n",
				tags: []string{"favorite", "life-changing"},
				attributes: map[string]interface{}{
					"isbn":  "9780807014271",
					"name":  "Viktor Frankl",
					"tags":  []interface{}{"favorite", "life-changing"},
					"title": "Mixed on single lines",
				},
			},

			{
				name: "Mixed on different lines",
				input: "\n" +
					"`#favorite` `#life-changing`\n" +
					"`@isbn: 9780807014271` `@name: Viktor Frankl`\n" +
					"\n",
				tags: []string{"favorite", "life-changing"},
				attributes: map[string]interface{}{
					"isbn":  "9780807014271",
					"name":  "Viktor Frankl",
					"tags":  []interface{}{"favorite", "life-changing"},
					"title": "Mixed on different lines",
				},
			},

			{
				name: "Array Types",
				input: "\n" +
					"`@references: [[fileA]]`\n" +
					"`@references: [[fileB#Section]]`\n",
				tags: []string{},
				attributes: map[string]interface{}{
					"references": []interface{}{
						"[[fileA]]",
						"[[fileB#Section]]",
					},
					"title": "Array Types",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				SetUpCollectionFromTempDir(t)

				// Preconditions
				schemasTypes := GetSchemaAttributeTypes()
				require.Equal(t, "array", schemasTypes["references"])

				file := NewEmptyFile("example.md")
				actual := NewNote(file, nil, tt.name, tt.input, 10)
				if len(tt.tags) == 0 {
					tt.tags = nil
				}
				if len(actual.Tags) == 0 {
					actual.Tags = nil
				}
				if len(tt.attributes) == 0 {
					tt.attributes = nil
				}
				if len(actual.Attributes) == 0 {
					actual.Attributes = nil
				}
				assert.EqualValues(t, tt.tags, actual.Tags)
				assert.EqualValues(t, tt.attributes, actual.Attributes)
			})
		}
	})

	t.Run("Body", func(t *testing.T) {
		var tests = []struct {
			name    string
			input   string // Input
			content string // Output
			strict  bool   // Output
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
` + "`@name: Henry Ford`" + `
Quality means doing it right when no one is looking.`,
				// Append the author after the quotation
				content: `
> Quality means doing it right when no one is looking.
> — Henry Ford`,
			},

			{
				name: "Quote: with tags",
				input: "\n" +
					"`#favorite` `#life`\n" +
					"Action is a necessary part of success.\n" +
					"\n" +
					"`#todo`\n",
				// Strip tags
				content: "> Action is a necessary part of success.",
				strict:  true,
			},

			{
				name: "Quote: With attribute",
				input: `
` + "`@source: https://jamesclear.com/3-2-1/july-14-2022`" + `
` + "`@author: James Clear`" + `

Knowledge is making the right choice with all the information.
Wisdom is making the right choice without all the information.
`,
				// Strip attributes
				content: "> Knowledge is making the right choice with all the information.\n> Wisdom is making the right choice without all the information.\n> — James Clear",
				strict:  true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				SetUpCollectionFromTempDir(t)
				file := NewEmptyFile("example.md")
				actual := NewNote(file, nil, tt.name, tt.input, 10)
				_, _, _, markdown, _, _, _, _, _ := actual.parseContentRaw()
				if tt.strict {
					assert.Equal(t, tt.content, markdown)
				} else {
					assert.Equal(t, strings.TrimSpace(tt.content), strings.TrimSpace(markdown))
				}
			})
		}
	})

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
			note := NewNote(NewEmptyFile("example.md"), nil, tt.title, tt.content, 1)
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
			note := NewNote(NewEmptyFile("example.md"), nil, tt.title, tt.content, 1)
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
	SetUpCollectionFromTempDir(t)
	UseFixedOID(t, "16252dafd6355e678bf8ae44b127f657cd3cdd0e")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	var tests = []struct {
		name         string // name
		title        string // input
		content      string // input
		expectedJSON string // output

		expectedMarkdown string // output
		expectedHTML     string // output
		expectedText     string // output
	}{
		{
			name:  "Basic note",
			title: "TODO: **Activities**",
			content: "\n" +
				"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\n",
			expectedJSON:     "{\n \"oid\": \"16252dafd6355e678bf8ae44b127f657cd3cdd0e\",\n \"relativePath\": \"\",\n \"wikilink\": \"#TODO: **Activities**\",\n \"attributes\": {\n  \"title\": \"**Activities**\"\n },\n \"tags\": null,\n \"shortTitleRaw\": \"**Activities**\",\n \"shortTitleMarkdown\": \"**Activities**\",\n \"shortTitleHTML\": \"\\u003cp\\u003e\\u003cstrong\\u003eActivities\\u003c/strong\\u003e\\u003c/p\\u003e\",\n \"shortTitleText\": \"Activities\",\n \"contentRaw\": \"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\",\n \"contentMarkdown\": \"* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`\",\n \"contentHTML\": \"\\u003cul\\u003e\\n\\u003cli\\u003e[ ] Buy \\u003cstrong\\u003eLego Christmas\\u003c/strong\\u003e sets to create a village \\u003ccode\\u003e#reminder-2025-09\\u003c/code\\u003e\\u003c/li\\u003e\\n\\u003c/ul\\u003e\",\n \"contentText\": \"* [ ] Buy Lego Christmas sets to create a village `#reminder-2025-09`\",\n \"createdAt\": \"2023-01-01T01:12:30Z\",\n \"updatedAt\": \"2023-01-01T01:12:30Z\",\n \"deletedAt\": null\n}",
			expectedMarkdown: "# TODO: **Activities**\n\n* [ ] Buy **Lego Christmas** sets to create a village `#reminder-2025-09`",
			expectedHTML:     "<h1><p>TODO: <strong>Activities</strong></p></h1>\n\n<ul>\n<li>[ ] Buy <strong>Lego Christmas</strong> sets to create a village <code>#reminder-2025-09</code></li>\n</ul>",
			expectedText:     "TODO: Activities\n\n* [ ] Buy Lego Christmas sets to create a village `#reminder-2025-09`",
			// TODO use backtip for more a readable test?
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile(""), nil, tt.title, tt.content, 1)
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

	CurrentLogger().SetVerboseLevel(VerboseTrace)

	// Insert a note
	note := NewNote(NewEmptyFile("example.md"), nil, "Reference: FTS5", "TODO", 2)
	err := CurrentDB().BeginTransaction()
	require.NoError(t, err)
	err = note.Insert()
	require.NoError(t, err)
	err = CurrentDB().CommitTransaction()
	require.NoError(t, err)

	// Search the note using a full-text query
	notes, err := CurrentCollection().SearchNotes("kind:reference fts5")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Update the note content
	note.updateContent("full-text")
	err = CurrentDB().BeginTransaction()
	require.NoError(t, err)
	err = note.Update()
	require.NoError(t, err)
	err = CurrentDB().CommitTransaction()
	require.NoError(t, err)

	// Search the note using a full-text query
	notes, err = CurrentCollection().SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	// Delete the note
	err = CurrentDB().BeginTransaction()
	require.NoError(t, err)
	err = note.Delete()
	require.NoError(t, err)
	err = CurrentDB().CommitTransaction()
	require.NoError(t, err)

	// Check the note is no longer
	notes, err = CurrentCollection().SearchNotes("kind:reference full")
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}

func TestNote(t *testing.T) {

	t.Run("YAML", func(t *testing.T) {
		SetUpCollectionFromTempDir(t)

		// Make tests reproductible
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

		noteSrc := NewNote(NewEmptyFile("example.md"), nil, "TODO: Backlog", "* [ ] Test", 2)

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
long_title: Backlog
short_title: Backlog
relative_path: example.md
wikilink: 'example.md#TODO: Backlog'
attributes:
    title: Backlog
line: 2
content_raw: '* [ ] Test'
content_hash: 3d14810b17a61366392fce8a69fbebf5d685f2fb
title_markdown: '# Backlog'
title_html: <h1>Backlog</h1>
title_text: |-
    Backlog
    =======
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

func TestFormatLongTitle(t *testing.T) {
	tests := []struct {
		name      string
		titles    []string // input
		longTitle string   // output
	}{
		{
			name:      "Basic",
			titles:    []string{"Go", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Empty titles",
			titles:    []string{"", "History"},
			longTitle: "History",
		},
		{
			name:      "Duplicate titles",
			titles:    []string{"Go", "History", "History"},
			longTitle: "Go / History",
		},
		{
			name:      "Common prefix",
			titles:    []string{"Go", "Go History"},
			longTitle: "Go History",
		},
		{
			name:      "Not common prefix",
			titles:    []string{"Go", "Goroutines"},
			longTitle: "Go / Goroutines",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatLongTitle(tt.longTitle)
			assert.Equal(t, tt.longTitle, actual)
		})
	}
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
