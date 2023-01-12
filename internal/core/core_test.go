package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notetaker/internal/testutil"
)

// SetUpCollectionFromGoldenDir populates a temp directory containing a valid .nt collection.
func SetUpCollectionFromGoldenDir(t *testing.T) string {
	return SetUpCollectionFromGoldenDirNamed(t, t.Name())
}

// SetUpCollectionFromGoldenDir populates a temp directory based on the given golden dir name.
func SetUpCollectionFromGoldenDirNamed(t *testing.T, testname string) string {
	dirname := testutil.SetUpFromGoldenDir(t)

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

	return dirname
}
