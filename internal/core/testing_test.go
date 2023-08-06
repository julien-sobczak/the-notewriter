package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpCollectionFromGoldenDirNamed(t *testing.T) {
	dirname := SetUpCollectionFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md"))
}

func TestReplaceLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	os.WriteFile(path, []byte("Hello\nWorld"), 0644)

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
	os.WriteFile(path, []byte("Hello\nWorld"), 0644)

	AppendLines(t, path, "Hi")

	newContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi", string(newContent))

	AppendLines(t, path, "Bonjour\nCoucou\n")

	newContent, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld\nHi\nBonjour\nCoucou\n", string(newContent))
}
