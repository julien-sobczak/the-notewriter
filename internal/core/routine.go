package core

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
)

// EvaluateJournalPath evaluates a path or content template using date functions
// (year, month, day, date) and returns the resulting string.
func EvaluateJournalPath(path string) (string, error) {
	tmpl, err := template.New("path").Funcs(journalDateFuncs()).Parse(path)
	if err != nil {
		return "", fmt.Errorf("parsing path template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("evaluating path template: %w", err)
	}
	return buf.String(), nil
}

// journalDateFuncs returns a template.FuncMap with date functions for journal path evaluation.
func journalDateFuncs() template.FuncMap {
	return template.FuncMap{
		"year":  func() string { return clock.Now().Format("2006") },
		"month": func() string { return clock.Now().Format("01") },
		"day":   func() string { return clock.Now().Format("02") },
		"date":  func() string { return clock.Now().Format("2006-01-02") },
	}
}

// GenerateRoutineContent evaluates the routine template with custom functions and
// returns the generated content ready for the user to edit.
func GenerateRoutineContent(routine *ConfigRoutine) (string, error) {
	funcMap := template.FuncMap{
		"input": func() string {
			return "_Your Answer_"
		},
		"morningpages": func() string {
			return "_Take a moment to clear your mind by capturing your unfiltered thoughts, feelings, and morning reflections before the day begins._"
		},
		"affirmation": func(wikilink string) (string, error) {
			return randomItemText(wikilink)
		},
		"prompt": func(wikilink string) (string, error) {
			text, err := randomItemText(wikilink)
			if err != nil {
				return "", err
			}
			return text + "\n\n_Write your reflection_", nil
		},
	}

	tmpl, err := template.New("routine").Funcs(funcMap).Parse(routine.Template)
	if err != nil {
		return "", fmt.Errorf("parsing routine template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("evaluating routine template: %w", err)
	}
	return buf.String(), nil
}

// randomItemText picks a random item from the note identified by wikilink and strips
// attributes and tags from the item text.
func randomItemText(wikilink string) (string, error) {
	note, err := CurrentRepository().FindNoteByWikilink(wikilink)
	if err != nil {
		return "", fmt.Errorf("querying note %q: %w", wikilink, err)
	}
	if note == nil {
		return "", fmt.Errorf("note not found for wikilink %q", wikilink)
	}
	if note.Items == nil || len(note.Items.Children) == 0 {
		return "", fmt.Errorf("note %q has no list items", wikilink)
	}

	item := note.Items.Children[rand.Intn(len(note.Items.Children))]
	stripped := item.Text.MustTransform(
		StripTags(),
		StripOnlyAttributes(),
	).TrimSpace()

	return stripped.String(), nil
}

// AppendRoutineToJournal appends the edited routine content to the appropriate journal file.
// If the journal file does not exist it is created and initialized with defaultContent.
// All headings in editedContent are shifted so that template level-1 headings become
// sub-headings (level 3) of the ## RoutineName section header.
// Returns the absolute path of the journal file.
func AppendRoutineToJournal(journal *ConfigJournal, routine *ConfigRoutine, editedContent string) (string, error) {
	// Resolve the journal file path
	relPath, err := EvaluateJournalPath(journal.Path)
	if err != nil {
		return "", fmt.Errorf("evaluating journal path: %w", err)
	}

	absPath := filepath.Join(CurrentConfig().RootDirectory, relPath)

	// Create the file if it doesn't exist
	if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return "", fmt.Errorf("creating journal directory: %w", err)
		}
		defaultContent, err := EvaluateJournalPath(journal.DefaultContent)
		if err != nil {
			return "", fmt.Errorf("evaluating default content: %w", err)
		}
		if err := os.WriteFile(absPath, []byte(defaultContent+"\n"), 0644); err != nil {
			return "", fmt.Errorf("creating journal file: %w", err)
		}
	}

	// Shift headings so template level-1 headings become level-3 sub-headings
	trimmed := strings.TrimSpace(editedContent)
	shiftedContent := markdown.Document(trimmed).MustTransform(markdown.ShiftHeadings(2))

	// Build the section to append (ensure a blank line before the ## header)
	section := fmt.Sprintf("\n## %s\n\n%s\n", routine.Name, shiftedContent)

	// Append to the journal file
	f, err := os.OpenFile(absPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening journal file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(section); err != nil {
		return "", fmt.Errorf("writing to journal file: %w", err)
	}
	return absPath, nil
}

