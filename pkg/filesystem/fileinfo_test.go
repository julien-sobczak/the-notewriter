package filesystem_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardFileInfoReader(t *testing.T) {
	// Create a temporary file
	tmpFile := CreateNonEmptyTempFile(t)

	// Chech the file info
	stat, err := filesystem.Stat(tmpFile.Name())
	require.NoError(t, err)

	// Time must match actual time
	assert.WithinDuration(t, time.Now(), stat.ModTime(), 10*time.Second)
	// Size must be the content size
	assert.Greater(t, stat.Size(), int64(1))
}

func TestClockBasedFileInfoReader_NonEmptyFile(t *testing.T) {
	clock.FreezeAt(time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC))
	defer clock.Unfreeze()
	filesystem.OverrideFileInfoReader(filesystem.NewClockBasedFileInfoReader())
	defer filesystem.RestoreFileInfoReader()

	// Create a temporary file
	tmpFile := CreateNonEmptyTempFile(t)

	// Chech the file info
	stat, err := filesystem.Stat(tmpFile.Name())
	require.NoError(t, err)
	lstat, err := filesystem.Lstat(tmpFile.Name())
	require.NoError(t, err)

	// Time must match clock time
	assert.Equal(t, clock.Now(), stat.ModTime())
	assert.Equal(t, clock.Now(), lstat.ModTime())
	// Size must be static
	assert.EqualValues(t, 1, stat.Size())
	assert.EqualValues(t, 1, lstat.Size())
}

func TestClockBasedFileInfoReader_EmptyFile(t *testing.T) {
	clock.FreezeAt(time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC))
	defer clock.Unfreeze()
	filesystem.OverrideFileInfoReader(filesystem.NewClockBasedFileInfoReader())
	defer filesystem.RestoreFileInfoReader()

	// Create a temporary file
	tmpFile := CreateEmptyTempFile(t)

	// Chech the file info
	stat, err := filesystem.Stat(tmpFile.Name())
	require.NoError(t, err)
	lstat, err := filesystem.Lstat(tmpFile.Name())
	require.NoError(t, err)

	// Time must match clock time
	assert.Equal(t, clock.Now(), stat.ModTime())
	assert.Equal(t, clock.Now(), lstat.ModTime())
	// Size must be static
	assert.EqualValues(t, 0, stat.Size())
	assert.EqualValues(t, 0, lstat.Size())
}

func ExampleClockBasedFileInfoReader() {
	clock.FreezeAt(time.Date(2023, 01, 01, 14, 00, 00, 00, time.UTC))
	defer clock.Unfreeze()
	filesystem.OverrideFileInfoReader(filesystem.NewClockBasedFileInfoReader())
	defer filesystem.RestoreFileInfoReader()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up

	// Chech the file info
	stat, err := filesystem.Stat(tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	// Time must match actual time
	fmt.Println(stat.Size(), stat.ModTime())
	// Output: 0 2023-01-01 14:00:00 +0000 UTC
}

/* Test Helpers */

func CreateEmptyTempFile(t *testing.T) *os.File {
	// Define a new file
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})
	return tmpFile
}

func CreateNonEmptyTempFile(t *testing.T) *os.File {
	// Define a new file
	tmpFile := CreateEmptyTempFile(t)

	// Write some data to the file
	content := []byte("temporary file content")
	_, err := tmpFile.Write(content)
	require.NoError(t, err)

	return tmpFile
}
