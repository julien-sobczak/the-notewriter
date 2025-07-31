package testutil_test

import (
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHumanTime(t *testing.T) {
	var tests = []struct {
		name     string
		value    string
		expected time.Time
	}{
		{
			name:     "A date",
			value:    "1985-09-29",
			expected: time.Date(1985, time.Month(9), 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "A date/hour",
			value:    "1985-09-29 02:15",
			expected: time.Date(1985, time.Month(9), 29, 2, 15, 0, 0, time.UTC),
		},
		{
			name:     "A date/hour/seconds",
			value:    "1985-09-29 02:15:10",
			expected: time.Date(1985, time.Month(9), 29, 2, 15, 10, 0, time.UTC),
		},
		{
			name:     "A date/hour/seconds/milliseconds",
			value:    "1985-09-29 02:15:10.555",
			expected: time.Date(1985, time.Month(9), 29, 2, 15, 10, 555000000, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, testutil.HumanTime(t, tt.value))
		})
	}
}
