package core_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory(t *testing.T) {
	testutil.FreezeNow(t)

	t.Run("Interfaces", func(t *testing.T) {
		var memory core.Memory
		assert.Implements(t, (*core.StatefulObject)(nil), &memory)
		assert.Implements(t, (*core.Packable)(nil), &memory)
	})

	t.Run("Memory from list item attributes", func(t *testing.T) {
		// Create test repository with custom config that includes memory attribute
		tr := core.NewTestRepository(t, core.WithConfigFileOverride(func(c *core.ConfigFile) {
			// Add read_date attribute with memory: true
			c.Attributes["read_date"] = &core.ConfigAttribute{
				Name:    "read_date",
				Type:    "date",
				Format:  "yyyy-mm-dd",
				Inherit: core.BoolPointer(true),
				Memory:  core.BoolPointer(true),
			}
		}))

		// Create test note with memory attributes in list items
		tr.WriteFile("books.md", `# My Reading List

## ReadingList: Books I've Read

* _The Alchemist_ by Paulo Coelho ★★★★★ ‛@read_date: 2025-03-21‛
* _Educated_ by Tara Westover ★★★★★ ‛@read_date: 2025-03-29‛
* _Siddhartha_ by Hermann Hesse ★★★★☆ ‛@read_date: 2025-04-01‛

These are some books I've enjoyed reading.
`)

		// Parse the file
		file := tr.ParseFile("books.md")

		// Verify we parsed the file correctly
		require.Len(t, file.Notes, 1)
		note := file.Notes[0]
		assert.Equal(t, "Books I've Read", note.ShortTitle.String())

		// Check that memories were extracted
		assert.Len(t, note.Memories, 3, "Should have extracted 3 memories from the read_date attributes")

		// Verify memory contents
		expectedMemories := []struct {
			text string
			date string
		}{
			{"Books I've Read / _The Alchemist_ by Paulo Coelho ★★★★★", "2025-03-21"},
			{"Books I've Read / _Educated_ by Tara Westover ★★★★★", "2025-03-29"},
			{"Books I've Read / _Siddhartha_ by Hermann Hesse ★★★★☆", "2025-04-01"},
		}

		for i, expected := range expectedMemories {
			assert.Equal(t, expected.text, note.Memories[i].Text.String())
			assert.Equal(t, expected.date, note.Memories[i].OccurredAt.Format("2006-01-02"))
		}
	})

	t.Run("Memory from note-level attributes", func(t *testing.T) {
		// Create test repository with custom config that includes memory attribute
		tr := core.NewTestRepository(t, core.WithConfigFileOverride(func(c *core.ConfigFile) {
			// Add published_date attribute with memory: true
			c.Attributes["published_date"] = &core.ConfigAttribute{
				Name:    "published_date",
				Type:    "date",
				Format:  "yyyy-mm-dd",
				Inherit: core.BoolPointer(true),
				Memory:  core.BoolPointer(true),
			}
		}))

		// Create test note with memory attribute at note level
		tr.WriteFile("article.md", `# My Articles

## Note: First Blog Post
‛@published_date: 2025-01-15‛

This was my first blog post about testing.
`)

		// Parse the file
		file := tr.ParseFile("article.md")

		// Verify we parsed the file correctly
		require.Len(t, file.Notes, 1)
		note := file.Notes[0]

		// Check that memory was extracted
		assert.Len(t, note.Memories, 1, "Should have extracted 1 memory from the published_date attribute")

		// Verify memory content only if memory was extracted
		if len(note.Memories) > 0 {
			memory := note.Memories[0]
			assert.Equal(t, "First Blog Post", memory.Text.String())
			assert.Equal(t, "2025-01-15", memory.OccurredAt.Format("2006-01-02"))
		}
	})
}
