package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/julien-sobczak/the-notewriter/internal/core"
)

func TestEvaluateJournalPath(t *testing.T) {
	NewTestRepository(t, WithFreezeOn("2026-02-26"))

	t.Run("path with date functions", func(t *testing.T) {
		result, err := EvaluateJournalPath("journal/{{ year }}/{{ year }}-{{ month }}-{{ day }}.md")
		require.NoError(t, err)
		assert.Equal(t, "journal/2026/2026-02-26.md", result)
	})

	t.Run("date function", func(t *testing.T) {
		result, err := EvaluateJournalPath("Journal: {{ date }}")
		require.NoError(t, err)
		assert.Equal(t, "Journal: 2026-02-26", result)
	})

	t.Run("plain string without template actions", func(t *testing.T) {
		result, err := EvaluateJournalPath("plain text")
		require.NoError(t, err)
		assert.Equal(t, "plain text", result)
	})
}

func TestGenerateRoutineContent(t *testing.T) {
	NewTestRepository(t)

	t.Run("plain template without functions", func(t *testing.T) {
		routine := &ConfigRoutine{
			Name:     "Test",
			Template: "# Heading\n\n_Your Answer_\n",
		}
		content, err := GenerateRoutineContent(routine)
		require.NoError(t, err)
		assert.Equal(t, "# Heading\n\n_Your Answer_\n", content)
	})
}

func TestShiftHeadings(t *testing.T) {
	NewTestRepository(t, WithFreezeOn("2026-02-26"))

	journal := &ConfigJournal{
		Name:           "Test Journal",
		Path:           "journal/{{ year }}-{{ month }}-{{ day }}.md",
		DefaultContent: "# Journal {{ date }}",
		Routines: []ConfigRoutine{
			{
				Name: "Morning Routine",
				Template: `# 💪 Affirmation

_Your Answer_

# 😘 Gratitude

* _Your Answer_
`,
			},
			{
				Name: "Shutdown Routine",
				Template: `# ❓ How was my day?

_Your Answer_
`,
			},
		},
	}

	t.Run("headings are shifted to level 3", func(t *testing.T) {
		// Generate and append a routine
		content := `# 💪 Affirmation

_Your Answer_

# 😘 Gratitude

* _Your Answer_`
		morningRoutine := &journal.Routines[0]
		absPath, err := AppendRoutineToJournal(journal, morningRoutine, content)
		require.NoError(t, err)

		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, `# Journal 2026-02-26

## Morning Routine

### 💪 Affirmation

_Your Answer_

### 😘 Gratitude

* _Your Answer_

`, string(data))
	})

	t.Run("second append uses same file", func(t *testing.T) {
		content := `# ❓ How was my day?

_Your Answer_`
		shutdownRoutine := &journal.Routines[1]
		absPath, err := AppendRoutineToJournal(journal, shutdownRoutine, content)
		require.NoError(t, err)

		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, `# Journal 2026-02-26

## Morning Routine

### 💪 Affirmation

_Your Answer_

### 😘 Gratitude

* _Your Answer_


## Shutdown Routine

### ❓ How was my day?

_Your Answer_

`, string(data))
	})
}

func TestAppendRoutineToJournal_NewDay(t *testing.T) {
	tr := NewTestRepository(t, WithFreezeOn("2026-02-26"))

	journal := &ConfigJournal{
		Name:           "Test Journal",
		Path:           "journal/{{ year }}-{{ month }}-{{ day }}.md",
		DefaultContent: "# Journal {{ date }}",
		Routines: []ConfigRoutine{
			{Name: "Morning Routine", Template: "# Task\n\n_Your Answer_\n"},
		},
	}

	morningRoutine := &journal.Routines[0]
	content := `# Task

_Your Answer_`

	// Day 1
	absPath1, err := AppendRoutineToJournal(journal, morningRoutine, content)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(absPath1, "2026-02-26.md"))
	require.FileExists(t, absPath1)

	// Advance clock to the next day
	tr.FastForward(24 * time.Hour)

	// Day 2 should create a new file
	absPath2, err := AppendRoutineToJournal(journal, morningRoutine, content)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(absPath2, "2026-02-27.md"))
	require.FileExists(t, absPath2)
	assert.NotEqual(t, absPath1, absPath2)

	// Verify the new file has its own defaultContent header
	data, err := os.ReadFile(absPath2)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Journal 2026-02-27")

	// Verify the day1 file has journal's relative path under root
	root := CurrentConfig().RootDirectory
	assert.Equal(t, filepath.Join(root, "journal/2026-02-26.md"), absPath1)
	assert.Equal(t, filepath.Join(root, "journal/2026-02-27.md"), absPath2)
}

func TestProcessEditedMarkdown(t *testing.T) {
	t.Run("no ephemeral sections", func(t *testing.T) {
		content := `# 💪 Affirmation

Some affirmation text

# 🎯 My BIG thing for today

_Your Answer_`

		result, err := ProcessEditedMarkdown(content)
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("strip section with ephemeral shorthand", func(t *testing.T) {
		content := `# 💪 Affirmation

Some affirmation text

# 😘 🗑️ Gratitude Journal

3 things I appreciate:

* _Thing 1_
* _Thing 2_
* _Thing 3_

# 🎯 My BIG thing for today

_Your Answer_`

		result, err := ProcessEditedMarkdown(content)
		require.NoError(t, err)
		assert.Contains(t, result, "# 💪 Affirmation")
		assert.Contains(t, result, "# 🎯 My BIG thing for today")
		assert.NotContains(t, result, "# 😘 🗑️ Gratitude Journal")
		assert.NotContains(t, result, "_Thing 1_")
	})

	t.Run("strip section with ephemeral tag name", func(t *testing.T) {
		content := `# 💪 Affirmation

Some affirmation text

# 😘 #ephemeral Gratitude Journal

3 things I appreciate:

* _Thing 1_

# 🎯 My BIG thing for today

_Your Answer_`

		result, err := ProcessEditedMarkdown(content)
		require.NoError(t, err)
		assert.Contains(t, result, "# 💪 Affirmation")
		assert.Contains(t, result, "# 🎯 My BIG thing for today")
		assert.NotContains(t, result, "Gratitude Journal")
		assert.NotContains(t, result, "_Thing 1_")
	})

	t.Run("empty content", func(t *testing.T) {
		result, err := ProcessEditedMarkdown("")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}
