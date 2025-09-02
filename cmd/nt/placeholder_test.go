package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []Placeholder
	}{
		{
			name:     "No placeholders",
			url:      "https://github.com/julien-sobczak/the-notewriter/",
			expected: []Placeholder{},
		},
		{
			name: "Single simple placeholder",
			url:  "https://github.com/julien-sobczak/the-notewriter/${page}",
			expected: []Placeholder{
				{
					Name:          "page",
					FullMatch:     "${page}",
					AllowedValues: nil,
					HasMore:       false,
				},
			},
		},
		{
			name: "Placeholder with allowed values",
			url:  "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions]}",
			expected: []Placeholder{
				{
					Name:          "page",
					FullMatch:     "${page:[issues,pulls,actions]}",
					AllowedValues: []string{"issues", "pulls", "actions"},
					HasMore:       false,
				},
			},
		},
		{
			name: "Placeholder with suggestions (has more)",
			url:  "https://github.com/julien-sobczak/the-notewriter/${page:[issues,pulls,actions,...]}",
			expected: []Placeholder{
				{
					Name:          "page",
					FullMatch:     "${page:[issues,pulls,actions,...]}",
					AllowedValues: []string{"issues", "pulls", "actions"},
					HasMore:       true,
				},
			},
		},
		{
			name: "Multiple placeholders",
			url:  "https://github.com/${user}/${repo}/${page:[issues,pulls]}",
			expected: []Placeholder{
				{
					Name:          "user",
					FullMatch:     "${user}",
					AllowedValues: nil,
					HasMore:       false,
				},
				{
					Name:          "repo",
					FullMatch:     "${repo}",
					AllowedValues: nil,
					HasMore:       false,
				},
				{
					Name:          "page",
					FullMatch:     "${page:[issues,pulls]}",
					AllowedValues: []string{"issues", "pulls"},
					HasMore:       false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePlaceholders(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
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
			result := expandURL(tt.url, tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPlaceholderType(t *testing.T) {
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
				HasMore:       false,
			},
			expected: "input",
		},
		{
			name: "Select type",
			placeholder: Placeholder{
				Name:          "page",
				AllowedValues: []string{"issues", "pulls"},
				HasMore:       false,
			},
			expected: "select",
		},
		{
			name: "Autocomplete type",
			placeholder: Placeholder{
				Name:          "page",
				AllowedValues: []string{"issues", "pulls"},
				HasMore:       true,
			},
			expected: "autocomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.placeholder.getPlaceholderType()
			assert.Equal(t, tt.expected, result)
		})
	}
}