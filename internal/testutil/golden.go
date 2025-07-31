package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// DuplicateDirHierarchy duplicates a directory hierarchy, copying symlinks as their target files.
func DuplicateDirHierarchy(t *testing.T, srcDir, destDir string) {
	// Ensure the destination directory exists
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("failed to create destination directory %s: %v", destDir, err)
	}

	// Walk through the source directory
	err = filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %v", srcPath, err)
		}

		// Determine the destination path
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %v", err)
		}
		destPath := filepath.Join(destDir, relPath)

		// Handle directories
		if info.IsDir() {
			err := os.MkdirAll(destPath, info.Mode())
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %v", destPath, err)
			}
			return nil
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %v", srcPath, err)
			}

			// Resolve the symlink target and copy the target file
			absTarget := target
			if !filepath.IsAbs(target) {
				absTarget = filepath.Join(filepath.Dir(srcPath), target)
			}

			// Check if the symlink target is a directory
			targetInfo, err := os.Stat(absTarget)
			if err != nil {
				return fmt.Errorf("failed to stat symlink target %s: %v", absTarget, err)
			}

			if targetInfo.IsDir() {
				// Recursively copy the directory contents
				DuplicateDirHierarchy(t, absTarget, destPath)
			} else {
				// Copy the target file
				input, err := os.ReadFile(absTarget)
				if err != nil {
					return fmt.Errorf("failed to read symlink target %s: %v", absTarget, err)
				}
				err = os.WriteFile(destPath, input, info.Mode())
				if err != nil {
					return fmt.Errorf("failed to copy symlink target %s to %s: %v", absTarget, destPath, err)
				}
			}
			return nil
		}

		// Copy the regular file
		input, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", srcPath, err)
		}
		err = os.WriteFile(destPath, input, info.Mode())
		if err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %v", srcPath, destPath, err)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("failed to duplicate directory hierarchy: %v", err)
	}
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
