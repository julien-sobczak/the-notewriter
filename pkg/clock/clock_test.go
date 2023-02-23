package clock_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClock(t *testing.T) {
	t1 := time.Now()
	assert.WithinDuration(t, t1, clock.Now(), 1*time.Second)
	time.Sleep(200 * time.Millisecond)
	// time is not frozen by default
	assert.NotEqual(t, t1, clock.Now())
}

func TestTestClock(t *testing.T) {
	clock.Freeze()
	defer clock.Unfreeze()
	t1 := clock.Now()
	time.Sleep(200 * time.Millisecond)
	// time is always the same
	t2 := clock.Now()
	assert.Equal(t, t1, t2)
}

func TestTestClockAt(t *testing.T) {
	point := time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC)
	clock.FreezeAt(point)
	defer clock.Unfreeze()
	assert.Equal(t, point, clock.Now())
}

func TestSwitchClock(t *testing.T) {
	// Time passes
	assert.WithinDuration(t, time.Now(), clock.Now(), 1*time.Second)

	// Fime is frozen
	point := time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC)
	clock.FreezeAt(point)
	defer clock.Unfreeze()
	assert.Equal(t, point, clock.Now(), 1*time.Second)

	// Fime is unfreezed
	clock.Unfreeze()
	assert.WithinDuration(t, time.Now(), clock.Now(), 1*time.Second)
}

func ExampleClock() {
	point := time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC)
	clock.FreezeAt(point)
	defer clock.Unfreeze()

	fmt.Println(clock.Now())
	// Output: 2023-01-01 14:00:00 +0000 UTC
}
