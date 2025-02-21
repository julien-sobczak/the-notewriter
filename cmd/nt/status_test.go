package main

import (
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		result   *core.StatusResult
		expected string
	}{
		{
			name: "no changes",
			result: &core.StatusResult{
				ChangesStaged:    core.FileStatuses{},
				ChangesNotStaged: core.FileStatuses{},
			},
			expected: "",
		},
		{
			name: "staged changes",
			result: &core.StatusResult{
				ChangesStaged: core.FileStatuses{
					{Status: "modified", RelativePath: "file1.txt", ObjectsModified: 2},
					{Status: "deleted", RelativePath: "file2.txt", ObjectsDeleted: 1},
					{Status: "added", RelativePath: "file3.txt", ObjectsAdded: 3},
				},
				ChangesNotStaged: core.FileStatuses{},
			},
			expected: `
Changes to be committed:
  (use "nt restore..." to unstage)
    modified: file1.txt (2)
     deleted: file2.txt (-1)
       added: file3.txt (+3)
`,
		},
		{
			name: "unstaged changes",
			result: &core.StatusResult{
				ChangesStaged: core.FileStatuses{},
				ChangesNotStaged: core.FileStatuses{
					{Status: "modified", RelativePath: "file1.txt"},
					{Status: "deleted", RelativePath: "file2.txt"},
				},
			},
			expected: `
Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
    modified: file1.txt
     deleted: file2.txt
`,
		},
		{
			name: "both staged and unstaged changes",
			result: &core.StatusResult{
				ChangesStaged: core.FileStatuses{
					{Status: "modified", RelativePath: "file1.txt", ObjectsModified: 2},
				},
				ChangesNotStaged: core.FileStatuses{
					{Status: "deleted", RelativePath: "file2.txt"},
				},
			},
			expected: `
Changes to be committed:
  (use "nt restore..." to unstage)
    modified: file1.txt (2)

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
     deleted: file2.txt
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatStatus(tt.result)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}
