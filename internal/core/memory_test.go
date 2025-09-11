package core_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMemoryExtraction(t *testing.T) {
	testutil.FreezeNow(t)
	
	tr := core.NewTestRepository(t)

	// Create custom config with memory attribute
	tr.WriteFile(".nt/config.jsonnet", `
local nt = import 'nt.libsonnet';

{
    core: {
        medias: {
            command: "ffmpeg",
            parallel: 1,
            preset: "ultrafast",
        },
    },

    attributes: nt.DefaultAttributes + {
        read_date: {
            name: "read_date",
            type: "date",
            format: "yyyy-mm-dd",
            inherit: true,
            memory: true,
        },
    },
    types: nt.DefaultTypes,
}`)

	// Create test note
	tr.WriteFile("test_note.md", `# Books I've Read

## ReadingList: My Reading Journey

* _The Alchemist_ by Paulo Coelho 👍 ★★★★★ `+"`@read_date: 2025-03-21`"+`
* _Educated_ by Tara Westover 👍 ★★★★★ `+"`@read_date: 2025-03-29`"+`
* _Siddhartha_ by Hermann Hesse ★★★★☆ `+"`@read_date: 2025-04-01`"+`

These are some great books I've read this year.`)

	// Parse the file
	md, err := markdown.ParseFile(filepath.Join(tr.Root, "test_note.md"))
	require.NoError(t, err)

	file, err := core.ParseFile(md, nil)
	require.NoError(t, err)

	fmt.Printf("Parsed file: %s\n", file.RelativePath)
	fmt.Printf("Number of notes: %d\n", len(file.Notes))

	for i, note := range file.Notes {
		fmt.Printf("\nNote %d: %s\n", i+1, note.ShortTitle.String())
		fmt.Printf("  Attributes: %+v\n", note.Attributes)
		fmt.Printf("  Number of memories: %d\n", len(note.Memories))
		
		for j, memory := range note.Memories {
			fmt.Printf("    Memory %d: %s (occurred: %s)\n", j+1, memory.Text.String(), memory.OccurredAt.Format("2006-01-02"))
		}

		// Check list items
		if note.Items != nil {
			fmt.Printf("  Number of list items: %d\n", len(note.Items.Children))
			for j, item := range note.Items.Children {
				fmt.Printf("    Item %d: %s\n", j+1, item.Text.String())
				fmt.Printf("      Attributes: %+v\n", item.Attributes)
			}
		}
	}

	// Check that we have at least one note with memories
	require.Greater(t, len(file.Notes), 0, "Should have at least one note")
	
	// Find the note with items and check for memories
	var foundMemories int
	for _, note := range file.Notes {
		foundMemories += len(note.Memories)
	}
	
	require.Greater(t, foundMemories, 0, "Should have extracted memories from read_date attributes")
}