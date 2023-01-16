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
	requireDirExists(t, filepath.Join(dirname, "medias"))                // Follow link
	requireFileExists(t, filepath.Join(dirname, "medias/wikimedia.svg")) // Follow link
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

func assertFileContains(t *testing.T, filename string, expected string) {
	actual, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}

func requireDirExists(t *testing.T, dirname string) {
	info, err := os.Stat(dirname) // follow symlinks unlike assert.DirExists
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("unable to find file %q", dirname)
		}
		t.Fatalf("error when running os.Stat(%q): %s", dirname, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is a file", dirname)
	}
}

func requireFileExists(t *testing.T, filename string) {
	info, err := os.Stat(filename) // follow symlinks unlike assert.FileExists
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("unable to find file %q", filename)
		}
		t.Fatalf("error when running os.Stat(%q): %s", filename, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory", filename)
	}
}
