package main

import (
	"os"
	"time"
)

// GetEditor returns the editor to use (from $EDITOR or default to vi)
func GetEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return editor
}

// GetPager returns the pager to use (from $PAGER or default to less)
func GetPager() string {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}
	return pager
}

// CreateTempFile creates a temporary file with the given content
func CreateTempFile(content []byte, suffix string) (string, error) {
	tmpFile, err := os.CreateTemp("", "nt-vault-*"+suffix)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// GetFileModTime returns the modification time of a file
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
