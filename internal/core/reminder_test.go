package core

import (
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
