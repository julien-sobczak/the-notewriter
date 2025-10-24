package filesystem

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFileIfNotExists copies a file from src to dst, creating any necessary
// directories. If dst already exists, it does nothing and returns nil.
func CopyFileIfNotExists(src, dst string) error {
	// Check if destination already exists
	if _, err := os.Stat(dst); err == nil {
		return nil // File already exists, nothing to do
	}

	// Create destination directory if needed
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy contents
	_, err = io.Copy(destFile, sourceFile)
	return err
}
