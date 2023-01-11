package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpFromGoldenFile(t *testing.T) {
	filename := SetUpFromGoldenFile(t)

	assert.Equal(t, "TestSetUpFromGoldenFile.md", filepath.Base(filename))
	bytes, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, GoldenFile(t), bytes)
}

func TestSetUpFromGoldenDir(t *testing.T) {
	dirname := SetUpFromGoldenDir(t)

	assertFileContains(t, filepath.Join(dirname, "notes.md"), "# Notes\n\nMy Personal notes\n")
	requireDirExists(t, filepath.Join(dirname, "medias"))
	requireFileExists(t, filepath.Join(dirname, "medias/wikimedia.svg"))
	assertFileContains(t, filepath.Join(dirname, "projects/todo.md"), "# TODO\n\n## TODO: Backlog\n\n* [x] Create backlog\n* [ ] Deploy\n")
}

func TestGoldenFile(t *testing.T) {
	content := GoldenFile(t)
	assert.Equal(t, "# TestGoldenFile\n\nHi!\n", string(content))
}

func TestGoldenFileNamed(t *testing.T) {
	content := GoldenFileNamed(t, "TestGoldenFileNamedWithAnotherName.md")
	assert.Equal(t, "# TestGoldenFileNamedWithAnotherName\n\nHello!\n", string(content))
}

/* Test Assertions */

func requireDirExists(t *testing.T, dirname string) {
	stat, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		t.Fatalf("%v doesn't exist", dirname)
	}
	if !stat.IsDir() {
		t.Fatalf("%v isn't a directory", dirname)
	}
}

func requireFileExists(t *testing.T, filename string) {
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Fatalf("%v doesn't exist", filename)
	}
	if stat.IsDir() {
		t.Fatalf("%v is a directory", filename)
	}
}

func assertFileContains(t *testing.T, filename string, expected string) {
	actual, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}
