package clock

import (
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/resync"
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
	now time.Time
}

func NewTestClock() *TestClock {
	return NewTestClockAt(time.Now())
}

func NewTestClockAt(date time.Time) *TestClock {
	return &TestClock{
		now: date,
	}
}

func (c *TestClock) Now() time.Time {
	return c.now
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

func FreezeAt(now time.Time) {
	clockSingleton = NewTestClockAt(now)
}

func Freeze() {
	clockSingleton = NewTestClock()
}

func Unfreeze() {
	clockSingleton = nil
	clockOnce.Reset()
}
