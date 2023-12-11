package reference_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

type DummyResult struct {
	ID         int
	Title      string
	attributes map[string]any
}

func (r *DummyResult) Description() string {
	return fmt.Sprintf("%d: %s", r.ID, r.Title)
}
func (r *DummyResult) Attributes() map[string]any {
	result := make(map[string]any)
	result["id"] = r.ID
	result["title"] = r.Title
	for k, v := range r.attributes {
		result[k] = v
	}
	return result
}

type DummyManager struct {
}

func (m *DummyManager) Ready() (bool, error) {
	return true, nil
}
func (m *DummyManager) Search(query string) ([]reference.Result, error) {
	return []reference.Result{
		&DummyResult{
			ID:    1,
			Title: "Item 1",
			attributes: map[string]any{
				"author":      "Bob",
				"illustrator": "Sponge",
				"content":     "Blabla",
			},
		},
		&DummyResult{
			ID:    3,
			Title: "Item 3",
			attributes: map[string]any{
				"author":      "Krab",
				"illustrator": "Captain",
				"content":     "Blabla",
			},
		},
	}, nil
}

func TestManager(t *testing.T) {
	manager := DummyManager{}
	results, _ := manager.Search("dummy")
	raw := "" +
		"## {{index . \"title\"}}\n" +
		"\n" +
		"`@id: {{index . \"id\"}}` `@author: {{index . \"author\"}}`\n" +
		"\n" +
		"{{index . \"content\"}}\n"

	tmpl, err := template.New("").Parse(raw)
	require.NoError(t, err)

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, results[0].Attributes())
	require.NoError(t, err)
	actual := tpl.String()
	expected := "" +
		"## Item 1\n" +
		"\n" +
		"`@id: 1` `@author: Bob`\n" +
		"\n" +
		"Blabla\n"
	assert.Equal(t, expected, actual)
}

/* Basic Result to test templates */
type MapResult map[string]any

func (r MapResult) Description() string {
	return fmt.Sprintf("map[%d]", len(r))
}

func (r MapResult) Attributes() map[string]any {
	return r
}

func TestEvaluateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		result   MapResult // input
		template string    // input
		output   string    // output
	}{
		{
			name: "Basic",
			result: map[string]any{
				"name":     "Julien",
				"location": "Douai",
			},
			template: `{{index . "name"}}`,
			output:   "Julien",
		},
		{
			name: "Conditional",
			result: map[string]any{
				"name":        "Julien",
				"nationality": "French",
				"location":    "Douai",
			},
			template: `---
name: {{index . "name"}}
{{- if index . "occupation"}}
occupation: {{index . "occupation"}}
{{end -}}
{{- if index . "nationality"}}
nationality: {{index . "nationality"}}
{{end -}}
location: {{index . "location"}}
---`,
			output: `---
name: Julien
nationality: French
location: Douai
---`,
		},

		// Custom functions
		{
			name: "json",
			result: map[string]any{
				"title": "My Note",
				"tags":  []string{"favorite", "learning"},
				"pages": 210,
			},
			template: `{{json .}}`,
			output:   `{"pages":210,"tags":["favorite","learning"],"title":"My Note"}`,
		},
		{
			name: "jsonPretty",
			result: map[string]any{
				"title": "My Note",
				"tags":  []string{"favorite", "learning"},
				"pages": 210,
			},
			template: `{{jsonPretty .}}`,
			output: `{
  "pages": 210,
  "tags": [
    "favorite",
    "learning"
  ],
  "title": "My Note"
}`,
		},
		{
			name: "yaml",
			result: map[string]any{
				"title": "My Note",
				"tags":  []string{"favorite", "learning"},
				"pages": 210,
			},
			template: `{{yaml .}}`,
			output: `pages: 210
tags:
    - favorite
    - learning
title: My Note
`,
		},
		{
			name: "jq",
			result: map[string]any{
				"title": "My Note",
				"tags":  []string{"favorite", "learning"},
				"pages": 210,
			},
			template: `{{jq ". | {title,pages}" .}}`,
			output:   `map[pages:210 title:My Note]`,
		},
		{
			name: "jq (advanced)",
			// See https://lzone.de/cheat-sheet/jq for examples
			result: map[string]any{
				"timestamp": 1234567890,
				"report":    "Age Report",
				"results": []any{
					map[string]any{"name": "John", "age": 43, "city": "TownA"},
					map[string]any{"name": "Joe", "age": 10, "city": "TownB"},
				},
			},
			template: `{{jq ".results[] | select((.name == \"Joe\") and (.age = 10))" .}}`,
			output:   "map[age:10 city:TownB name:Joe]",
		},
		{
			name: "title",
			result: map[string]any{
				"title": "my note",
				"pages": 210,
			},
			template: `{{index . "title" | title}}`,
			output:   `My Note`,
		},
		{
			name: "slug",
			result: map[string]any{
				"title": "my note",
				"pages": 210,
			},
			template: `{{index . "title" | slug}}`,
			output:   `my-note`,
		},
		{
			name: "join (with []string)",
			result: map[string]any{
				"authors": []string{"Bob", "Alice"},
				"pages":   210,
			},
			template: `{{index . "authors" | join ", " }}`,
			output:   "Bob, Alice",
		},
		{
			name: "join (with string)",
			result: map[string]any{
				"authors": "Alice",
				"pages":   210,
			},
			template: `{{index . "authors" | join ", " }}`,
			output:   "Alice",
		},
		{
			name: "join (with any)",
			result: map[string]any{
				"authors": []interface{}{"Bob", "Alice"},
				"pages":   210,
			},
			template: `{{index . "authors" | join ", " }}`,
			output:   "Bob, Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := reference.EvaluateTemplate(tt.template, &tt.result)
			require.NoError(t, err)
			assert.Equal(t, tt.output, actual)
		})
	}
}
