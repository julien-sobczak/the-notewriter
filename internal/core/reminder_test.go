package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReminder(t *testing.T) {

	content := `
## TODO: Backlog

* [ ] Test ”#reminder-2085-09”
`

	t.Run("NewReminder", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		note, err := NewNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Init the reminder
		parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
		require.True(t, ok)
		reminder, err := NewReminder(note, parsedReminder)
		require.NoError(t, err)

		// Check all fields
		assert.Equal(t, "42d74d967d9b4e989502647ac510777ca1e22f4a", reminder.OID)
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
		assert.Empty(t, reminder.LastCheckedAt)
	})

	t.Run("Save", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		AssertNoReminders(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		note, err := NewNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Init the reminder
		parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
		require.True(t, ok)
		reminder, err := NewReminder(note, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, reminder.Save())

		require.Equal(t, 1, MustCountReminders(t))

		// Reread and check the flashcard
		actual, err := CurrentRepository().LoadReminderByOID(reminder.OID)
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Equal(t, reminder.OID, actual.OID)
		assert.Equal(t, reminder.FileOID, actual.FileOID)
		assert.Equal(t, reminder.OID, actual.NoteOID)
		assert.Equal(t, reminder.RelativePath, actual.RelativePath)
		assert.Equal(t, reminder.Description, actual.Description)
		assert.Equal(t, reminder.Tag, actual.Tag)
		assert.Empty(t, reminder.LastPerformedAt)
		assert.Equal(t, HumanTime(t, "2085-09-01 00:00:00"), reminder.NextPerformedAt)
		assert.WithinDuration(t, clock.Now(), actual.CreatedAt, 1*time.Second)
		assert.WithinDuration(t, clock.Now(), actual.UpdatedAt, 1*time.Second)
		assert.WithinDuration(t, clock.Now(), actual.LastCheckedAt, 1*time.Second)
		assert.Empty(t, actual.DeletedAt)
	})

	t.Run("NewOrExistingReminder", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		AssertNoReminders(t)

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		note, err := NewNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Save the reminder
		parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
		require.True(t, ok)
		previousReminder, err := NewReminder(note, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, previousReminder.Save())

		// Edit the reminder text
		ReplaceLine(t, filepath.Join(root, "project.md"), 4,
			"* [ ] Test `#reminder-2085-09`",
			"* [ ] Test `#reminder-2050-01`")
		parsedFile, err = ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		parsedNote, ok = parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		newNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, newNote.Save())
		parsedReminder, ok = parsedNote.FindReminderByTag("#reminder-2050-01")
		require.True(t, ok)
		newReminder, err := NewOrExistingReminder(newNote, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, newReminder.Save())

		// Compare
		assert.Equal(t, previousReminder.OID, newReminder.OID) // Must have found the previous one
		assert.Equal(t, "#reminder-2050-01", newReminder.Tag)
	})

	t.Run("Update", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(content))

		c := FreezeNow(t)
		createdAt := c.Now()
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		note, err := NewNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Save the reminder
		parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
		require.True(t, ok)
		createdReminder, err := NewReminder(note, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, createdReminder.Save())

		// Edit the reminder text
		updatedAt := c.FastForward(10 * time.Minute)
		ReplaceLine(t, filepath.Join(root, "project.md"), 4,
			"* [ ] Test `#reminder-2085-09`",
			"* [ ] Test `#reminder-2050-01`")
		parsedFile, err = ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		parsedNote, ok = parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		newNote, err := NewOrExistingNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, newNote.Save())
		parsedReminder, ok = parsedNote.FindReminderByTag("#reminder-2050-01")
		require.True(t, ok)
		updatedReminder, err := NewOrExistingReminder(newNote, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, updatedReminder.Save())

		// Check all fields has been updated
		updatedReminder, err = CurrentRepository().LoadReminderByOID(updatedReminder.OID)
		require.NoError(t, err)
		// Some fields must not have changed
		assert.Equal(t, createdReminder.OID, updatedReminder.OID)
		assert.Equal(t, createdReminder.NoteOID, updatedReminder.NoteOID)
		assert.Equal(t, createdReminder.RelativePath, updatedReminder.RelativePath)
		// Some fields must have changed
		assert.WithinDuration(t, createdAt, updatedReminder.CreatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedReminder.UpdatedAt, 1*time.Second)
		assert.WithinDuration(t, updatedAt, updatedReminder.LastCheckedAt, 1*time.Second)
	})

	t.Run("Delete", func(t *testing.T) {
		root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(content))

		FreezeNow(t)
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")

		// Init the file
		parsedFile, err := ParseFileFromRelativePath(root, "project.md")
		require.NoError(t, err)
		file, err := NewFile(nil, parsedFile)
		require.NoError(t, err)
		require.NoError(t, file.Save())
		parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
		require.True(t, ok)
		note, err := NewNote(file, nil, parsedNote)
		require.NoError(t, err)
		require.NoError(t, note.Save())

		// Save the reminder
		parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
		require.True(t, ok)
		reminder, err := NewReminder(note, parsedReminder)
		require.NoError(t, err)
		require.NoError(t, reminder.Save())

		// Delete the reminder
		require.NoError(t, reminder.Delete())

		assert.Equal(t, clock.Now(), reminder.DeletedAt)
		AssertNoReminders(t)
	})
}

func TestReminderFormats(t *testing.T) {
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	root := SetUpRepositoryFromFileContent(t, "project.md", UnescapeTestContent(`
## TODO: Backlog

* [ ] Test ”#reminder-2085-09”
`))

	// Init the file
	parsedFile, err := ParseFileFromRelativePath(root, "project.md")
	require.NoError(t, err)
	file, err := NewFile(nil, parsedFile)
	require.NoError(t, err)

	// Init the reminder
	parsedNote, ok := parsedFile.FindNoteByTitle("TODO: Backlog")
	require.True(t, ok)
	note, err := NewNote(file, nil, parsedNote)
	require.NoError(t, err)
	parsedReminder, ok := parsedNote.FindReminderByTag("#reminder-2085-09")
	require.True(t, ok)
	reminder, err := NewReminder(note, parsedReminder)
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
