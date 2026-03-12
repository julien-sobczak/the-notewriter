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

{{ randomListItem "journaling#List: Affirmations" }}

# 🎯 My BIG thing for today

_Your Answer_
`,
						},
						{
							Name: "Shutdown Routine",
							Template: `# ❓ How was my day? Why?

{{ randomListItem "journaling#List: Prompts" }}
_Your Answer_

# 📋 3 tasks to complete tomorrow:

* [ ] _Task 1_
* [ ] _Task 2_
* [ ] _Task 3_
`,
						},
					},
				},
			}
		}),
	)

	// Set up note files with list items for randomListItem function
	tr.WriteFile("journaling.md", `# Journaling

## List: Affirmations

* All I need is within me right now.
* I am grateful for this new day.
* I have the power to create positive change.

## List: Prompts

* How can I make today meaningful?
* What am I most excited about this week?
* What is one thing I can do today to move closer to my goals?
`)

	_, err := CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	err = CurrentRepository().Commit(false)
	require.NoError(t, err)

	journal := CurrentConfigFile().Journals[0]
	morningRoutine := &journal.Routines[0]
	shutdownRoutine := &journal.Routines[1]

	// --- Scenario 1: Morning Routine creates the journal file ---

	morningContent, err := GenerateRoutineContent(morningRoutine)
	require.NoError(t, err)
	// randomListItem should have been replaced by a list item
	assert.NotContains(t, morningContent, `{{ randomListItem`)

	absPath1, err := AppendRoutineToJournal(journal, morningRoutine, morningContent)
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

	shutdownContent, err := GenerateRoutineContent(shutdownRoutine)
	require.NoError(t, err)
	// randomListItem should have been replaced by a list item
	assert.NotContains(t, shutdownContent, `{{ randomListItem`)

	absPath2, err := AppendRoutineToJournal(journal, shutdownRoutine, shutdownContent)
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

	morningContentDay2, err := GenerateRoutineContent(morningRoutine)
	require.NoError(t, err)

	absPath3, err := AppendRoutineToJournal(journal, morningRoutine, morningContentDay2)
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
