package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

type LintResult struct {
	AnalyzedFiles int
	AffectedFiles int
	Warnings      []*Violation
	Errors        []*Violation
}

type Violation struct {
	// The human-readable description of the violation
	Message string
	// The relative path to the file containing the violation
	RelativePath string
	// The line number in the file containing the violation
	Line int
}

type LintRuleDefinition struct {
	Eval LintRule
}

// LintRule describes the interface that rules must conform.
type LintRule func(*ParsedFile, []string) ([]*Violation, error)

var LintRules = map[string]LintRuleDefinition{
	// Enforce no duplicate between note titles
	"no-duplicate-note-title": {
		Eval: NoDuplicateNoteTitle,
	},

	// Enforce a minimum number of lines between notes
	"min-lines-between-notes": {
		Eval: MinLinesBetweenNotes,
	},

	// Enforce a consistent naming for notes
	"note-title-match": {
		Eval: NoteTitleMatch,
	},

	// Forbid untyped notes
	"no-free-note": {
		Eval: NoFreeNote,
	},

	// Path to media files must exist
	"no-dangling-media": {
		Eval: NoDanglingMedia,
	},

	// Links between notes must exist
	"no-dead-wikilink": {
		Eval: NoDeadWikilink,
	},

	// No extension in wikilink
	"no-extension-wikilink": {
		Eval: NoExtensionWikilink,
	},
}

// NoDuplicateNoteTitle implements the rule "no-duplicate-note-title".
func NoDuplicateNoteTitle(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	uniqueNoteTitles := make(map[string]bool)
	notes := ParseNotes(file.Body)
	for _, note := range notes {
		if _, ok := uniqueNoteTitles[note.LongTitle]; ok {
			violations = append(violations, &Violation{
				Message:      fmt.Sprintf("duplicated note with title %q", note.ShortTitle),
				RelativePath: file.RelativePath,
				Line:         note.Line,
			})
		} else {
			uniqueNoteTitles[note.LongTitle] = true
		}
	}

	return violations, nil
}

// MinLinesBetweenNotes implements the rule "min-lines-between-notes".
func MinLinesBetweenNotes(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}
	minLines, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, fmt.Errorf("argument %s must be an integer", args[0])
	}

	body := file.Body
	lines := strings.Split(body, "\n")

	notes := ParseNotes(body)
	for i, note := range notes {
		if i == 0 {
			// No need to check space before the first note. Only between successive notes
			continue
		}

		for j := 1; j <= minLines; j++ {
			lineNumber := note.Line - j
			lineIndex := lineNumber - 1
			if lineIndex < 0 || !text.IsBlank(lines[lineIndex]) {
				violations = append(violations, &Violation{
					RelativePath: file.RelativePath,
					Message:      fmt.Sprintf("missing blank lines before note %q", note.LongTitle),
					Line:         note.Line,
				})
			}
		}
	}

	return violations, nil
}

// NoteTitleMatch implements the rule "note-title-match".
func NoteTitleMatch(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}
	re, err := regexp.Compile(args[0])
	if err != nil {
		return nil, fmt.Errorf("argument %s must be a valid regular expression", args[0])
	}

	body := file.Body

	notes := ParseNotes(body)
	for _, note := range notes {
		if note.Kind == KindFree {
			// Free notes can used any syntax
			continue
		}
		if !re.MatchString(note.LongTitle) {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("note title %q does not match regex %q", note.LongTitle, args[0]),
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// NoFreeNote implements the rule "no-free-note".
func NoFreeNote(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	notes := ParseNotes(file.Body)
	for _, note := range notes {
		if note.Kind == KindFree {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("free note %q not allowed", note.LongTitle),
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// NoDanglingMedia implements the rule "no-dangling-media".
func NoDanglingMedia(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	medias := ParseMedias(file.RelativePath, file.Body)
	for _, media := range medias {
		_, err := os.Stat(media.AbsolutePath)
		if errors.Is(err, os.ErrNotExist) {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("dangling media %s detected in %s", media.RawPath, file.RelativePath),
				Line:         file.BodyLine + media.Line - 1,
			})
		}
	}

	return violations, nil
}

// NoDeadWikilink implements the rule "no-dead-wikilink".
func NoDeadWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
	// TODO now implement
	return nil, nil
}

// NoExtensionWikilink implements the rule "no-extension-wikilink".
func NoExtensionWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
	// TODO now implement
	return nil, nil
}
