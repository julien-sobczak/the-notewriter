package testutil

import (
	"regexp"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/stretchr/testify/require"
)

// FreezeNow freezes the clock at the current time and ensures it is cleaned up after the test.
func FreezeNow(t *testing.T) *clock.TestClock {
	now := clock.Freeze()
	t.Cleanup(clock.Unfreeze)
	return now
}

// FreezeAtXXX freezes the clock at a specific time and ensures it is cleaned up after the test.
func FreezeAt(t *testing.T, point time.Time) *clock.TestClock {
	now := clock.FreezeAt(point)
	t.Cleanup(clock.Unfreeze)
	return now
}

// FreezeOnXXX freezes the clock at a specific date string and ensures it is cleaned up after the test.
func FreezeOn(t *testing.T, date string) *clock.TestClock {
	point := HumanTime(t, date)
	require.False(t, point.IsZero())
	return FreezeAt(t, point)
}

// HumanTime parses a string into a time.Time supporting different formats to make tests more readable.
func HumanTime(t *testing.T, str string) time.Time {
	patterns := map[string]string{
		"2006-01-02":              `^\d{4}-\d{2}-\d{2}$`,
		"2006-01-02 15:04":        `^\d{4}-\d{2}-\d{2} \d{1,2}:\d{2}$`,
		"2006-01-02 15:04:05":     `^\d{4}-\d{2}-\d{2} \d{1,2}:\d{2}:\d{2}$`,
		"2006-01-02 15:04:05.000": `^\d{4}-\d{2}-\d{2} \d{1,2}:\d{2}:\d{2}[.]\d{3}$`,
	}
	for layout, regex := range patterns {
		if match, _ := regexp.MatchString(regex, str); match {
			result, err := time.Parse(layout, str)
			require.NoError(t, err)
			return result
		}
	}
	t.Fatalf("No matching pattern for date %q", str)
	return time.Time{} // zero
}
