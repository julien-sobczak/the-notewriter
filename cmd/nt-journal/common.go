package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

// GetJournalPath returns the absolute path to the journal entry for the given date
func GetJournalPath(date time.Time) string {
	year, month, day := date.Date()
	dirPath := filepath.Join(core.CurrentConfig().RootDirectory, "journal", fmt.Sprintf("%04d", year)) // TODO use Config instead
	filePath := filepath.Join(dirPath, fmt.Sprintf("%04d-%02d-%02d.md", year, month, day))
	return filePath
}

// CreateJournalEntryIfMissing creates the journal entry for today.
func CreateJournalEntryIfMissing(date time.Time) (string, error) {
	year, month, day := date.Date()
	entryPath := GetJournalPath(date)
	entryDir := filepath.Dir(entryPath)

	// Create the directory hierarchy
	if err := os.MkdirAll(entryDir, 0750); err != nil {
		return "", err
	}

	// Create the file is missing
	if _, err := os.Stat(entryPath); errors.Is(err, os.ErrNotExist) {
		// Today entry doesn't exist
		fileContent := fmt.Sprintf("# Journal: %04d-%02d-%02d\n", year, month, day)
		if err := os.WriteFile(entryPath, []byte(fileContent), 0660); err != nil {
			return "", err
		}
	}

	return entryPath, nil
}

// AppendToJournal completes a journal entry with a new Markdown block
func AppendToJournal(sectionTitle string, sectionContent string) error {
	today := time.Now()
	entryPath := GetJournalPath(today)

	f, err := os.OpenFile(entryPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Unable to open file %q: %v", entryPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("\n## %s\n\n%s\n", sectionTitle, sectionContent)); err != nil {
		log.Fatalf("Unable to write to file %q: %v", entryPath, err)
	}
	return nil
}

func OpenInEditor(entryPath string) error {
	workspacePath := core.CurrentConfig().RootDirectory

	// Check env variable $EDITOR
	editor, ok := os.LookupEnv("EDITOR")
	var cmdStr string
	if ok && editor != "code" {
		// Default to $EDITOR
		cmdStr = fmt.Sprintf("$EDITOR %s", entryPath)
	} else {
		// Default to opening the workspace with VS Code
		cmdStr = fmt.Sprintf("code %s -g %s", workspacePath, entryPath)
	}

	// Open the journal entry file
	cmd := exec.Command("sh", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// GenerateTodaySymlink creates or updates the symlink journal/today.md
func GenerateTodaySymlink(entryPath string) error {
	todayPath := filepath.Join(core.CurrentConfig().RootDirectory, "journal", "today.md")
	if err := os.Symlink(entryPath, todayPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			if err := os.Remove(todayPath); err != nil {
				return err
			}
			if err := os.Symlink(entryPath, todayPath); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	// TODO generate another symlink "yesterday.md"?
	return nil
}

/* Helpers */

func haveCommonElements(slice1, slice2 []string) bool {
	for _, elem := range slice1 {
		if slices.Contains(slice2, elem) {
			return true
		}
	}
	return false
}

// ContainsMarkdownSection checks if a file contains a heading contains the given substring
func ContainsMarkdownSection(filePath, sectionTitle string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "##") && strings.Contains(line, sectionTitle) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}
