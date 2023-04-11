package core

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoTextScanner(t *testing.T) {
	// Learning test to demonstrate the standard API

	const src = `
// This is scanned code.
if a > 10 {
	someParsable = "some text"
}`

	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Filename = "example"

	var tokens []string
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		t.Logf("%s: %s", s.Position, s.TokenText())
		tokens = append(tokens, s.TokenText())
	}

	expected := []string{
		"if",
		"a",
		">",
		"10",
		"{",
		"someParsable",
		"=",
		"\"some text\"",
		"}",
	}
	assert.EqualValues(t, expected, tokens)
}

func TestGoTextScannerWithQuery(t *testing.T) {
	// Same as above but with a specific query
	const src = `#tag subject @title:"Note Title"`

	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Filename = ""

	var tokens []string
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		t.Logf("%s: %s", s.Position, s.TokenText())
		tokens = append(tokens, s.TokenText())
	}

	expected := []string{
		`#`, `tag`, `subject`, `@`, `title`, `:`, `"Note Title"`,
	}
	assert.EqualValues(t, expected, tokens)
}

func TestParseQuery(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		q := `#favorite keyword1 kind:note kind:flashcard @title:"Note Title" path:"projects/toto" "keyword 2" #life-changing @name:Epictectus`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.Equal(t, "projects/toto", query.Path)
		assert.EqualValues(t, []string{"note", "flashcard"}, query.Kinds)
		assert.EqualValues(t, []string{"favorite", "life-changing"}, query.Tags)
		assert.EqualValues(t, map[string]interface{}{
			"title": "Note Title",
			"name":  "Epictectus",
		}, query.Attributes)
		assert.EqualValues(t, []string{"keyword1", "keyword 2"}, query.Terms)
	})

	t.Run("Invalid", func(t *testing.T) {
		_, err := ParseQuery("#")
		require.ErrorContains(t, err, "unexpected EOF")
	})

}
