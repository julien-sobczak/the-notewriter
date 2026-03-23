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
		q := `#favorite keyword1 type:note type:flashcard @title:"Note Title" path:"projects/toto" "keyword 2" #life-changing @name:Epictectus`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{{Operator: "=", Operand: "projects/toto"}}, query.Path)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "note"},
			{Operator: "=", Operand: "flashcard"},
		}, query.Types)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "favorite"},
			{Operator: "=", Operand: "life-changing"},
		}, query.Tags)
		assert.EqualValues(t, map[string]QueryCondition{
			"title": {Operator: "=", Operand: "Note Title"},
			"name":  {Operator: "=", Operand: "Epictectus"},
		}, query.Attributes)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "keyword1"},
			{Operator: "=", Operand: "keyword 2"},
		}, query.Terms)
	})

	t.Run("Invalid", func(t *testing.T) {
		_, err := ParseQuery("#")
		require.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("PathWithoutQuotes", func(t *testing.T) {
		q := `path:thoughts/on-learning.md`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{{Operator: "=", Operand: "thoughts/on-learning.md"}}, query.Path)
	})

	t.Run("PathWithQuotesAndSpaces", func(t *testing.T) {
		q := `path:"thoughts with spaces/on-learning.md"`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{{Operator: "=", Operand: "thoughts with spaces/on-learning.md"}}, query.Path)
	})

	t.Run("NestedTagsWithSlash", func(t *testing.T) {
		q := `#todo/read #project/personal/goals`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "todo/read"},
			{Operator: "=", Operand: "project/personal/goals"},
		}, query.Tags)
	})

	t.Run("MixedPathsAndTags", func(t *testing.T) {
		q := `#todo/read path:projects/learning.md #done/completed path:"with spaces/file.md"`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "projects/learning.md"},
			{Operator: "=", Operand: "with spaces/file.md"},
		}, query.Path)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "todo/read"},
			{Operator: "=", Operand: "done/completed"},
		}, query.Tags)
	})

	t.Run("NegationOnPath", func(t *testing.T) {
		q := `-path:/top/sub/directory +path:/top`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "<>", Operand: "/top/sub/directory"},
			{Operator: "=", Operand: "/top"},
		}, query.Path)
	})

	t.Run("NegationOnType", func(t *testing.T) {
		q := `-type:Note type:Flashcard`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "<>", Operand: "Note"},
			{Operator: "=", Operand: "Flashcard"},
		}, query.Types)
	})

	t.Run("NegationOnTag", func(t *testing.T) {
		q := `#favorite -#archived`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "favorite"},
			{Operator: "<>", Operand: "archived"},
		}, query.Tags)
	})

	t.Run("NegationOnTerm", func(t *testing.T) {
		q := `golang -deprecated`
		query, err := ParseQuery(q)
		require.NoError(t, err)
		assert.EqualValues(t, []QueryCondition{
			{Operator: "=", Operand: "golang"},
			{Operator: "<>", Operand: "deprecated"},
		}, query.Terms)
	})

}
