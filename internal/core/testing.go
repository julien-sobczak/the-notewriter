package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reset forces singletons to be recreated. Useful between unit tests.
func Reset() {
	collectionOnce.Reset()
	configOnce.Reset()
	dbRemoteOnce.Reset()
	dbClientOnce.Reset()
	dbOnce.Reset()
	loggerOnce.Reset()
}

/* Test Helpers */

func mustCountFiles(t *testing.T) int {
	count, err := CountFiles()
	require.NoError(t, err)
	return count
}

func mustCountMedias(t *testing.T) int {
	count, err := CountMedias()
	require.NoError(t, err)
	return count
}

func mustCountNotes(t *testing.T) int {
	count, err := CountNotes()
	require.NoError(t, err)
	return count
}

func mustCountLinks(t *testing.T) int {
	count, err := CountLinks()
	require.NoError(t, err)
	return count
}

func mustCountFlashcards(t *testing.T) int {
	count, err := CountFlashcards()
	require.NoError(t, err)
	return count
}

func mustCountReminders(t *testing.T) int {
	count, err := CountReminders()
	require.NoError(t, err)
	return count
}

func assertNoFiles(t *testing.T) {
	count, err := CountFiles()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertFrontMatterEqual(t *testing.T, expected string, file *File) {
	actual, err := file.FrontMatterString()
	require.NoError(t, err)
	assertTrimEqual(t, expected, actual)
}

func assertContentEqual(t *testing.T, expected string, file *File) {
	actual := file.Content
	assertTrimEqual(t, expected, actual)
}

func assertTrimEqual(t *testing.T, expected string, actual string) {
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}
