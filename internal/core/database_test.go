package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsOnDisk(t *testing.T) {
	SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

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
	err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)
	statsAdd, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	// Objects are already written before a commit
	assert.Greater(t, statsAdd.Objects["file"], 0)
	assert.Greater(t, statsAdd.Objects["note"], 0)
	assert.Greater(t, statsAdd.Objects["flashcard"], 0)
	assert.Greater(t, statsAdd.Objects["media"], 0)
	assert.Greater(t, statsAdd.Objects["link"], 0)
	assert.Greater(t, statsAdd.Objects["reminder"], 0)
	assert.Greater(t, statsAdd.Blobs, 0)
	assert.Greater(t, statsAdd.ObjectFiles, 0)
	assert.Greater(t, statsAdd.IndexObjects, 0)
	assert.Greater(t, statsAdd.TotalSizeKB, int64(0))

	// Commit
	err = CurrentRepository().Commit()
	require.NoError(t, err)
	statsCommit, err := CurrentDB().StatsOnDisk()
	require.NoError(t, err)

	assert.Equal(t, statsAdd.Objects["file"], statsCommit.Objects["file"])
	assert.Equal(t, statsAdd.Objects["note"], statsCommit.Objects["note"])
	assert.Equal(t, statsAdd.Objects["flashcard"], statsCommit.Objects["flashcard"])
	assert.Equal(t, statsAdd.Objects["media"], statsCommit.Objects["media"])
	assert.Equal(t, statsAdd.Objects["link"], statsCommit.Objects["link"])
	assert.Equal(t, statsAdd.Objects["reminder"], statsCommit.Objects["reminder"])
	assert.Equal(t, statsAdd.Blobs, statsCommit.Blobs)
	assert.Equal(t, statsAdd.ObjectFiles, statsCommit.ObjectFiles)
	assert.Equal(t, statsAdd.IndexObjects, statsCommit.IndexObjects)
	assert.Equal(t, statsAdd.TotalSizeKB, statsCommit.TotalSizeKB)
}
