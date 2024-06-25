package core

import (
	"os"
	"path/filepath"
	"regexp"
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
	repositoryOnce.Reset()
	configOnce.Reset()
	dbRemoteOnce.Reset()
	dbClientOnce.Reset()
	dbOnce.Reset()
	loggerOnce.Reset()
	sectionsInventoryOnce.Reset()
	slugInventoryOnce.Reset()
}

/* Fixtures */

// SetUpRepositoryFromGoldenFile populates a temp directory containing a valid .nt repository and a single file.
func SetUpRepositoryFromGoldenFile(t *testing.T) string {
	return SetUpRepositoryFromGoldenFileNamed(t, t.Name()+".md")
}

// SetUpRepositoryFromGoldenFileNamed populates a temp directory based on the given golden file name.
func SetUpRepositoryFromGoldenFileNamed(t *testing.T, testname string) string {
	filename := testutil.SetUpFromGoldenFileNamed(t, testname)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpRepositoryFromFileContent populates a temp directory based on the given file content.
func SetUpRepositoryFromFileContent(t *testing.T, name, content string) string {
	filename := testutil.SetUpFromFileContent(t, name, content)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpRepositoryFromGoldenDir populates a temp directory containing a valid .nt repository.
func SetUpRepositoryFromGoldenDir(t *testing.T) string {
	return SetUpRepositoryFromGoldenDirNamed(t, t.Name())
}

// SetUpRepositoryFromGoldenDir populates a temp directory based on the given golden dir name.
func SetUpRepositoryFromGoldenDirNamed(t *testing.T, testname string) string {
	dirname := testutil.SetUpFromGoldenDirNamed(t, testname)
	configureDir(t, dirname)
	return dirname
}

// SetUpRepositoryFromTempDir populates a temp directory containing a valid .nt repository.
func SetUpRepositoryFromTempDir(t *testing.T) string {
	dirname := t.TempDir()
	configureDir(t, dirname)
	t.Logf("Working in configured directory %s", dirname)
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

func MustCountFiles(t *testing.T) int {
	count, err := CurrentRepository().CountFiles()
	require.NoError(t, err)
	return count
}

func MustCountMedias(t *testing.T) int {
	count, err := CurrentRepository().CountMedias()
	require.NoError(t, err)
	return count
}

func MustCountNotes(t *testing.T) int {
	count, err := CurrentRepository().CountNotes()
	require.NoError(t, err)
	return count
}

func MustCountGoLinks(t *testing.T) int {
	count, err := CurrentRepository().CountGoLinks()
	require.NoError(t, err)
	return count
}

func MustCountFlashcards(t *testing.T) int {
	count, err := CurrentRepository().CountFlashcards()
	require.NoError(t, err)
	return count
}

func MustCountReminders(t *testing.T) int {
	count, err := CurrentRepository().CountReminders()
	require.NoError(t, err)
	return count
}

func AssertNoFiles(t *testing.T) {
	count, err := CurrentRepository().CountFiles()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertNoNotes(t *testing.T) {
	count, err := CurrentRepository().CountNotes()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertNoFlashcards(t *testing.T) {
	count, err := CurrentRepository().CountFlashcards()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertNoGoLinks(t *testing.T) {
	count, err := CurrentRepository().CountGoLinks()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertNoReminders(t *testing.T) {
	count, err := CurrentRepository().CountReminders()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertNoMedias(t *testing.T) {
	count, err := CurrentRepository().CountMedias()
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func AssertFrontMatterEqual(t *testing.T, expected string, file *File) {
	actual, err := file.FrontMatter.AsBeautifulYAML()
	require.NoError(t, err)
	AssertTrimEqual(t, expected, actual)
}

func AssertContentEqual(t *testing.T, expected string, file *File) {
	actual := file.Body
	AssertTrimEqual(t, expected, string(actual))
}

func AssertTrimEqual(t *testing.T, expected string, actual string) {
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func MustFindFlashcardByShortTitle(t *testing.T, shortTitle string) *Flashcard {
	flashcard, err := CurrentRepository().FindFlashcardByShortTitle(shortTitle)
	require.NoError(t, err)
	require.NotNil(t, flashcard)
	return flashcard
}

func MustFindNoteByPathAndTitle(t *testing.T, relativePath, longTitle string) *Note {
	note, err := CurrentRepository().FindNoteByPathAndTitle(relativePath, longTitle)
	require.NoError(t, err)
	require.NotNil(t, note)
	return note
}

/* Test Helpers */

// MustWriteFile edits the file in the current repository to force the given content.
func MustWriteFile(t *testing.T, path string, content string) {
	root := CurrentConfig().RootDirectory
	newFilepath := filepath.Join(root, path)
	err := os.WriteFile(newFilepath, []byte(UnescapeTestContent(content)), 0644)
	require.NoError(t, err)
}

// UnescapeTestContent supports content using a special character instead of backticks.
func UnescapeTestContent(content string) string {
	// We support a special syntax for backticks in content.
	// Backticks are used to define note attributes (= common syntax with The NoteWriter) but
	// multiline strings in Golang cannot contains backticks.
	// We allows the ” character instead as suggested here: https://stackoverflow.com/a/59900008
	//
	// Example: ”@slug: toto” will become `@slug: toto`
	return strings.ReplaceAll(content, "”", "`")
}

// MustDeleteFile remove a file iin the current repository.
func MustDeleteFile(t *testing.T, path string) {
	root := CurrentConfig().RootDirectory
	existingFilepath := filepath.Join(root, path)

	err := os.Remove(existingFilepath)
	require.NoError(t, err)
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

/* Date Management */

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
