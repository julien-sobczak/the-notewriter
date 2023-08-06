package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/testutil"
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

/* Fixtures */

// SetUpCollectionFromGoldenFile populates a temp directory containing a valid .nt collection and a single file.
func SetUpCollectionFromGoldenFile(t *testing.T) string {
	return SetUpCollectionFromGoldenFileNamed(t, t.Name()+".md")
}

// SetUpCollectionFromGoldenFileNamed populates a temp directory based on the given golden file name.
func SetUpCollectionFromGoldenFileNamed(t *testing.T, testname string) string {
	filename := testutil.SetUpFromGoldenFileNamed(t, testname)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpCollectionFromFileContent populates a temp directory based on the given file content.
func SetUpCollectionFromFileContent(t *testing.T, name, content string) string {
	filename := testutil.SetUpFromFileContent(t, name, content)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpCollectionFromGoldenDir populates a temp directory containing a valid .nt collection.
func SetUpCollectionFromGoldenDir(t *testing.T) string {
	return SetUpCollectionFromGoldenDirNamed(t, t.Name())
}

// SetUpCollectionFromGoldenDir populates a temp directory based on the given golden dir name.
func SetUpCollectionFromGoldenDirNamed(t *testing.T, testname string) string {
	dirname := testutil.SetUpFromGoldenDirNamed(t, testname)
	configureDir(t, dirname)
	return dirname
}

// SetUpCollectionFromTempDir populates a temp directory containing a valid .nt collection.
func SetUpCollectionFromTempDir(t *testing.T) string {
	dirname := t.TempDir()
	configureDir(t, dirname)
	return dirname
}

func configureDir(t *testing.T, dirname string) {
	ntDir := filepath.Join(dirname, ".nt")
	if _, err := os.Stat(ntDir); os.IsNotExist(err) {
		// Create a default configuration if not exists for CurrentConfig() to work
		if err := os.Mkdir(ntDir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ntDir, "config"), []byte(`
[core]
extensions=["md", "markdown"]

[medias]
command="random"
`), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}
	// Force the application to consider the temporary directory as the home
	os.Setenv("NT_HOME", dirname)
	t.Cleanup(func() {
		os.Unsetenv("NT_HOME")
		Reset()
	})

	// Force debug level in tests to diagnose more easily
	CurrentLogger().SetVerboseLevel(VerboseDebug)
	CurrentLogger().Debugf("✨ Set up directory %q", ntDir)
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

/* Text Helpers */

// ReplaceLine replaces a line inside a file.
func ReplaceLine(t *testing.T, path string, lineNumber int, oldLine string, newLine string) {
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(string(data), "\n")
	require.LessOrEqual(t, lineNumber, len(lines))
	require.Equal(t, oldLine, lines[lineNumber-1])
	lines[lineNumber-1] = newLine
	content := strings.Join(lines, "\n")
	os.WriteFile(path, []byte(content), 0644)
}

// AppendLines append multiple lines in a file.
func AppendLines(t *testing.T, path string, text string) {
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(string(data), "\n")
	newLines := strings.Split(text, "\n")
	lines = append(lines, newLines...)
	content := strings.Join(lines, "\n")
	os.WriteFile(path, []byte(content), 0644)
}
