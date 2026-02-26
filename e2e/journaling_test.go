package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournaling(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeOn("2026-02-26"),
		WithConfigFileOverride(func(c *ConfigFile) {
			c.Journals = ConfigJournals{
				{
					Name:           "My Diary",
					Path:           "journal/{{ year }}/{{ year }}-{{ month }}-{{ day }}.md",
					DefaultContent: "# Journal {{ date }}",
					Routines: []ConfigRoutine{
						{
							Name: "Morning Routine",
							Template: `# 💪 Affirmation

{{ input }}

# 🎯 My BIG thing for today

{{ input }}
`,
						},
						{
							Name: "Shutdown Routine",
							Template: `# ❓ How was my day? Why?

{{ input }}

# 📋 3 tasks to complete tomorrow:

* [ ] {{ input }}
* [ ] {{ input }}
* [ ] {{ input }}
`,
						},
					},
				},
			}
		}),
	)

	journal := CurrentConfigFile().Journals[0]

	// --- Scenario 1: Morning Routine creates the journal file ---

	morningContent, err := GenerateRoutineContent(&journal.Routines[0])
	require.NoError(t, err)
	assert.Contains(t, morningContent, "_Your Answer_")

	absPath1, err := AppendRoutineToJournal(journal, journal.Routines[0].Name, morningContent)
	require.NoError(t, err)

	expectedPath1 := filepath.Join(tr.Root, "journal/2026/2026-02-26.md")
	assert.Equal(t, expectedPath1, absPath1)
	require.FileExists(t, absPath1)

	data1, err := os.ReadFile(absPath1)
	require.NoError(t, err)
	content1 := string(data1)

	// Journal file should have been initialized with defaultContent
	assert.Contains(t, content1, "# Journal 2026-02-26")
	// Morning routine section at level 2
	assert.Contains(t, content1, "## Morning Routine")
	// Template headings shifted to level 3
	assert.Contains(t, content1, "### 💪 Affirmation")
	assert.Contains(t, content1, "### 🎯 My BIG thing for today")

	// --- Scenario 2: Shutdown Routine appends to the same file ---

	shutdownContent, err := GenerateRoutineContent(&journal.Routines[1])
	require.NoError(t, err)
	assert.Contains(t, shutdownContent, "_Your Answer_")

	absPath2, err := AppendRoutineToJournal(journal, journal.Routines[1].Name, shutdownContent)
	require.NoError(t, err)

	// Same file as morning routine
	assert.Equal(t, absPath1, absPath2)

	data2, err := os.ReadFile(absPath2)
	require.NoError(t, err)
	content2 := string(data2)

	// Both routines present in the same file
	assert.Contains(t, content2, "## Morning Routine")
	assert.Contains(t, content2, "## Shutdown Routine")
	assert.Contains(t, content2, "### ❓ How was my day? Why?")
	assert.Contains(t, content2, "### 📋 3 tasks to complete tomorrow:")

	// File should only be created once (defaultContent appears once)
	assert.Equal(t, 1, strings.Count(content2, "# Journal 2026-02-26"))

	// --- Scenario 3: Next day creates a new journal file ---

	tr.FastForward(24 * time.Hour)

	morningContentDay2, err := GenerateRoutineContent(&journal.Routines[0])
	require.NoError(t, err)

	absPath3, err := AppendRoutineToJournal(journal, journal.Routines[0].Name, morningContentDay2)
	require.NoError(t, err)

	expectedPath3 := filepath.Join(tr.Root, "journal/2026/2026-02-27.md")
	assert.Equal(t, expectedPath3, absPath3)
	require.FileExists(t, absPath3)
	assert.NotEqual(t, absPath1, absPath3)

	data3, err := os.ReadFile(absPath3)
	require.NoError(t, err)
	content3 := string(data3)

	// New file has its own defaultContent
	assert.Contains(t, content3, "# Journal 2026-02-27")
	assert.Contains(t, content3, "## Morning Routine")
	// Previous day's file is unchanged
	assert.NotContains(t, content3, "# Journal 2026-02-26")
}
