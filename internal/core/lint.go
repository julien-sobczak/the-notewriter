package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/resync"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"golang.org/x/exp/slices"
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

	// No extension in wikilinks
	"no-extension-wikilink": {
		Eval: NoExtensionWikilink,
	},

	// No ambigiuty in wikilinks
	"no-ambiguous-wikilink": {
		Eval: NoAmbiguousWikilink,
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

/* Keep an inventory of all Markdown sections to determine easily if a wikilink is dead.  */

var sectionsInventory map[string][]string // path without extension => section titles (without the leading characters)
var sectionsInventoryOnce resync.Once     // Build the inventory on first occurrence only.

func buildSectionsInventory() {
	sectionsInventory = make(map[string][]string)
	CurrentCollection().walk(CurrentConfig().RootDirectory, func(path string, stat fs.FileInfo) error {
		relativePath, err := CurrentCollection().GetFileRelativePath(path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Extract all sections
		var sections []string
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if ok, longTitle, _ := markdown.IsHeading(line); ok {
				sections = append(sections, longTitle)
			}
		}

		sectionsInventory[text.TrimExtension(relativePath)] = sections

		return nil
	})
}

// NoDeadWikilink implements the rule "no-dead-wikilink".
func NoDeadWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
	sectionsInventoryOnce.Do(buildSectionsInventory)

	var violations []*Violation

	wikilinks := ParseWikilinks(file.Body)
	for _, wikilink := range wikilinks {
		foundPath := false

		searchedPath := text.TrimExtension(wikilink.Path())
		if wikilink.Anchored() {
			searchedPath = text.TrimExtension(file.RelativePath)
		}

		for path, sections := range sectionsInventory {
			if strings.HasSuffix(path, searchedPath) {
				// found the link
				foundPath = true

				if wikilink.Section() != "" && !slices.Contains(sections, wikilink.Section()) {
					violations = append(violations, &Violation{
						RelativePath: file.RelativePath,
						Message:      fmt.Sprintf("section not found for wikilink %s", wikilink),
						Line:         wikilink.Line,
					})
				}
			}
		}
		if !foundPath {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("file not found for wikilink %s", wikilink),
				Line:         wikilink.Line,
			})

		}
	}

	return violations, nil
}

// NoExtensionWikilink implements the rule "no-extension-wikilink".
func NoExtensionWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	wikilinks := ParseWikilinks(file.Body)
	for _, wikilink := range wikilinks {
		if wikilink.ContainsExtension() {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("extension found in wikilink %s", wikilink),
				Line:         wikilink.Line,
			})
		}
	}

	return violations, nil
}

// NoAmbiguousWikilink implements the rule "no-ambiguous-wikilink"
func NoAmbiguousWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
	sectionsInventoryOnce.Do(buildSectionsInventory)

	var violations []*Violation

	wikilinks := ParseWikilinks(file.Body)
	for _, wikilink := range wikilinks {
		foundMatchingPaths := 0

		searchedPath := text.TrimExtension(wikilink.Path())
		if wikilink.Anchored() {
			searchedPath = text.TrimExtension(file.RelativePath)
		}

		for path := range sectionsInventory {
			if strings.HasSuffix(path, searchedPath) {
				// potentially found the link
				foundMatchingPaths += 1
			}
		}

		if foundMatchingPaths > 1 {
			violations = append(violations, &Violation{
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("ambiguous reference for wikilink %s", wikilink),
				Line:         wikilink.Line,
			})

		}
	}

	return violations, nil
}
