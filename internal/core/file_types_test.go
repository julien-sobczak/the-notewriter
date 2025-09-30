package core

import (
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestFileTypeWithProcessors(t *testing.T) {
tr := NewTestRepository(t)

// Configure a custom file type with a processor
tr.WriteFile(".nt/config.jsonnet", `
local nt = import 'nt.libsonnet';
{
attributes: nt.DefaultAttributes,
noteTypes: nt.DefaultNoteTypes,
fileTypes: {
ReadingNotes: {
name: "ReadingNotes",
pattern: "(?i)^Reading:\\s*(.*)$",
processors: ["toc"],
},
},
}
`)
CurrentConfig().Reload()

// Create a file matching the file type
tr.WriteFile("reading-book.md", `
---
title: "Reading: My Book"
tags: []
---

# Reading: My Book

## Note: Chapter 1

Content for chapter 1.

## Note: Chapter 2

Content for chapter 2.
`)

file := tr.ParseFile("reading-book.md")
require.NotNil(t, file)

// Verify that the file matches the file type
fileType, ok := CurrentConfigFile().MatchFileType(file.Title.String())
assert.True(t, ok)
require.NotNil(t, fileType)
assert.Equal(t, "ReadingNotes", fileType.Name)
assert.Equal(t, []string{"toc"}, fileType.Processors)

// Verify notes were parsed (toc processor adds a "Table of Content" note)
assert.GreaterOrEqual(t, len(file.Notes), 2)

// Find the chapter notes (excluding the TOC note)
var chapterNotes []*ParsedNote
for _, note := range file.Notes {
	if note.ShortTitle.String() != "Table of Content" {
		chapterNotes = append(chapterNotes, note)
	}
}
assert.Len(t, chapterNotes, 2)
assert.Equal(t, "Chapter 1", chapterNotes[0].ShortTitle.String())
assert.Equal(t, "Chapter 2", chapterNotes[1].ShortTitle.String())
}

func TestFileTypeBackwardCompatibility(t *testing.T) {
tr := NewTestRepository(t)

// Test that old "types" field still works
tr.WriteFile(".nt/config.jsonnet", `
local nt = import 'nt.libsonnet';
{
attributes: nt.DefaultAttributes,
types: nt.DefaultTypes,  // Using old field name
}
`)
CurrentConfig().Reload()

tr.WriteFile("test.md", `
---
title: "Test"
---

# Test

## Note: Simple Note

Test content.
`)

file := tr.ParseFile("test.md")
require.NotNil(t, file)
assert.Len(t, file.Notes, 1)
assert.Equal(t, "Simple Note", file.Notes[0].ShortTitle.String())
}
