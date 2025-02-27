package filesystem

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSize(t *testing.T) {
	dir := t.TempDir()

	knownPath := filepath.Join(dir, "known.txt")
	unknownPath := filepath.Join(dir, "unknown.txt")

	// Create the known file
	err := os.WriteFile(knownPath, []byte("Hello World!"), 0644)
	require.NoError(t, err)

	size, err := FileSize(knownPath)
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))

	size, err = FileSize(unknownPath)
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()

	err := os.MkdirAll(filepath.Join(dir, "sub/sub"), 0755)
	require.NoError(t, err)
	filepathA := filepath.Join(dir, "fileA")
	filepathB := filepath.Join(dir, "sub/fileB")
	filepathC := filepath.Join(dir, "sub/sub/fileC")

	// Create some files
	randomTextFile(t, filepathA, 2*KB)
	randomTextFile(t, filepathB, 4*KB)
	randomTextFile(t, filepathC, 8*KB)

	size, err := DirSize(dir)
	require.NoError(t, err)
	assert.Greater(t, size, int64(12*KB))
	assert.Less(t, size, int64(16*KB))
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()

	err := os.MkdirAll(filepath.Join(dir, "sub/sub"), 0755)
	require.NoError(t, err)
	filepathA := filepath.Join(dir, "fileA")
	filepathB := filepath.Join(dir, "sub/fileB")
	filepathC := filepath.Join(dir, "sub/sub/fileC")

	// Create some files
	randomTextFile(t, filepathA, 2*KB)
	randomTextFile(t, filepathB, 4*KB)
	randomTextFile(t, filepathC, 8*KB)

	paths, err := ListFiles(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "fileA"),
		filepath.Join(dir, "sub/fileB"),
		filepath.Join(dir, "sub/sub/fileC"),
	}, paths)
}

/* Test Helpers */

func randomTextFile(t *testing.T, path string, n int) {
	rand.NewSource(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	err := os.WriteFile(path, []byte(string(b)), 0644)
	require.NoError(t, err)
}
