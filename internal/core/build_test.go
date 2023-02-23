package core

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	outputDir := t.TempDir()
	dirname := SetUpCollectionFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md"))

	result, err := CurrentCollection().Build(outputDir)
	require.NoError(t, err)
	assertActionOnFile(t, result, "projects/project-A/notes", Added)

	// Check database exists
	dbFilepath := filepath.Join(CurrentConfig().RootDirectory, ".nt/database.db")
	require.FileExists(t, dbFilepath)

	// Open the database file
	db, err := sql.Open("sqlite3", dbFilepath)
	require.NoError(t, err)
	var count int
	err = db.QueryRow(`SELECT count(*) FROM file`).Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)

	countFiles := mustCountFiles(t)
	countNotes := mustCountNotes(t)
	countMedias := mustCountMedias(t)
	countFlashcards := mustCountFlashcards(t)
	countLinks := mustCountLinks(t)
	countReminders := mustCountReminders(t)

	// Rebuild without any changes
	result, err = CurrentCollection().Build(outputDir)
	require.NoError(t, err)
	assertActionOnFile(t, result, "thoughts/on-notetaking", None)
	assert.Equal(t, countFiles, mustCountFiles(t))
	assert.Equal(t, countNotes, mustCountNotes(t))
	assert.Equal(t, countMedias, mustCountMedias(t))
	assert.Equal(t, countFlashcards, mustCountFlashcards(t))
	assert.Equal(t, countLinks, mustCountLinks(t))
	assert.Equal(t, countReminders, mustCountReminders(t))

	// Delete a file and rebuild
	err = os.Remove(filepath.Join(dirname, "thoughts/on-notetaking.md"))
	require.NoError(t, err)
	result, err = CurrentCollection().Build(outputDir)
	require.NoError(t, err)

	// A file must be missing
	assertActionOnFile(t, result, "thoughts/on-notetaking", Deleted)
	assert.Equal(t, countFiles-1, mustCountFiles(t))
}

/* Custom Assertions */

func assertActionOnFile(t *testing.T, result *BuildResult, fileName string, state State) {
	value, ok := result.files[fileName]
	require.True(t, ok, "file %q unknown", fileName)
	require.Equal(t, state, value, "mismatch action. Got: %q, Want: %q", value, state)
}
