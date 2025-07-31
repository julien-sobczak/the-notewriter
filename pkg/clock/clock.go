package clock

import (
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/resync"
)

var (
	// Lazy-load
	clockOnce      resync.Once
	clockSingleton Clock
)

type Clock interface {
	Now() time.Time
}

type DefaultClock struct{}

func (c DefaultClock) Now() time.Time {
	return time.Now()
}

type TestClock struct {
	frozen time.Time
}

func NewTestClock() *TestClock {
	return &TestClock{}
}

func NewTestClockAt(date time.Time) *TestClock {
	return &TestClock{
		frozen: date,
	}
}

func (c *TestClock) FastForward(d time.Duration) time.Time {
	if c.frozen.IsZero() {
		c.frozen = time.Now()
	}
	c.frozen = c.frozen.Add(d)
	return c.frozen
}

func (c *TestClock) Now() time.Time {
	if c.frozen.IsZero() {
		return time.Now()
	}
	// Return the frozen time
	return c.frozen
}

func CurrentClock() Clock {
	if clockSingleton != nil {
		return clockSingleton
	}
	clockOnce.Do(func() {
		clockSingleton = DefaultClock{}
	})
	return clockSingleton
}

// Same as time.Now() but makes possible to control time from unit tests.
func Now() time.Time {
	return CurrentClock().Now()
}

func FreezeAt(now time.Time) *TestClock {
	testClock := NewTestClockAt(now)
	clockSingleton = testClock
	return testClock
}

func Freeze() *TestClock {
	return FreezeAt(time.Now())
}

func Unfreeze() {
	clockSingleton = nil
	clockOnce.Reset()
}
