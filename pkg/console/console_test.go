package console_test

import (
	"bytes"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/console"
	"gotest.tools/assert"
)

func TestNewProgressLog_default(t *testing.T) {
	var out bytes.Buffer

	l := console.NewProgressLog(2,
		// Override options for unit-testing purposes
		console.ToWriter(&out),
		console.LineLength(30))

	for i := range 2 + 1 {
		l.Log(i, "Processing...")
	}
	l.Clear("Done!!!!!!!!!!!!!!!!!!!!!!!!!!")

	expected := "" +
		"           (0/2) Processing...\r" +
		"#####      (1/2) Processing...\r" +
		"########## (2/2) Processing...\r" +
		"Done!!!!!!!!!!!!!!!!!!!!!!!!!!\n"
	assert.Equal(t, out.String(), expected)
}

func TestNewProgressLog_percent(t *testing.T) {
	var out bytes.Buffer

	l := console.NewProgressLog(5,
		console.ShowPercent(),
		// Override options for unit-testing purposes
		console.ToWriter(&out),
		console.LineLength(30))

	for i := range 5 + 1 {
		l.Log(i, "Processing...")
	}
	l.Clear("")

	actual := out.String()
	expected := "" +
		"           (  0%) Processing..\r" +
		"##         ( 20%) Processing..\r" +
		"####       ( 40%) Processing..\r" +
		"######     ( 60%) Processing..\r" +
		"########   ( 80%) Processing..\r" +
		"########## (100%) Processing..\r" +
		"                              \r"
	assert.Equal(t, actual, expected)
}

/* Test Helpers */

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
