package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestRepository(t *testing.T) {

	t.Run("No option", func(t *testing.T) {
		tr := NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))
		require.FileExists(t, filepath.Join(tr.Root, "go.md"))
	})
}

func TestReplaceLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	err := os.WriteFile(path, []byte("Hello\nWorld"), 0644)
	require.NoError(t, err)

	ReplaceLine(t, path, 1, "Hello", "Hi")

	newContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hi\nWorld", string(newContent))

	ReplaceLine(t, path, 2, "World", "You")

	newContent, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hi\nYou", string(newContent))
}

func TestAppendLines(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	err := os.WriteFile(path, []byte("Hello\nWorld"), 0644)
	require.NoError(t, err)

	AppendLines(t, path, "Hi")

	newContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi", string(newContent))

	AppendLines(t, path, "Bonjour\nCoucou\n")

	newContent, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi\nBonjour\nCoucou\n", string(newContent))
}
