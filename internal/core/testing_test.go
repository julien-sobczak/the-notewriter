package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpRepositoryFromGoldenDirNamed(t *testing.T) {
	dirname := SetUpRepositoryFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md"))
}

func TestReplaceLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	os.WriteFile(path, []byte("Hello\nWorld"), 0644)

	ReplaceLine(t, path, 1, "Hello", "Hi")

	newContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hi\nWorld", string(newContent))

	ReplaceLine(t, path, 2, "World", "You")

	newContent, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hi\nYou", string(newContent))
}

func TestAppendLines(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	os.WriteFile(path, []byte("Hello\nWorld"), 0644)

	AppendLines(t, path, "Hi")

	newContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi", string(newContent))

	AppendLines(t, path, "Bonjour\nCoucou\n")

	newContent, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi\nBonjour\nCoucou\n", string(newContent))
}

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
			assert.Equal(t, tt.expected, HumanTime(t, tt.value))
		})
	}
}
