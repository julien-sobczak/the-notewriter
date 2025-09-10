package core_test

import (
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOCFilePreprocessor(t *testing.T) {
	// Create a test repository 
	tr := core.NewTestRepository(t)

	// Create test markdown content with toc tag
	tr.WriteFile("test_toc.md", `---
tags: "toc"
---
# Test File

## Note: First Note
This is the first note.

### Note: Sub Note
This is a sub-note.

### Flashcard: Test Card

What is testing?

---

Validation of functionality

## Resources
This is a section with child notes.

### Quote: Test Quote

> Testing is important.

## Empty Section
This section has no child notes.
`)

	// Parse the file which should trigger the TOC preprocessor
	parsedFile := tr.ParseFile("test_toc.md")
	require.NotNil(t, parsedFile)

	// Should have a TOC note plus the actual content notes
	// Expected: 1 TOC + 4 content notes (First Note, Sub Note, Test Card, Test Quote) = 5 total
	assert.True(t, len(parsedFile.Notes) >= 5, "Should have at least 5 notes including TOC")

	// First note should be the generated TOC
	tocNote := parsedFile.Notes[0]
	assert.Equal(t, "Table of Content", tocNote.Title.String())
	assert.Equal(t, "Table of Content", tocNote.ShortTitle.String()) 
	assert.Equal(t, 0, tocNote.Line) // Generated note has line number 0

	// Check that TOC content contains expected wikilinks
	tocBody := tocNote.Body.String()
	assert.Contains(t, tocBody, "[[#Note: First Note]]")
	assert.Contains(t, tocBody, "[[#Note: Sub Note]]")
	assert.Contains(t, tocBody, "[[#Flashcard: Test Card]]")
	assert.Contains(t, tocBody, "[[#Quote: Test Quote]]")
	assert.Contains(t, tocBody, "Resources") // Untyped section with children

	// Should not contain empty section (no child notes)
	assert.NotContains(t, tocBody, "Empty Section")

	// TOC should have proper indentation
	lines := strings.Split(strings.TrimSpace(tocBody), "\n")
	var foundIndentedNote bool
	for _, line := range lines {
		if strings.Contains(line, "[[#Note: Sub Note]]") && strings.HasPrefix(line, "  ") {
			foundIndentedNote = true
			break
		}
	}
	assert.True(t, foundIndentedNote, "Sub Note should be indented in TOC")
}