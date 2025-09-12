package core_test

import (
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryExtraction(t *testing.T) {
	testutil.FreezeNow(t)

	t.Run("Memory from list item attributes", func(t *testing.T) {
		// Create test repository with custom config that includes memory attribute
		tr := core.NewTestRepository(t, core.WithConfigFileOverride(func(c *core.ConfigFile) {
			// Add read_date attribute with memory: true
			c.Attributes["read_date"] = &core.ConfigAttribute{
				Name:     "read_date",
				Type:     "date",
				Format:   "yyyy-mm-dd",
				Inherit:  core.BoolPointer(true),
				Memory:   core.BoolPointer(true),
			}
		}))

		// Create test note with memory attributes in list items
		rdq := string(rune(0x201D)) // RIGHT DOUBLE QUOTATION MARK
		content := "# My Reading List\n\n## ReadingList: Books I've Read\n\n" +
			"* _The Alchemist_ by Paulo Coelho ★★★★★ " + rdq + "@read_date: 2025-03-21" + rdq + "\n" +
			"* _Educated_ by Tara Westover ★★★★★ " + rdq + "@read_date: 2025-03-29" + rdq + "\n" +
			"* _Siddhartha_ by Hermann Hesse ★★★★☆ " + rdq + "@read_date: 2025-04-01" + rdq + "\n\n" +
			"These are some books I've enjoyed reading."
		tr.WriteFile("books.md", content)

		// Parse the file
		md, err := markdown.ParseFile(filepath.Join(tr.Root, "books.md"))
		require.NoError(t, err)

		file, err := core.ParseFile(md, nil)
		require.NoError(t, err)

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
			{"_The Alchemist_ by Paulo Coelho ★★★★★", "2025-03-21"},
			{"_Educated_ by Tara Westover ★★★★★", "2025-03-29"},
			{"_Siddhartha_ by Hermann Hesse ★★★★☆", "2025-04-01"},
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
				Name:     "published_date",
				Type:     "date",
				Format:   "yyyy-mm-dd",
				Inherit:  core.BoolPointer(true),
				Memory:   core.BoolPointer(true),
			}
		}))

		// Create test note with memory attribute at note level
		rdq := string(rune(0x201D)) // RIGHT DOUBLE QUOTATION MARK
		content := "# My Articles\n\n## Note: First Blog Post\n" + rdq + "@published_date: 2025-01-15" + rdq + "\n\nThis was my first blog post about testing."
		tr.WriteFile("article.md", content)

		// Parse the file
		md, err := markdown.ParseFile(filepath.Join(tr.Root, "article.md"))
		require.NoError(t, err)

		file, err := core.ParseFile(md, nil)
		require.NoError(t, err)

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