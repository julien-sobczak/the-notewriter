package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// SetUpFromGoldenFile creates a temp file based on the golden file of the current test.
// The file must exist in directory testdata/.
func SetUpFromGoldenFile(t *testing.T) string {
	return SetUpFromGoldenFileNamed(t, t.Name()+".md")
}

// SetUpFromGoldenFileNamed createa a temp file based on the given golden file name.
func SetUpFromGoldenFileNamed(t *testing.T, filename string) string {
	dir := t.TempDir()

	fileIn := filepath.Join("testdata", filename)
	stat, err := os.Lstat(fileIn)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile(fileIn)
	if err != nil {
		t.Fatal(err)
	}

	fileOut := filepath.Join(dir, filename)
	err = os.WriteFile(fileOut, in, stat.Mode())
	if err != nil {
		t.Fatal(err)
	}

	return fileOut
}

// SetUpFromFileContent createa a temp file based on the given file content.
func SetUpFromFileContent(t *testing.T, filename string, content string) string {
	dir := t.TempDir()

	fileOut := filepath.Join(dir, filename)
	err := os.WriteFile(fileOut, []byte(content), 0755)
	if err != nil {
		t.Fatal(err)
	}

	return fileOut
}

// SetUpFromGoldenDir populates a temp directory based on the given test name.
func SetUpFromGoldenDir(t *testing.T) string {
	return SetUpFromGoldenDirNamed(t, t.Name())
}

// SetUpFromGoldenDir populates a temp directory based on the given golden dir name.
func SetUpFromGoldenDirNamed(t *testing.T, testname string) string {
	dir := t.TempDir()

	dirIn := filepath.Join("testdata", testname)
	dirOut := filepath.Join(dir, testname)

	// Symlink the golden directory
	absolutePath, err := filepath.Abs(dirIn)
	if err != nil {
		t.Fatal(err)
	}
	os.Symlink(absolutePath, dirOut)

	return dirOut
}

// GoldenFile reads the content of the golden file of the current test.
func GoldenFile(t *testing.T) []byte {
	return GoldenFileNamed(t, t.Name()+".md")
}

// GoldenFileNamed reads the content of the given golden file.
func GoldenFileNamed(t *testing.T, filename string) []byte {
	path := filepath.Join("testdata", filename)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading golden file %s: %v", path, err)
	}
	return b
}
