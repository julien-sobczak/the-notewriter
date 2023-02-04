package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoteQuote(t *testing.T) {
	f := NewEmptyFile()
	f.SetAttribute("tags", []string{"favorite"})
	f.SetAttribute("name", "Austin Kleon")

	note := NewNote(f, "Quote: On Advices",
		"`#creativity`\n\n<!-- source: Steal Like an Artist -->\n\nWhen people give you advice, they’re really just talking to themselves in the past.", 10)

	assert.Equal(t, "> When people give you advice, they’re really just talking to themselves in the past.\n> -- Austin Kleon", note.ContentMarkdown)
	// FIXME uncomment
	// assert.Equal(t, "<blockquote>When people give you advice, they’re really just talking to themselves in the past. — Austin Kleon</blockquote>", note.ContentHTML)
	// assert.Equal(t, "When people give you advice, they’re really just talking to themselves in the past. — Austin Kleon", note.ContentText)
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
			file := NewEmptyFile()
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile(), tt.title, tt.content, 1)
			links, err := note.GetLinks()
			require.NoError(t, err)
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
	clock.FreezeAt(time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
	defer clock.Unfreeze()

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
			// TODO complete with more supported syntaxes
			expected: []*Reminder{
				{
					DescriptionRaw:  "Buy **Lego Christmas** sets to create a village",
					Tag:             "#reminder-2025-09",
					NextPerformedAt: time.Date(2025, time.Month(9), 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := NewNote(NewEmptyFile(), tt.title, tt.content, 1)
			reminders, err := note.GetReminders()
			require.NoError(t, err)
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
