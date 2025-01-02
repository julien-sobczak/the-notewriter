package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestContainsMarkdownSection(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		sectionTitle string
		expected     bool
	}{
		{
			name:         "Section exists",
			fileContent:  "## Section 1\nContent\n## Section 2\nContent",
			sectionTitle: "Section 1",
			expected:     true,
		},
		{
			name:         "Section does not exist",
			fileContent:  "## Section 1\nContent\n## Section 2\nContent",
			sectionTitle: "Section 3",
			expected:     false,
		},
		{
			name:         "Empty file",
			fileContent:  "",
			sectionTitle: "Section 1",
			expected:     false,
		},
		{
			name:         "Section title is a substring",
			fileContent:  "## Section 123\nContent",
			sectionTitle: "Section 1",
			expected:     true,
		},
		{
			name:         "Text exists but not a section",
			fileContent:  "## Section 1\nContent",
			sectionTitle: "Content",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpfile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			// Write the test content to the file
			if _, err := tmpfile.WriteString(tt.fileContent); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Call the function
			actual, err := ContainsMarkdownSection(tmpfile.Name(), tt.sectionTitle)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}


