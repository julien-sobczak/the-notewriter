package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoto(t *testing.T) {
	tr := NewTestRepository(t, WithFreezeNow())

	tr.AssertNoGotos()

	createdAt := clock.Now()
	gotoLink := &Goto{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "project.md",
		Text:         "Golang",
		URL:          "https://go.dev/doc/",
		Title:        "",
		Name:         "go",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		IndexedAt:    createdAt,
	}

	// Save
	require.NoError(t, gotoLink.Save())
	require.Equal(t, 1, tr.CountGotos())

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadGotoByOID(gotoLink.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, gotoLink.OID, actual.OID)
	assert.Equal(t, gotoLink.PackFileOID, actual.PackFileOID)
	assert.Equal(t, gotoLink.NoteOID, actual.NoteOID)
	assert.Equal(t, gotoLink.RelativePath, actual.RelativePath)
	assert.Equal(t, gotoLink.Text, actual.Text)
	assert.Equal(t, gotoLink.URL, actual.URL)
	assert.Equal(t, gotoLink.Title, actual.Title)
	assert.Equal(t, gotoLink.Name, actual.Name)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.IndexedAt, 1*time.Second)

	// Force update
	gotoLink.Text = "Go Language"
	gotoLink.URL = "https://go.dev"
	require.NoError(t, gotoLink.Save())
	require.Equal(t, 1, tr.CountGotos())

	// ...and compare again
	actual, err = CurrentRepository().LoadGotoByOID(gotoLink.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"), actual.OID) // Must have found the previous one
	assert.Equal(t, "Go Language", actual.Text.String())
	assert.Equal(t, "https://go.dev", actual.URL)

	// Delete
	require.NoError(t, gotoLink.Delete())
	tr.AssertNoGotos()
}

func TestGotoFormats(t *testing.T) {
	testutil.FreezeOn(t, "2023-01-01 01:12:30")

	gotoLink := &Goto{
		OID:          "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID:  "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		NoteOID:      "52d02a28a961471db62c6d40d30639dafe4aba00",
		RelativePath: "go.md",
		Text:         "Golang",
		URL:          "https://go.dev/doc/",
		Title:        "",
		Name:         "go",
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		IndexedAt:    clock.Now(),
	}

	t.Run("ToYAML", func(t *testing.T) {
		actual := gotoLink.ToYAML()

		expected := text.UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
note_oid: 52d02a28a961471db62c6d40d30639dafe4aba00
relative_path: go.md
text: Golang
url: https://go.dev/doc/
title: ""
name: go
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := gotoLink.ToJSON()
		expected := text.UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "note_oid": "52d02a28a961471db62c6d40d30639dafe4aba00",
  "relative_path": "go.md",
  "text": "Golang",
  "url": "https://go.dev/doc/",
  "title": "",
  "name": "go",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := gotoLink.ToMarkdown()
		expected := text.UnescapeTestContent(`[Golang](https://go.dev/doc/)`)
		assert.Equal(t, expected, actual)
	})

}

func TestParameterizedURL(t *testing.T) {

	t.Run("Expand", func(t *testing.T) {
		tests := []struct {
			name     string
			url      ParameterizedURL
			values   map[string]string
			expected string
		}{
			{
				name:     "No placeholders",
				url:      "https://github.com/julien-sobczak/the-notewriter/",
				values:   map[string]string{},
				expected: "https://github.com/julien-sobczak/the-notewriter/",
			},
			{
				name:     "Single simple placeholder",
				url:      "https://github.com/julien-sobczak/the-notewriter/${page}",
				values:   map[string]string{"page": "issues"},
				expected: "https://github.com/julien-sobczak/the-notewriter/issues",
			},
			{
				name:     "Placeholder with allowed values",
				url:      "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions]}",
				values:   map[string]string{"page": "pulls"},
				expected: "https://github.com/julien-sobczak/the-notewriter/pulls",
			},
			{
				name:     "Multiple placeholders",
				url:      "https://github.com/${user}/${repo}/${page:[issues,pulls]}",
				values:   map[string]string{"user": "julien-sobczak", "repo": "the-notewriter", "page": "issues"},
				expected: "https://github.com/julien-sobczak/the-notewriter/issues",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.url.Expand(tt.values)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Placeholders", func(t *testing.T) {
		tests := []struct {
			name     string
			url      ParameterizedURL
			expected []Placeholder
		}{
			{
				name:     "No placeholders",
				url:      "https://github.com/julien-sobczak/the-notewriter/",
				expected: nil,
			},
			{
				name: "Single simple placeholder",
				url:  "https://github.com/julien-sobczak/the-notewriter/${page}",
				expected: []Placeholder{
					{
						Name:          "page",
						Raw:           "${page}",
						AllowedValues: nil,
						Ellipsis:      false,
					},
				},
			},
			{
				name: "Placeholder with allowed values",
				url:  "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions]}",
				expected: []Placeholder{
					{
						Name:          "page",
						Raw:           "${page:[issues,pulls,actions]}",
						AllowedValues: []string{"issues", "pulls", "actions"},
						Ellipsis:      false,
					},
				},
			},
			{
				name: "Placeholder with suggestions (has more)",
				url:  "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions,...]}",
				expected: []Placeholder{
					{
						Name:          "page",
						Raw:           "${page:[issues,pulls,actions,...]}",
						AllowedValues: []string{"issues", "pulls", "actions"},
						Ellipsis:      true,
					},
				},
			},
			{
				name: "Multiple placeholders",
				url:  "https://github.com/${user}/${repo}/${page:[issues,pulls]}",
				expected: []Placeholder{
					{
						Name:          "user",
						Raw:           "${user}",
						AllowedValues: nil,
						Ellipsis:      false,
					},
					{
						Name:          "repo",
						Raw:           "${repo}",
						AllowedValues: nil,
						Ellipsis:      false,
					},
					{
						Name:          "page",
						Raw:           "${page:[issues,pulls]}",
						AllowedValues: []string{"issues", "pulls"},
						Ellipsis:      false,
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.url.Placeholders()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Short", func(t *testing.T) {
		tests := []struct {
			name     string
			input    ParameterizedURL
			expected ParameterizedURL
		}{
			{
				name:     "No placeholders",
				input:    "https://github.com/julien-sobczak/the-notewriter/",
				expected: "https://github.com/julien-sobczak/the-notewriter/",
			},
			{
				name:     "Simple placeholder without allowed values",
				input:    "https://github.com/julien-sobczak/the-notewriter/${page}",
				expected: "https://github.com/julien-sobczak/the-notewriter/${page}",
			},
			{
				name:     "Placeholder with allowed values",
				input:    "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions]}",
				expected: "https://github.com/julien-sobczak/the-notewriter/${page}",
			},
			{
				name:     "Placeholder with allowed values and ellipsis",
				input:    "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,...]}",
				expected: "https://github.com/julien-sobczak/the-notewriter/${page}",
			},
			{
				name:     "Multiple placeholders with allowed values",
				input:    "https://github.com/${user}/${repo}/${page:[issues,pulls]}",
				expected: "https://github.com/${user}/${repo}/${page}",
			},
			{
				name:     "Mixed placeholders with and without allowed values",
				input:    "https://github.com/${user}/${repo}/${page:[issues,pulls]}/${section}",
				expected: "https://github.com/${user}/${repo}/${page}/${section}",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.input.Short()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("ToANSI", func(t *testing.T) {
		url := ParameterizedURL("https://github.com/${user}/${repo}/${page:[issues,pulls,...]}")
		actual := url.ToANSI()
		require.Equal(t, "https://github.com/\033[31m${user}\033[0m/\033[31m${repo}\033[0m/\033[31m${page:[issues,pulls,...]}\033[0m", actual)
	})
}

func TestPlaceholder(t *testing.T) {

	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name        string
			placeholder Placeholder
			expected    string
		}{
			{
				name: "Input type",
				placeholder: Placeholder{
					Name:          "page",
					AllowedValues: nil,
					Ellipsis:      false,
				},
				expected: "page (enter any value)",
			},
			{
				name: "Select type",
				placeholder: Placeholder{
					Name:          "page",
					AllowedValues: []string{"issues", "pulls"},
					Ellipsis:      false,
				},
				expected: "page (choose from: issues, pulls)",
			},
			{
				name: "Autocomplete type",
				placeholder: Placeholder{
					Name:          "page",
					AllowedValues: []string{"issues", "pulls"},
					Ellipsis:      true,
				},
				expected: "page (suggestions: issues, pulls, or enter custom value)",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.placeholder.String()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

}

/* Test Helpers */

func MustFindGotoByName(t *testing.T, name string) *Goto {
	obj, err := CurrentRepository().FindGotoByName(name)
	require.NoError(t, err)
	require.NotNil(t, obj)
	return obj
}
