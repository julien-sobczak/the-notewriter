package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestReminder(t *testing.T) {
	// Make tests reproductible
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("YAML", func(t *testing.T) {
		noteSrc := NewNote(NewEmptyFile("example.md"), "TODO: Backlog", "* [ ] Test `#reminder-2025-09`", 2)
		reminderSrc, err := NewReminder(noteSrc, "Test", "#reminder-2025-09")
		require.NoError(t, err)

		// Marshall
		buf := new(bytes.Buffer)
		err = reminderSrc.Write(buf)
		require.NoError(t, err)
		reminderYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: example.md
description_raw: Test
description_markdown: Test
description_html: <p>Test</p>
description_text: Test
tag: '#reminder-2025-09'
last_performed_at: 0001-01-01T00:00:00Z
next_performed_at: 2025-09-01T00:00:00Z
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(reminderYAML))

		// Unmarshall
		reminderDest := new(Reminder)
		err = reminderDest.Read(buf)
		require.NoError(t, err)

		// Compare ignore certain attributes
		reminderSrc.File = nil
		reminderSrc.Note = nil
		reminderSrc.new = false
		reminderSrc.stale = false
		assert.EqualValues(t, reminderSrc, reminderDest)
	})

}
