package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepository is the main abstraction to set up a test environment.
// Use options to customize the repository.
type TestRepository struct {
	t *testing.T

	clock *clock.TestClock

	// The repository temporary root directory
	Root string
}

type TestRepositoryOption func(*TestRepository)

// NewTestRepository creates a new TestRepository instance with a temporary directory
// and apply options.
func NewTestRepository(t *testing.T, options ...TestRepositoryOption) *TestRepository {
	t.Helper()

	tr := &TestRepository{
		t:     t,
		clock: clock.NewTestClock(), // default to current time
		Root:  t.TempDir(),
	}

	tr.configureDir()
	t.Logf("Working in configured directory %s", tr.Root)

	for _, opt := range options {
		opt(tr)
	}

	return tr
}

func (tr *TestRepository) configureDir() {
	ntDir := filepath.Join(tr.Root, ".nt")
	if _, err := os.Stat(ntDir); os.IsNotExist(err) {
		// Create a default configuration dir if not exists for CurrentConfig() to work
		if err := os.Mkdir(ntDir, os.ModePerm); err != nil {
			tr.t.Fatal(err)
		}
	}

	// Force debug level in tests to diagnose more easily
	CurrentLogger().SetVerboseLevel(VerboseDebug)
	CurrentLogger().Debugf("✨ Set up directory %q", ntDir)

	configFile := filepath.Join(ntDir, "config.jsonnet")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create a default configuration file but ensure no side effects
		InitConfigFileFromDirectory(tr.Root, ConfigOptions{
			MediaConverter: "random",
		})
	}

	// Force the application to consider the temporary directory as the home
	os.Setenv("NT_HOME", tr.Root)
	tr.t.Cleanup(func() {
		os.Unsetenv("NT_HOME")
		Reset()
	})
}

// Reset forces singletons to be recreated. Useful between unit tests.
func Reset() {
	repositoryOnce.Reset()
	configOnce.Reset()
	converterOnce.Reset()
	dbRemoteOnce.Reset()
	dbClientOnce.Reset()
	dbOnce.Reset()
	loggerOnce.Reset()
	sectionsInventoryOnce.Reset()
	slugInventoryOnce.Reset()
}

/* File Helpers */

// WriteFile edits the file in the current repository to force the given content.
func (tr *TestRepository) WriteFile(relativePath string, content string) {
	root := CurrentConfig().RootDirectory
	newFilepath := filepath.Join(root, relativePath)
	err := os.MkdirAll(filepath.Dir(newFilepath), 0755)
	require.NoError(tr.t, err)
	tr.t.Logf("Writing file %s...", newFilepath)
	err = os.WriteFile(newFilepath, []byte(text.UnescapeTestContent(content)), 0644)
	require.NoError(tr.t, err)
}

// WriteFileRaw rewrite a file in the current repository to force the given content in bytes.
func (tr *TestRepository) WriteFileRaw(relativePath string, data []byte) {
	root := CurrentConfig().RootDirectory
	newFilepath := filepath.Join(root, relativePath)
	tr.t.Logf("Writing file %s...", newFilepath)
	err := os.WriteFile(newFilepath, data, 0644)
	require.NoError(tr.t, err)
}

// DeleteFile removes a file in the current repository.
func (tr *TestRepository) DeleteFile(relativePath string) {
	root := CurrentConfig().RootDirectory
	existingFilepath := filepath.Join(root, relativePath)
	err := os.Remove(existingFilepath)
	require.NoError(tr.t, err)
}

/* Options */

// WithFileRaw creates a file in the repository with the given content.
func WithFileRaw(path string, content []byte) TestRepositoryOption {
	return func(repo *TestRepository) {
		root := repo.Root
		newFilepath := filepath.Join(root, path)
		err := os.MkdirAll(filepath.Dir(newFilepath), 0755)
		require.NoError(repo.t, err)
		repo.t.Logf("Writing file %s...", newFilepath)
		err = os.WriteFile(newFilepath, content, 0644)
		require.NoError(repo.t, err)
	}
}

// WithFile creates a file in the repository with the given content.
// Special quotes are unescaped to support test content.
func WithFile(path string, content string) TestRepositoryOption {
	return WithFileRaw(path, []byte(text.UnescapeTestContent(content)))
}

// FromGoldenFile creates a file in the repository from the golden file.
func FromGoldenFile(t *testing.T) TestRepositoryOption {
	filename := t.Name() + ".md"
	return WithFileRaw(filename, testutil.GoldenFileNamed(t, filename))
}

// FromGoldenFileNamed creates a file in the repository from the golden file.
func FromGoldenFileNamed(t *testing.T, filename string) TestRepositoryOption {
	newFilename := filepath.Base(filename)
	return WithFileRaw(newFilename, testutil.GoldenFileNamed(t, filename))
}

// FromGoldenDir copies in the repository a testdata directory based on the given test name.
func FromGoldenDir(t *testing.T) TestRepositoryOption {
	return FromGoldenDirNamed(t.Name())
}

// FromGoldenDirNamed copies in the repository a testdata directory.
func FromGoldenDirNamed(testname string) TestRepositoryOption {
	return func(tr *TestRepository) {
		dirIn := filepath.Join("testdata", testname)
		dirOut := filepath.Join(tr.Root)

		dirIn, err := filepath.Abs(dirIn)
		require.NoError(tr.t, err)

		// We duplicate everything so that test can create files/directories like .nt
		// inside it without impacting the testdata original directory.)
		testutil.DuplicateDirHierarchy(tr.t, dirIn, dirOut)
	}
}

/* Reproducible Tests */

// FastForward advances the clock by the given duration.
// If the clock is not frozen, it will be frozen first.
func (tr *TestRepository) FastForward(duration time.Duration) time.Time {
	return tr.clock.FastForward(duration)
}

// WithClockBasedFileInfoReader forces the use of a clock-based file info reader
// to have predictable file modification times in tests.
func WithClockBasedFileInfoReader() TestRepositoryOption {
	return func(tr *TestRepository) {
		testutil.FreezeFileInfoReader(tr.t)
	}
}

func (tr *TestRepository) FreezeNow() *clock.TestClock {
	tr.clock = testutil.FreezeNow(tr.t)
	return tr.clock
}

func WithFreezeNow() TestRepositoryOption {
	return func(tr *TestRepository) {
		tr.FreezeNow()
	}
}

func (tr *TestRepository) FreezeAt(point time.Time) *clock.TestClock {
	tr.clock = testutil.FreezeAt(tr.t, point)
	return tr.clock
}

func WithFreezeAt(point time.Time) TestRepositoryOption {
	return func(tr *TestRepository) {
		tr.FreezeAt(point)
	}
}

func WithFreezeOn(date string) TestRepositoryOption {
	return func(tr *TestRepository) {
		tr.FreezeOn(date)
	}
}

func (tr *TestRepository) FreezeOn(date string) *clock.TestClock {
	tr.clock = testutil.FreezeOn(tr.t, date)
	return tr.clock
}

// WithSequenceOIDs allows to use predictable OIDs in tests.
func WithSequenceOIDs() TestRepositoryOption {
	return func(tr *TestRepository) {
		oid.UseSequence(tr.t)
	}
}

// WithFixedOIDs allows to force a fixed OID for every object.
func WithFixedOIDs(value oid.OID) TestRepositoryOption {
	return func(tr *TestRepository) {
		oid.UseFixed(tr.t, value)
	}
}

// WithNextOIDs allows to set the next OIDs to use in tests.
func WithNextOIDs(oids ...string) TestRepositoryOption {
	return func(tr *TestRepository) {
		oid.UseNext(tr.t, oids...)
	}
}

// WithConfigOverride allows to override the current global configuration
func WithConfigOverride(override func(c *Config)) TestRepositoryOption {
	return func(tr *TestRepository) {
		override(CurrentConfig())
	}
}

// WithConfigFileOverride allows to override the current configuration file
func WithConfigFileOverride(override func(c *ConfigFile)) TestRepositoryOption {
	return func(tr *TestRepository) {
		override(CurrentConfigFile())
	}
}

/* Test Helpers */

func (tr *TestRepository) CountFiles() int {
	count, err := CurrentRepository().CountFiles()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountMedias() int {
	count, err := CurrentRepository().CountMedias()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountNotes() int {
	count, err := CurrentRepository().CountNotes()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountGotos() int {
	count, err := CurrentRepository().CountGotos()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountFlashcards() int {
	count, err := CurrentRepository().CountFlashcards()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountReminders() int {
	count, err := CurrentRepository().CountReminders()
	require.NoError(tr.t, err)
	return count
}

func (tr *TestRepository) CountMemories() int {
	count, err := CurrentRepository().CountMemories()
	require.NoError(tr.t, err)
	return count
}

/* Test Assertions */

func (tr *TestRepository) AssertNoFiles() {
	require.Equal(tr.t, 0, tr.CountFiles())
}

func (tr *TestRepository) AssertNoNotes() {
	require.Equal(tr.t, 0, tr.CountNotes())
}

func (tr *TestRepository) AssertNoFlashcards() {
	require.Equal(tr.t, 0, tr.CountFlashcards())
}

func (tr *TestRepository) AssertNoGotos() {
	require.Equal(tr.t, 0, tr.CountGotos())
}

func (tr *TestRepository) AssertNoReminders() {
	require.Equal(tr.t, 0, tr.CountReminders())
}

func (tr *TestRepository) AssertNoMedias() {
	require.Equal(tr.t, 0, tr.CountMedias())
}

func (tr *TestRepository) AssertFrontMatterEqual(expected string, file *File) {
	actual, err := file.FrontMatter.AsBeautifulYAML()
	require.NoError(tr.t, err)
	tr.AssertTrimEqual(expected, actual)
}

func (tr *TestRepository) AssertContentEqual(expected string, file *File) {
	actual := file.Body
	tr.AssertTrimEqual(expected, string(actual))
}

func (tr *TestRepository) AssertTrimEqual(expected string, actual string) {
	assert.Equal(tr.t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

/* Test Queriers */

func (tr *TestRepository) FindFlashcardByShortTitle(shortTitle string) *Flashcard {
	flashcard, err := CurrentRepository().FindFlashcardByShortTitle(shortTitle)
	require.NoError(tr.t, err)
	require.NotNil(tr.t, flashcard)
	return flashcard
}

func (tr *TestRepository) FindNoteByPathAndTitle(relativePath, longTitle string) *Note {
	note, err := CurrentRepository().FindNoteByPathAndTitle(relativePath, longTitle)
	require.NoError(tr.t, err)
	require.NotNil(tr.t, note)
	return note
}

/* Text Helpers */

func (tr *TestRepository) RequireNoFileExists(relativePath string) {
	filepath := filepath.Join(tr.Root, relativePath)
	require.NoFileExists(tr.t, filepath)
}
func (tr *TestRepository) RequireFileExists(relativePath string) {
	filepath := filepath.Join(tr.Root, relativePath)
	require.FileExists(tr.t, filepath)
}

func (tr *TestRepository) AssertFileExists(relativePath string) {
	filepath := filepath.Join(tr.Root, relativePath)
	assert.FileExists(tr.t, filepath)
}

// ReplaceLine replaces a line inside a file.
func (tr *TestRepository) ReplaceLine(relativePath string, lineNumber int, oldLine string, newLine string) {
	ReplaceLine(tr.t, filepath.Join(tr.Root, relativePath), lineNumber, oldLine, newLine)
}

// ReplaceLine replaces a line inside a file.
func ReplaceLine(t *testing.T, path string, lineNumber int, oldLine string, newLine string) {
	t.Helper()
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
func (tr *TestRepository) AppendLines(relativePath string, text string) {
	AppendLines(tr.t, filepath.Join(tr.Root, relativePath), text)
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

// ParseFile creates a ParsedFile from a file in the repository.
func (tr *TestRepository) ParseFile(relativePath string) *ParsedFile {
	absolutePath := CurrentRepository().GetFileAbsolutePath(relativePath)

	// Read the markdown
	markdownFile, err := markdown.ParseFile(absolutePath)
	require.NoError(tr.t, err)

	parsedFile, err := ParseOrphanFile(markdownFile)
	require.NoError(tr.t, err)
	require.NotNil(tr.t, parsedFile)

	return parsedFile
}

// NewPackFile creates a PackFile from a file in the repository.
// The file can be a Markdown document or a media file.
// Parent files are not supported when parsing the Markdown file.
func (tr *TestRepository) NewPackFile(fileRelativePath string) *PackFile {
	fileAbsolutePath := CurrentRepository().GetFileAbsolutePath(fileRelativePath)

	// We create pack files for Markdown and media files.
	// We need to check the kind of file first.

	if filepath.Ext(fileAbsolutePath) == ".md" {
		// Read the markdown
		markdownFile, err := markdown.ParseFile(fileAbsolutePath)
		require.NoError(tr.t, err)

		parsedFile, err := ParseOrphanFile(markdownFile)
		require.NoError(tr.t, err)

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(tr.t, err)

		return packFile
	}

	// Read the media
	parsedMedia := ParseMedia(CurrentRepository().Path(), fileAbsolutePath)
	packFile, err := NewPackFileFromParsedMedia(parsedMedia)
	require.NoError(tr.t, err)

	return packFile
}

/* Dummy Objects */

func DummyPackFile() *PackFile {
	return &PackFile{
		OID: oid.New(),

		// Init a fake file
		FileRelativePath: ".",
		FileMTime:        clock.Now(),
		FileSize:         1,

		// Init pack file properties
		CTime: clock.Now(),
	}
}
