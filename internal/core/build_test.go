package core

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	outputDir := t.TempDir()
	dirname := SetUpCollectionFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md")) // DEBUG why not exist :thinking

	err := CurrentCollection().Build(outputDir)
	require.NoError(t, err)

	// Check database exists
	dbFilepath := filepath.Join(CurrentConfig().RootDirectory, ".nt/database.db")
	require.FileExists(t, dbFilepath)

	db, err := sql.Open("sqlite3", dbFilepath)
	require.NoError(t, err)
	var count int
	err = db.QueryRow(`SELECT count(*) FROM file`).Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}
