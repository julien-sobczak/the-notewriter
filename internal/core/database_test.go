package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsOnDisk(t *testing.T) {

	SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")

	stats, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)
	assert.Equal(t, map[string]int{
		"file":      0,
		"note":      0,
		"flashcard": 0,
		"media":     0,
		"link":      0,
		"reminder":  0,
	}, stats.Objects)
	require.Equal(t, 0, stats.IndexObjects)
	require.Equal(t, int64(0), stats.TotalSizeKB)

	// Add
	err = CurrentCollection().Add(".")
	require.NoError(t, err)
	stats, err = CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	// Objects are still not written before a commit (except blobs)
	assert.Equal(t, map[string]int{
		"file":      0,
		"note":      0,
		"flashcard": 0,
		"media":     0,
		"link":      0,
		"reminder":  0,
	}, stats.Objects)
	assert.Greater(t, stats.Blobs, 0)
	assert.Equal(t, 0, stats.Commits)
	assert.Equal(t, 0, stats.IndexObjects) // still in staging area

	// Commit
	err = CurrentDB().Commit("")
	require.NoError(t, err)
	stats, err = CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	assert.Equal(t, 1, stats.Commits)
	assert.Greater(t, stats.Objects["file"], 0)
	assert.Greater(t, stats.Objects["note"], 0)
	assert.Greater(t, stats.Objects["flashcard"], 0)
	assert.Greater(t, stats.Objects["media"], 0)
	assert.Greater(t, stats.Objects["link"], 0)
	assert.Greater(t, stats.Objects["reminder"], 0)
	assert.Greater(t, stats.Blobs, 0)
	assert.Greater(t, stats.ObjectFiles, 0)
	assert.Greater(t, stats.IndexObjects, 0)
	require.Greater(t, stats.TotalSizeKB, int64(0))
}
