package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttributeListFrontMatterString(t *testing.T) {
	var tests = []struct {
		name     string        // name
		input    AttributeList // input
		expected string        // expected result
	}{
		{
			"Scalar values",
			AttributeList{
				&Attribute{
					Key:   "key1",
					Value: "value1",
				},
				&Attribute{
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
			actual, err := tt.input.FrontMatterString()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}

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

func TestFile(t *testing.T) {
	f := NewFile()
	f.AddAttribute("tags", []string{"toto"})
	f.GetAttribute("tags")
	f.FrontMatterString()

	f := NewFileFromPath("toto.md")

}
