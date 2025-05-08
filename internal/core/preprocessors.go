package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
)

func init() {
	RegisterPreprocessor("date-extractor", DateExtractorPreprocessor)
	RegisterPreprocessor("quote-rewriter", QuoteRewriterPreprocessor)
	RegisterPreprocessor("generator", GeneratorPreprocessor)
	RegisterPreprocessor("flashcard-extractor", FlashcardExtractorPreprocessor)
}

/* Preprocessors implementation */

// DateExtractorPreprocessor extracts the date from the note title and sets it as an attribute.
func DateExtractorPreprocessor(file *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	if date, ok := ExtractDateFromTitle(note.Title); ok {
		note.Attributes.SetIfMissing("date", date)
		note.NoteAttributes.SetIfMissing("date", date)
	}
	return nil, nil
}

// ExtractDateFromTitle extracts a date from the title of a note.
// Several patterns matching common US date formats are supported.
func ExtractDateFromTitle(input markdown.Document) (string, bool) {
	// Define common date formats
	formats := []string{
		"2006-01-02", // "2006-01-02"
		"2006-01",    // "2006-01"
		"2006",       // "2006"
	}

	// Regular expression to find potential date substrings
	dateRegex := regexp.MustCompile(`\b\d{4}(-\d{2})?(-\d{2})?\b`)

	// Find all potential date substrings
	matches := dateRegex.FindAllString(input.String(), -1)

	// Try parsing each match with the common date formats
	for _, match := range matches {
		for _, format := range formats {
			if _, err := time.Parse(format, match); err == nil {
				if match != "" {
					return match, true
				}
			}
		}
	}

	// Return false if no valid date is found
	return "", false
}

// QuoteRewriterPreprocessor rewrites the quotes in the note body to use the Markdown syntax (sugar syntax).
func QuoteRewriterPreprocessor(file *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	lines := strings.Split(note.Body.String(), "\n")
	var rewrittenLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !OnlyTagsAndAttributes(trimmed) && !strings.HasPrefix(trimmed, ">") {
			rewrittenLines = append(rewrittenLines, "> "+line)
		} else {
			rewrittenLines = append(rewrittenLines, line)
		}
	}

	note.Body = markdown.Document(strings.Join(rewrittenLines, "\n"))
	return []*ParsedNote{note}, nil
}

// GeneratorPreprocessor executes a script to generate new notes.
// This generator is useful for automation when creating many similar notes.
func GeneratorPreprocessor(file *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	// Generator notes are not saved in database.
	// They are parsed, evaluated and the results is injected as if the generated notes had been edited manually.

	// Inline or external?
	filename := note.Attributes.CastValueAsString("file")
	interpreter := note.Attributes.CastValueAsString("interpreter")

	var cmdArgs []string

	if interpreter != "" {
		// Check binary exists...
		if !CommandExists(interpreter) {
			return nil, fmt.Errorf("interpreter %q doesn't exist in generator %q", interpreter, note.ShortTitle)
		}

		cmdArgs = append(cmdArgs, interpreter)
	} else if filename != "" { // External
		scriptPath := filepath.Join(filepath.Dir(file.Markdown.AbsolutePath), filename)

		// Check file exists
		scriptStat, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("script %q doesn't exist in generator %q", filename, note.ShortTitle)
		}

		// check file is executable
		if interpreter == "" && !IsExec(scriptStat.Mode()) {
			return nil, fmt.Errorf("script %q is not executable in generator %q", filename, note.ShortTitle)
		}

		cmdArgs = append(cmdArgs, scriptPath)
	} else { // Internal

		// Search for the first code block in note
		codeBlocks := note.Body.ExtractCodeBlocks()
		if len(codeBlocks) == 0 {
			return nil, fmt.Errorf("missing 'file' attribute or code block inside generator %q", note.ShortTitle)
		}

		script := codeBlocks[0]

		scriptLanguage := script.Language
		scriptContent := script.Source

		if scriptLanguage == "" {
			return nil, fmt.Errorf("missing language in code block inside generator %q", note.ShortTitle)
		}

		// Expect the Markdown language
		cmdArgs = append(cmdArgs, scriptLanguage)

		scriptPath, err := os.CreateTemp("", "ntscript")
		if err != nil {
			return nil, fmt.Errorf("unable to create temporary script for generator %q: %w", note.ShortTitle, err)
		}
		defer os.Remove(scriptPath.Name())
		os.WriteFile(scriptPath.Name(), []byte(scriptContent), 0755)

		cmdArgs = append(cmdArgs, scriptPath.Name())
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", stderr.Bytes())
		return nil, fmt.Errorf("failed to run generator command %q: %w", strings.Join(cmdArgs, " "), err)
	}

	mdPath, err := os.CreateTemp("", "ntmd")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary Markdown file for generator %q: %w", note.ShortTitle, err)
	}
	defer os.Remove(mdPath.Name())
	if err := os.WriteFile(mdPath.Name(), stdout.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("unable to write temporary Markdown file for generator %q: %w", note.ShortTitle, err)
	}

	mdFile, err := markdown.ParseFile(mdPath.Name())
	if err != nil {
		return nil, err
	}
	generatedFile, err := ParseFile(mdFile, nil)
	if err != nil {
		return nil, err
	}

	// Use original line number to make easy to jump to the generator note
	for _, generatedNote := range generatedFile.Notes {
		generatedNote.Line = note.Line
	}

	return generatedFile.Notes, nil
}

// FlashcardExtractorPreprocessor extracts the flashcard to enrich a note. This preprocessor returns the same input note.
func FlashcardExtractorPreprocessor(file *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	// Only front/back to parse
	parts := note.Body.SplitByHorizontalRules()
	if len(parts) < 2 {
		return nil, errors.New("missing flashcard separator")
	}
	if len(parts) > 2 {
		return nil, errors.New("too many flashcard separator")
	}
	front := parts[0]
	back := parts[1]

	// Simply enrich the current note with the flashcard
	note.Flashcard = &ParsedFlashcard{
		ShortTitle: note.ShortTitle,
		Slug:       note.Slug,
		Front:      front,
		Back:       back,
	}

	return []*ParsedNote{note}, nil
}

/* Helpers */

// CommandExists checks if a command exists in the system's PATH.
func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
