package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notetaker/internal/testutil"
	"github.com/stretchr/testify/require"
)

// SetUpCollectionFromGoldenFile populates a temp directory containing a valid .nt collection and a single file.
func SetUpCollectionFromGoldenFile(t *testing.T) string {
	return SetUpCollectionFromGoldenFileNamed(t, t.Name()+".md")
}

// SetUpCollectionFromGoldenFileNamed populates a temp directory based on the given golden file name.
func SetUpCollectionFromGoldenFileNamed(t *testing.T, testname string) string {
	filename := testutil.SetUpFromGoldenFileNamed(t, testname)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpCollectionFromFileContent populates a temp directory based on the given file content.
func SetUpCollectionFromFileContent(t *testing.T, name, content string) string {
	filename := testutil.SetUpFromFileContent(t, name, content)
	dirname := filepath.Dir(filename)
	configureDir(t, dirname)
	return filename
}

// SetUpCollectionFromGoldenDir populates a temp directory containing a valid .nt collection.
func SetUpCollectionFromGoldenDir(t *testing.T) string {
	return SetUpCollectionFromGoldenDirNamed(t, t.Name())
}

// SetUpCollectionFromGoldenDir populates a temp directory based on the given golden dir name.
func SetUpCollectionFromGoldenDirNamed(t *testing.T, testname string) string {
	dirname := testutil.SetUpFromGoldenDirNamed(t, testname)
	configureDir(t, dirname)
	return dirname
}

// SetUpCollectionFromTempDir populates a temp directory containing a valid .nt collection.
func SetUpCollectionFromTempDir(t *testing.T) string {
	dirname := t.TempDir()
	configureDir(t, dirname)
	return dirname
}

func configureDir(t *testing.T, dirname string) {
	ntDir := filepath.Join(dirname, ".nt")
	if _, err := os.Stat(ntDir); os.IsNotExist(err) {
		// Create a default configuration if not exists for CurrentConfig() to work
		if err := os.Mkdir(ntDir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ntDir, "config"), []byte(`[core]
extensions=["md", "markdown"]`), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}
	// Force the application to consider the temporary directory as the home
	os.Setenv("NT_HOME", dirname)
	t.Cleanup(func() {
		os.Unsetenv("NT_HOME")
		Reset()
	})
}

/* Test */

func TestSetUpCollectionFromGoldenDirNamed(t *testing.T) {
	dirname := SetUpCollectionFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md"))
}
