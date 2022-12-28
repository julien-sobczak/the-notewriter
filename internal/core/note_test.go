package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			if tt.strict {
				assert.Equal(t, tt.content, actual.Content)
			} else {
				assert.Equal(t, strings.TrimSpace(tt.content), strings.TrimSpace(actual.Content))
			}
		})
	}
}
