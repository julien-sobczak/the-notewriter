package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReminder(t *testing.T) {
	root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(`
## TODO: Backlog

* [ ] Test ”#reminder-2085-09”
`))
	oid.UseSequence(t)
	AssertNoReminders(t)
	c := FreezeNow(t)
	createdAt := clock.Now()

	dummyPackFile := DummyPackFile()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "project.md")
	require.NoError(t, err)
	file, err := NewFile(dummyPackFile, parsedFile)
	require.NoError(t, err)
	require.NoError(t, file.Save())
	parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
	require.True(t, ok)
	note, err := NewNote(dummyPackFile, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, note.Save())

	// Create
	parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
	require.True(t, ok)
	reminder, err := NewReminder(dummyPackFile, note, parsedReminder)
	require.NoError(t, err)

	// Check all fields
	assert.NotNil(t, reminder.OID)
	assert.Equal(t, note.FileOID, reminder.FileOID)
	assert.Equal(t, note.OID, reminder.NoteOID)
	assert.Equal(t, note.RelativePath, reminder.RelativePath)
	assert.Equal(t, "Test", reminder.Description.String())
	assert.Equal(t, "#reminder-2085-09", reminder.Tag)
	assert.Empty(t, reminder.LastPerformedAt)
	assert.Equal(t, HumanTime(t, "2085-09-01 00:00:00"), reminder.NextPerformedAt)
	assert.Equal(t, clock.Now(), reminder.CreatedAt)
	assert.Equal(t, clock.Now(), reminder.UpdatedAt)
	assert.Empty(t, reminder.DeletedAt)
	assert.Empty(t, reminder.LastIndexedAt)

	// Save
	require.NoError(t, reminder.Save())
	require.Equal(t, 1, MustCountReminders(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadReminderByOID(reminder.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, reminder.OID, actual.OID)
	assert.Equal(t, reminder.FileOID, actual.FileOID)
	assert.Equal(t, reminder.NoteOID, actual.NoteOID)
	assert.Equal(t, reminder.RelativePath, actual.RelativePath)
	assert.Equal(t, reminder.Description, actual.Description)
	assert.Equal(t, reminder.Tag, actual.Tag)
	assert.Empty(t, reminder.LastPerformedAt)
	assert.Equal(t, HumanTime(t, "2085-09-01 00:00:00"), reminder.NextPerformedAt)
	assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, clock.Now(), actual.LastIndexedAt, 1*time.Second)
	assert.Empty(t, actual.DeletedAt)

	// Force update
	updatedAt := c.FastForward(10 * time.Minute)
	ReplaceLine(t, filepath.Join(root, "project.md"), 4,
		"* [ ] Test `#reminder-2085-09`",
		"* [ ] Test `#reminder-2050-01`")
	parsedFile, err = ParseFileFromRelativePath(root, "project.md")
	require.NoError(t, err)
	parsedNote, ok = parsedFile.FindNoteByTitle("TODO: Backlog")
	require.True(t, ok)
	newNote, err := NewOrExistingNote(dummyPackFile, file, parsedNote)
	require.NoError(t, err)
	require.NoError(t, newNote.Save())
	parsedReminder, ok = parsedNote.FindReminderByTag("#reminder-2050-01")
	require.True(t, ok)
	newReminder, err := NewOrExistingReminder(dummyPackFile, newNote, parsedReminder)
	require.NoError(t, err)
	require.NoError(t, newReminder.Save())

	// Compare
	assert.Equal(t, reminder.OID, newReminder.OID) // Must have found the previous one
	assert.Equal(t, "#reminder-2050-01", newReminder.Tag)

	// Retrieve
	updatedReminder, err := CurrentRepository().LoadReminderByOID(reminder.OID)
	require.NoError(t, err)
	// Timestamps must have changed
	assert.WithinDuration(t, createdAt, updatedReminder.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedReminder.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, updatedAt, updatedReminder.LastIndexedAt, 1*time.Second)

	// Delete
	require.NoError(t, reminder.Delete())
	assert.Equal(t, clock.Now(), reminder.DeletedAt)

	AssertNoReminders(t)
}

func TestReminderFormats(t *testing.T) {
	oid.UseFixed(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(`
## TODO: Backlog

* [ ] Test ”#reminder-2085-09”
`))

	dummyPackFile := DummyPackFile()

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "project.md")
	require.NoError(t, err)
	file, err := NewFile(dummyPackFile, parsedFile)
	require.NoError(t, err)

	// Init the reminder
	parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
	require.True(t, ok)
	note, err := NewNote(dummyPackFile, file, parsedNote)
	require.NoError(t, err)
	parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
	require.True(t, ok)
	reminder, err := NewReminder(dummyPackFile, note, parsedReminder)
	require.NoError(t, err)

	t.Run("ToYAML", func(t *testing.T) {
		actual := reminder.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: project.md
description: Test
tag: '#reminder-2085-09'
last_performed_at: 0001-01-01T00:00:00Z
next_performed_at: 2085-09-01T00:00:00Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := reminder.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "file_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "note_oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "relative_path": "project.md",
  "description": "Test",
  "tag": "#reminder-2085-09",
  "last_performed_at": "0001-01-01T00:00:00Z",
  "next_performed_at": "2085-09-01T00:00:00Z",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "deleted_at": "0001-01-01T00:00:00Z"
}
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToMarkdown", func(t *testing.T) {
		actual := reminder.ToMarkdown()
		expected := UnescapeTestContent("Test `#reminder-2085-09`")
		assert.Equal(t, expected, actual)
	})

}

func TestEvaluateTimeExpression(t *testing.T) {
	clock.FreezeAt(time.Date(2023, time.Month(7), 1, 1, 12, 30, 0, time.UTC))
	defer clock.Unfreeze()

	var tests = []struct {
		name     string    // name
		expr     string    // input
		expected time.Time // output
	}{

		{
			name:     "static date",
			expr:     "2023-02-01",
			expected: time.Date(2023, time.Month(2), 1, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "static date every year",
			expr:     "every-${year}-02-01",
			expected: time.Date(2024, time.Month(2), 1, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "static date every even year",
			expr:     "${even-year}-02-01",
			expected: time.Date(2025, time.Month(2), 1, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "static date every odd year",
			expr:     "${odd-year}-02-01",
			expected: time.Date(2024, time.Month(2), 1, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "every start of month in 2025",
			expr:     "every-2025-${month}-02",
			expected: time.Date(2025, time.Month(1), 2, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "odd month with unspecified day",
			expr:     "every-2025-${odd-month}",
			expected: time.Date(2025, time.Month(2), 1, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "every day",
			expr:     "every-${day}",
			expected: time.Date(2023, time.Month(7), 2, 0, 0, 0, 0, time.UTC),
		},

		{
			name:     "every tuesday",
			expr:     "every-${tuesday}",
			expected: time.Date(2023, time.Month(7), 4, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := EvaluateTimeExpression(tt.expr)
			require.NoError(t, err)
			assert.EqualValues(t, tt.expected, actual)
		})
	}
}
