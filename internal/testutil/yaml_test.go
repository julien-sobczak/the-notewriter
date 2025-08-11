package testutil_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertYAMLMatches(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		yamlStr := `
title: "Test Note"
tags:
  - tag1
  - tag2
source: "https://example.com"
`
		template := map[string]any{
			"title":  "Test Note",
			"tags":   []any{"tag1", "tag2"},
			"source": "https://example.com",
		}
		testutil.AssertYAMLMatches(t, template, strings.NewReader(yamlStr))
	})

	t.Run("Regex", func(t *testing.T) {
		yamlStr := `
title: "Note 123"
tags:
  - tag1
  - tag2
`
		template := map[string]any{
			"title": regexp.MustCompile(`Note \d+`),
			"tags":  []any{"tag1", "tag2"},
		}
		testutil.AssertYAMLMatches(t, template, strings.NewReader(yamlStr))
	})

	t.Run("NestedMap", func(t *testing.T) {
		yamlStr := `
meta:
  author: "Julien"
  year: 2024
`
		template := map[string]any{
			"meta": map[string]any{
				"author": "Julien",
				"year":   2024,
			},
		}
		testutil.AssertYAMLMatches(t, template, strings.NewReader(yamlStr))
	})

	t.Run("SliceOfMaps", func(t *testing.T) {
		yamlStr := `
items:
  - name: "A"
    value: 1
  - name: "B"
    value: 2
`
		template := map[string]any{
			"items": []any{
				map[string]any{"name": "A", "value": 1},
				map[string]any{"name": "B", "value": 2},
			},
		}
		testutil.AssertYAMLMatches(t, template, strings.NewReader(yamlStr))
	})

	t.Run("MissingKeyFails", func(t *testing.T) {
		yamlStr := `
title: "Test"
`
		template := map[string]any{
			"title":  "Test",
			"author": "Julien",
		}
		err := testutil.CompareYAMLReader(template, strings.NewReader(yamlStr))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing key in YAML: author")
	})

	t.Run("UnexpectedKeyFails", func(t *testing.T) {
		yamlStr := `
title: "Test"
extra: "oops"
`
		template := map[string]any{
			"title": "Test",
		}
		err := testutil.CompareYAMLReader(template, strings.NewReader(yamlStr))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected key in YAML: extra")
	})

	t.Run("RegexpMismatchFails", func(t *testing.T) {
		yamlStr := `
title: "Not a number"
`
		template := map[string]any{
			"title": regexp.MustCompile(`\d+`),
		}
		err := testutil.CompareYAMLReader(template, strings.NewReader(yamlStr))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match regexp")
	})
}
