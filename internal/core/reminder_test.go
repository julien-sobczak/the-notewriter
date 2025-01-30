package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReminder(t *testing.T) {
	SetUpRepositoryFromTempDir(t)
	FreezeNow(t)

	AssertNoReminders(t)

	createdAt := clock.Now()
	reminder := &Reminder{
		OID:         "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID: "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:     "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		NoteOID:     "52d02a28a961471db62c6d40d30639dafe4aba00",

		RelativePath: "project.md",
		Description:  "Test",
		Tag:          "#reminder-2085-09",

		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		IndexedAt: createdAt,
	}

	// Process and save the reminder
	require.NoError(t, reminder.Next())
	require.NoError(t, reminder.Save())

	require.Equal(t, 1, MustCountReminders(t))

	// Reread and recheck all fields
	actual, err := CurrentRepository().LoadReminderByOID(reminder.OID)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, reminder.OID, actual.OID)
	assert.Equal(t, reminder.PackFileOID, actual.PackFileOID)
	assert.Equal(t, reminder.FileOID, actual.FileOID)
	assert.Equal(t, reminder.NoteOID, actual.NoteOID)
	assert.Equal(t, reminder.RelativePath, actual.RelativePath)
	assert.Equal(t, reminder.Description, actual.Description)
	assert.Equal(t, reminder.Tag, actual.Tag)
	assert.Empty(t, reminder.LastPerformedAt)
	assert.Equal(t, HumanTime(t, "2085-09-01 00:00:00"), reminder.NextPerformedAt)
	assert.WithinDuration(t, createdAt, actual.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, createdAt, actual.UpdatedAt, 1*time.Second)
	assert.WithinDuration(t, createdAt, actual.IndexedAt, 1*time.Second)

	// Force update
	actual.Tag = "#reminder-2050-01"
	require.NoError(t, actual.Save())
	require.Equal(t, 1, MustCountReminders(t))

	// Compare
	actual, err = CurrentRepository().LoadReminderByOID(reminder.OID)
	require.NoError(t, err)
	assert.Equal(t, "#reminder-2050-01", actual.Tag)

	// Delete
	require.NoError(t, reminder.Delete())
	AssertNoReminders(t)
}

func TestReminderFormats(t *testing.T) {
	FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))

	reminder := &Reminder{
		OID:         "42d74d967d9b4e989502647ac510777ca1e22f4a",
		PackFileOID: "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
		FileOID:     "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
		NoteOID:     "52d02a28a961471db62c6d40d30639dafe4aba00",

		RelativePath: "project.md",
		Description:  "Test",
		Tag:          "#reminder-2085-09",

		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),
		IndexedAt: clock.Now(),
	}
	require.NoError(t, reminder.Next())

	t.Run("ToYAML", func(t *testing.T) {
		actual := reminder.ToYAML()

		expected := UnescapeTestContent(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
packfile_oid: 9c0c0682bd18439d992639f19f8d552bde3bd3c0
file_oid: 3e8d915d4e524560ae8a2e5a45553f3034b391a2
note_oid: 52d02a28a961471db62c6d40d30639dafe4aba00
relative_path: project.md
description: Test
tag: '#reminder-2085-09'
last_performed_at: 0001-01-01T00:00:00Z
next_performed_at: 2085-09-01T00:00:00Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
indexed_at: 2023-01-01T01:12:30Z
`)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToJSON", func(t *testing.T) {
		actual := reminder.ToJSON()
		expected := UnescapeTestContent(`
{
  "oid": "42d74d967d9b4e989502647ac510777ca1e22f4a",
  "packfile_oid": "9c0c0682bd18439d992639f19f8d552bde3bd3c0",
  "file_oid": "3e8d915d4e524560ae8a2e5a45553f3034b391a2",
  "note_oid": "52d02a28a961471db62c6d40d30639dafe4aba00",
  "relative_path": "project.md",
  "description": "Test",
  "tag": "#reminder-2085-09",
  "last_performed_at": "0001-01-01T00:00:00Z",
  "next_performed_at": "2085-09-01T00:00:00Z",
  "created_at": "2023-01-01T01:12:30Z",
  "updated_at": "2023-01-01T01:12:30Z",
  "indexed_at": "2023-01-01T01:12:30Z"
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
