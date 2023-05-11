package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
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
	sectionsInventoryOnce.Reset()
}

/* Reproducible Tests */

// FreezeNow wraps the clock API to register the cleanup function at the end of the test.
func FreezeNow(t *testing.T) time.Time {
	now := clock.Freeze()
	t.Cleanup(clock.Unfreeze)
	return now
}

// FreezeAt wraps the clock API to register the cleanup function at the end of the test.
func FreezeAt(t *testing.T, point time.Time) time.Time {
	now := clock.FreezeAt(point)
	t.Cleanup(clock.Unfreeze)
	return now
}

// SetNextOIDs configures a predefined list of OID
func SetNextOIDs(t *testing.T, oids ...string) {
	oidGenerator = &suiteOIDGenerator{
		nextOIDs: oids,
	}
	t.Cleanup(ResetOID)
}

// UseFixedOID configures a fixed OID value
func UseFixedOID(t *testing.T, value string) {
	oidGenerator = &fixedOIDGenerator{
		oid: value,
	}
	t.Cleanup(ResetOID)
}

// UseFixedOID configures a fixed OID value
func UseSequenceOID(t *testing.T) {
	oidGenerator = &sequenceOIDGenerator{}
	t.Cleanup(ResetOID)
}

/* Test Helpers */

func mustCountFiles(t *testing.T) int {
	count, err := CurrentCollection().CountFiles()
	require.NoError(t, err)
	return count
}

func mustCountMedias(t *testing.T) int {
	count, err := CurrentCollection().CountMedias()
	require.NoError(t, err)
	return count
}

func mustCountNotes(t *testing.T) int {
	count, err := CurrentCollection().CountNotes()
	require.NoError(t, err)
	return count
}

func mustCountLinks(t *testing.T) int {
	count, err := CurrentCollection().CountLinks()
	require.NoError(t, err)
	return count
}

func mustCountFlashcards(t *testing.T) int {
	count, err := CurrentCollection().CountFlashcards()
	require.NoError(t, err)
	return count
}

func mustCountReminders(t *testing.T) int {
	count, err := CurrentCollection().CountReminders()
	require.NoError(t, err)
	return count
}

func assertNoFiles(t *testing.T) {
	count, err := CurrentCollection().CountFiles()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertNoNotes(t *testing.T) {
	count, err := CurrentCollection().CountNotes()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertNoFlashcards(t *testing.T) {
	count, err := CurrentCollection().CountFlashcards()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertNoLinks(t *testing.T) {
	count, err := CurrentCollection().CountLinks()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertNoReminders(t *testing.T) {
	count, err := CurrentCollection().CountReminders()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertNoMedias(t *testing.T) {
	count, err := CurrentCollection().CountMedias()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func assertFrontMatterEqual(t *testing.T, expected string, file *File) {
	actual, err := file.FrontMatterString()
	require.NoError(t, err)
	assertTrimEqual(t, expected, actual)
}

func assertContentEqual(t *testing.T, expected string, file *File) {
	actual := file.Body
	assertTrimEqual(t, expected, actual)
}

func assertTrimEqual(t *testing.T, expected string, actual string) {
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}
