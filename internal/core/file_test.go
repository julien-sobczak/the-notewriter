package core

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontMatterString(t *testing.T) {
	var tests = []struct {
		name     string      // name
		input    []Attribute // input
		expected string      // expected result
	}{
		{
			"Scalar values",
			[]Attribute{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key2",
					Value: 2,
				},
			},
			`
key1: value1
key2: 2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := NewFileFromAttributes(tt.input)
			actual, err := file.FrontMatterString()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}

/*
func TestNewAttributeListFromString(t *testing.T) {
	content := `---
id: "Note-20220928-1413"
tags: [text, inspiration]
reference: [[other-note.md]]
url: http://www.google.com
cited:
  - [[very-interesting-note.md]]
  - A New Note System
---

This is a basic multi-line
note saying nothing interesing.

`

	attributes, err := NewAttributeListFromString(content)
	require.NoError(t, err)
	require.Equal(t, 5, len(attributes))
}
*/

func TestNewFile(t *testing.T) {
	f := NewEmptyFile()
	f.SetAttribute("tags", []string{"toto"})

	assert.Equal(t, []interface{}{"toto"}, f.GetAttribute("tags"))

	actual, err := f.FrontMatterString()
	require.NoError(t, err)
	expected := `
tags:
- toto`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func TestNewFileFromPath(t *testing.T) {
	fc, err := os.CreateTemp("", "sample.md")
	require.NoError(t, err)
	defer os.Remove(fc.Name())

	_, err = fc.Write([]byte(`
---
tags: [favorite, inspiration]
---

Blabla`))
	require.NoError(t, err)
	fc.Close()

	// Init the file
	f, err := NewFileFromPath(fc.Name())
	require.NoError(t, err)

	// Check initial content
	assertFrontMatterEqual(t, `tags: [favorite, inspiration]`, f)
	assertContentEqual(t, `Blabla`, f)

	// Override an attribute
	f.SetAttribute("tags", []string{"ancient"})
	assertFrontMatterEqual(t, `tags: [ancient]`, f)

	// Add an attribute
	f.SetAttribute("extras", map[string]string{"key1": "value1", "key2": "value2"})
	assertFrontMatterEqual(t, `
tags: [ancient]
extras:
  key1: value1
  key2: value2
`, f)
}


func TestPreserveComments(t *testing.T) {
	fc, err := os.CreateTemp("", "sample.md")
	require.NoError(t, err)
	defer os.Remove(fc.Name())

	_, err = fc.Write([]byte(`
---
# Front-Matter
tags: [favorite, inspiration] # Custom tags
# published: true
---
`))
	require.NoError(t, err)
	fc.Close()

	// Init the file
	f, err := NewFileFromPath(fc.Name())
	require.NoError(t, err)

	// Change attributes
	f.SetAttribute("tags", []string{"ancient"})
	f.SetAttribute("new", 10)
	assertFrontMatterEqual(t, `
# Front-Matter
tags: [ancient] # Custom tags
# published: true

new: 10
`, f)
	// FIXME debug why an additional newline
}

/* Test Helpers */

func assertFrontMatterEqual(t *testing.T, expected string, file *File) {
	actual, err := file.FrontMatterString()
	require.NoError(t, err)
	assertTrimEqual(t, expected, actual)
}

func assertContentEqual(t *testing.T, expected string, file *File) {
	actual := file.Content
	assertTrimEqual(t, expected, actual)
}

func assertTrimEqual(t *testing.T, expected string, actual string) {
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}
