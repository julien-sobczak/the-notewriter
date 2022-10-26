package markdown_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMarkdown(t *testing.T) {
	var tests = []struct {
		name string // name
	}{
		{"BasicMarkdown"},
		{"StripTags"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			input, err := ioutil.ReadFile(filepath.Join("testdata", tt.name+".md"))
			require.NoError(t, err)
			output, err := ioutil.ReadFile(filepath.Join("testdata", tt.name+".markdown.md"))
			require.NoError(t, err)

			actual := markdown.ToMarkdown(string(input))
			assert.Equal(t, strings.TrimSpace(string(output)), actual)
		})
	}
}
