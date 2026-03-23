package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type LintResult struct {
	AnalyzedFiles int
	AffectedFiles int
	Errors        []*Violation
}

// Append merges new violations into the current result.
func (r *LintResult) Append(violations ...*Violation) {
	for _, violation := range violations {
		r.Errors = append(r.Errors, violation)
	}
}

func (r LintResult) String() string {
	var res strings.Builder
	res.WriteString(fmt.Sprintf("%d invalid files on %d analyzed files (%d errors)\n",
		r.AffectedFiles,
		r.AnalyzedFiles,
		len(r.Errors)))
	for _, violation := range r.Errors {
		res.WriteString(fmt.Sprintf("%s (%s:%d)\n", violation.Message, violation.RelativePath, violation.Line))
	}
	return res.String()
}

type Violation struct {
	// The name of the violation
	Name string
	// The human-readable description of the violation
	Message string
	// The relative path to the file containing the violation
	RelativePath string
	// The line number in the file containing the violation
	Line int
}

func (v Violation) String() string {
	return v.Message
}

// LintRule describes the interface that rules must conform.
type LintRule func(*ParsedFile, *Query, []any) ([]*Violation, error)

var LintRulesFn = map[string]LintRule{
	// Enforce no empty titles
	"no-empty-title": NoEmptyTitle,

	// Enforce no duplicate between note titles
	"no-duplicate-note-title": NoDuplicateNoteTitle,

	// Every slug must be unique
	"no-duplicate-slug": NoDuplicateSlug,

	// Flashcards must have a slug
	"no-implicit-slug-on-flashcard": NoImplicitSlugOnFlashcard,

	// Enforce a minimum number of lines between notes
	"min-lines-between-notes": MinLinesBetweenNotes,

	// Enforce a maximum number of lines between notes
	"max-lines-between-notes": MaxLinesBetweenNotes,

	// Enforce a consistent naming for notes
	"note-title-match": NoteTitleMatch,

	// Path to media files must exist
	"no-dangling-media": NoDanglingMedia,

	// Links between notes must exist
	"no-dead-wikilink": NoDeadWikilink,

	// No extension in wikilinks
	"no-extension-wikilink": NoExtensionWikilink,

	// No ambiguity in wikilinks
	"no-ambiguous-wikilink": NoAmbiguousWikilink,

	// At least one tag is present (must match the optional pattern).
	"require-tag": RequireTag,

	// Flashcards must have an explicit slug attribute
	"require-flashcard-slug": RequireFlashcardSlug,

	// Flashcards must match at least one deck
	"no-orphan-flashcard": NoOrphanFlashcard,

	// Flashcards must not match more than one deck
	"no-overlapping-deck": NoOverlappingDeck,
}

/* Rules */

// NoEmptyTitle implements the rule "no-empty-title".
func NoEmptyTitle(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	if text.IsBlank(file.ShortTitle.String()) {
		violations = append(violations, &Violation{
			Name:         "no-empty-title",
			Message:      "note with empty title",
			RelativePath: file.RelativePath,
			Line:         1, // The title is always on the first line
		})
	}
	for _, note := range file.Notes {
		if text.IsBlank(note.ShortTitle.String()) {
			violations = append(violations, &Violation{
				Name:         "no-empty-title",
				Message:      "note with empty title",
				RelativePath: file.RelativePath,
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// NoDuplicateNoteTitle implements the rule "no-duplicate-note-title".
func NoDuplicateNoteTitle(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	uniqueNoteTitles := make(map[string]bool)
	for _, note := range file.Notes {
		cleanTitle := note.Title.MustTransform(markdown.StripEmphasis()).String()
		if _, ok := uniqueNoteTitles[cleanTitle]; ok {
			violations = append(violations, &Violation{
				Name:         "no-duplicate-note-title",
				Message:      fmt.Sprintf("duplicated note with title %q", note.ShortTitle),
				RelativePath: file.RelativePath,
				Line:         note.Line,
			})
		} else {
			uniqueNoteTitles[cleanTitle] = true
		}
	}

	return violations, nil
}

// Keep an inventory of all slugs to easily determine if a slug is unique
var slugInventory map[string]bool // slug => true
var slugInventoryOnce resync.Once // Build the inventory on first occurrence only.

// NoDuplicateSlug implements the rule "no-duplicate-slug".
func NoDuplicateSlug(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	slugInventoryOnce.Do(func() {
		slugInventory = make(map[string]bool)
	})

	var violations []*Violation

	for _, note := range file.Notes {
		// Check if not already in use
		if _, ok := slugInventory[note.Slug]; ok {
			violations = append(violations, &Violation{
				Name:         "no-duplicate-slug",
				Message:      fmt.Sprintf("duplicated slug %q", note.Slug),
				RelativePath: file.RelativePath,
				Line:         note.Line,
			})
		} else {
			if markdown.Slug(note.Slug) != note.Slug {
				// Slug does not match the expected format
				// (important to use slug in URLs)
				violations = append(violations, &Violation{
					Name:         "no-duplicate-slug",
					Message:      fmt.Sprintf("invalid slug format %q", note.Slug),
					RelativePath: file.RelativePath,
					Line:         note.Line,
				})
			}
			slugInventory[note.Slug] = true
		}
	}

	return violations, nil
}

// NoImplicitSlugOnFlashcard implements the rule "enforce-flashcard-slug".
func NoImplicitSlugOnFlashcard(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	for _, note := range file.Notes {
		if len(note.Flashcards) > 0 {
			if _, ok := note.NoteAttributes.Slug(); !ok {
				violations = append(violations, &Violation{
					Name:         "no-implicit-slug-on-flashcard",
					Message:      "flashcard must have a slug",
					RelativePath: file.RelativePath,
					Line:         note.Line,
				})

			}
		}
	}

	return violations, nil
}

// MinLinesBetweenNotes implements the rule "min-lines-between-notes".
func MinLinesBetweenNotes(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}

	minLines, err := IntFromJSON(args[0])
	if err != nil {
		return nil, err
	}

	lines := file.Markdown.Lines()

	for i, note := range file.Notes {
		if note.Generated {
			// Skip generated notes as they don't exist in the original file
			// and thus can't be expected to have blank lines before them.
			continue
		}
		if i == 0 {
			// No need to check space before the first note. Only between successive notes
			continue
		}

		for j := 1; j <= minLines; j++ {
			lineNumber := note.Line - j
			lineIndex := lineNumber - 1
			if lineIndex < 0 || !text.IsBlank(lines[lineIndex]) {
				violations = append(violations, &Violation{
					Name:         "min-lines-between-notes",
					RelativePath: file.RelativePath,
					Message:      fmt.Sprintf("missing blank lines before note %q", note.Title),
					Line:         note.Line,
				})
			}
		}
	}

	return violations, nil
}

// MaxLinesBetweenNotes implements the rule "min-lines-between-notes".
func MaxLinesBetweenNotes(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}

	maxLines, err := IntFromJSON(args[0])
	if err != nil {
		return nil, err
	}

	lines := file.Markdown.Lines()

	for _, note := range file.Notes {
		if note.Generated {
			// Skip generated notes as they don't exist in the original file
			// and thus can't be expected to have blank lines before them.
			continue
		}
		countBlankLinesBefore := 0

		j := 1
		for {
			lineNumber := note.Line - j
			lineIndex := lineNumber - 1
			if lineIndex < 0 {
				break
			}
			if text.IsBlank(lines[lineIndex]) {
				countBlankLinesBefore++
			} else {
				break
			}

			j++
		}

		if countBlankLinesBefore > maxLines {
			violations = append(violations, &Violation{
				Name:         "max-lines-between-notes",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("too many blank lines before note %q", note.Title),
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// NoteTitleMatch implements the rule "note-title-match".
func NoteTitleMatch(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}
	argStr, ok := args[0].(string)
	if !ok {
		return nil, errors.New("argument must be a string")
	}
	re, err := regexp.Compile(argStr)
	if err != nil {
		return nil, fmt.Errorf("argument %s must be a valid regular expression", args[0])
	}

	for _, note := range file.Notes {
		if !re.MatchString(string(note.Title)) {
			violations = append(violations, &Violation{
				Name:         "note-title-match",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("note title %q does not match regex %q", note.Title, args[0]),
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// NoDanglingMedia implements the rule "no-dangling-media".
func NoDanglingMedia(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	for _, media := range file.Medias {
		_, err := os.Stat(media.AbsolutePath)
		if errors.Is(err, os.ErrNotExist) {
			violations = append(violations, &Violation{
				Name:         "no-dangling-media",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("dangling media %s detected in %s", media.RawPath, file.RelativePath),
				Line:         file.FileLineNumber(media.Line),
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
	pathSpecs := []PathSpec{"."}
	err := CurrentRepository().Walk(pathSpecs, func(md *markdown.File) error {
		relativePath := CurrentRepository().GetFileRelativePath(md.AbsolutePath)

		// Extract all sections
		var sections []string
		md.WalkSections(func(parent, current *markdown.Section, children []*markdown.Section) error {
			sections = append(sections, string(current.HeadingText))
			return nil
		})
		// Use a leading / to only match full filename
		// Ex: "productivity#Note: XXX" is not ambiguous if files productivity.md and on-productivity.md exist
		sectionsInventory["/"+text.TrimExtension(relativePath)] = sections

		return nil
	})
	if err != nil {
		log.Fatalf("Unable to build sections inventory: %v", err)
	}
}

// NoDeadWikilink implements the rule "no-dead-wikilink".
func NoDeadWikilink(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	sectionsInventoryOnce.Do(buildSectionsInventory)

	var violations []*Violation

	for _, wikilink := range file.Wikilinks {
		foundPath := false

		searchedPath := text.TrimExtension(wikilink.Path())
		if wikilink.Anchored() {
			searchedPath = text.TrimExtension(file.RelativePath)
		}

		for path, sections := range sectionsInventory {
			if strings.HasSuffix(path, "/"+searchedPath) { // Match full filename
				// found the link
				foundPath = true

				if wikilink.Section() != "" && !slices.Contains(sections, wikilink.Section()) {
					violations = append(violations, &Violation{
						Name:         "no-dead-wikilink",
						RelativePath: file.RelativePath,
						Message:      fmt.Sprintf("section not found for wikilink %s", wikilink),
						Line:         file.FileLineNumber(wikilink.Line),
					})
				}
			}
		}
		if !foundPath {
			violations = append(violations, &Violation{
				Name:         "no-dead-wikilink",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("file not found for wikilink %s", wikilink),
				Line:         file.FileLineNumber(wikilink.Line),
			})

		}
	}

	return violations, nil
}

// NoExtensionWikilink implements the rule "no-extension-wikilink".
func NoExtensionWikilink(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	for _, wikilink := range file.Wikilinks {
		if wikilink.ContainsExtension() {
			violations = append(violations, &Violation{
				Name:         "no-extension-wikilink",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("extension found in wikilink %s", wikilink),
				Line:         file.FileLineNumber(wikilink.Line),
			})
		}
	}

	return violations, nil
}

// NoAmbiguousWikilink implements the rule "no-ambiguous-wikilink"
func NoAmbiguousWikilink(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	sectionsInventoryOnce.Do(buildSectionsInventory)

	var violations []*Violation

	for _, wikilink := range file.Wikilinks {
		foundMatchingPaths := 0

		searchedPath := text.TrimExtension(wikilink.Path())
		if wikilink.Anchored() {
			searchedPath = text.TrimExtension(file.RelativePath)
		}

		for path := range sectionsInventory {
			if strings.HasSuffix(path, "/"+searchedPath) { // Match full filename
				// potentially found the link
				foundMatchingPaths += 1
			}
		}

		if foundMatchingPaths > 1 {
			violations = append(violations, &Violation{
				Name:         "no-ambiguous-wikilink",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("ambiguous reference for wikilink %s", wikilink),
				Line:         file.FileLineNumber(wikilink.Line),
			})

		}
	}

	return violations, nil
}

// RequireTag implements the rule "require-tag"
func RequireTag(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	if len(args) > 1 {
		return nil, errors.New("only a single argument is allowed")
	}
	regexPattern := regexp.MustCompile(".*")
	if len(args) == 1 {
		regexStr, ok := args[0].(string)
		if !ok {
			return nil, errors.New("argument must be a string")
		}
		regexArgument, err := regexp.Compile(regexStr)
		if err != nil {
			return nil, fmt.Errorf("argument %s must be a valid regular expression", args[0])
		}
		regexPattern = regexArgument
	}

	for _, note := range file.Notes {
		if !note.Matches(query) {
			continue
		}

		atLeastOneTagMatch := false
		for _, tag := range note.Attributes.Tags() {
			if regexPattern.MatchString(tag) {
				atLeastOneTagMatch = true
				break
			}
		}

		if !atLeastOneTagMatch {
			violations = append(violations, &Violation{
				Name:         "require-tag",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("note %q does not have tags", note.Title),
				Line:         note.Line,
			})
		}
	}

	return violations, nil
}

// RequireFlashcardSlug implements the rule "require-flashcard-slug".
// Note: This rule has the same behavior as NoImplicitSlugOnFlashcard but with a different name
// as requested in the issue requirements.
func RequireFlashcardSlug(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	var violations []*Violation

	for _, note := range file.Notes {
		if len(note.Flashcards) > 0 {
			if _, ok := note.NoteAttributes.Slug(); !ok {
				violations = append(violations, &Violation{
					Name:         "require-flashcard-slug",
					Message:      "flashcard must have an explicit slug attribute",
					RelativePath: file.RelativePath,
					Line:         note.Line,
				})
			}
		}
	}

	return violations, nil
}

// Keep an inventory of parsed deck queries to avoid reparsing for every file
var deckQueriesInventory []*Query
var deckQueriesInventoryOnce resync.Once

func buildDeckQueriesInventory() {
	deckQueriesInventory = make([]*Query, 0)
	decks := CurrentConfigFile().Decks
	for _, deck := range decks {
		if deck.Query != "" {
			deckQuery, err := ParseQuery(deck.Query)
			if err != nil {
				log.Printf("Warning: failed to parse deck query %q: %v", deck.Query, err)
				continue
			}
			deckQueriesInventory = append(deckQueriesInventory, deckQuery)
		}
	}
}

// NoOrphanFlashcard implements the rule "no-orphan-flashcard".
func NoOrphanFlashcard(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	deckQueriesInventoryOnce.Do(buildDeckQueriesInventory)

	var violations []*Violation

	// Check each note with flashcards
	for _, note := range file.Notes {
		if len(note.Flashcards) > 0 {
			// Skip if the flashcard has the "suspended" tag
			if note.NoteTags.Includes("suspended") {
				continue
			}

			// Check if the note matches at least one deck query
			matchesAnyDeck := false
			for _, deckQuery := range deckQueriesInventory {
				if deckQuery.MatchesParsed(note) {
					matchesAnyDeck = true
					break
				}
			}

			if !matchesAnyDeck {
				violations = append(violations, &Violation{
					Name:         "no-orphan-flashcard",
					Message:      fmt.Sprintf("flashcard %q does not match any deck", note.Title),
					RelativePath: file.RelativePath,
					Line:         note.Line,
				})
			}
		}
	}

	return violations, nil
}

// deckNamedQuery associates a deck name with its parsed query.
type deckNamedQuery struct {
	name  string
	query *Query
}

// Keep an inventory of named deck queries to avoid reparsing for every file
var overlappingDeckInventory []*deckNamedQuery
var overlappingDeckInventoryOnce resync.Once

func buildOverlappingDeckInventory() {
	overlappingDeckInventory = make([]*deckNamedQuery, 0)
	decks := CurrentConfigFile().Decks
	for _, deck := range decks {
		if deck.Query != "" {
			deckQuery, err := ParseQuery(deck.Query)
			if err != nil {
				log.Printf("Warning: failed to parse deck query %q: %v", deck.Query, err)
				continue
			}
			overlappingDeckInventory = append(overlappingDeckInventory, &deckNamedQuery{
				name:  deck.Name,
				query: deckQuery,
			})
		}
	}
}

// NoOverlappingDeck implements the rule "no-overlapping-deck".
func NoOverlappingDeck(file *ParsedFile, query *Query, args []any) ([]*Violation, error) {
	overlappingDeckInventoryOnce.Do(buildOverlappingDeckInventory)

	var violations []*Violation

	for _, note := range file.Notes {
		if len(note.Flashcards) > 0 {
			// Skip if the flashcard has the "suspended" tag
			if note.NoteTags.Includes("suspended") {
				continue
			}

			// Find all decks that match this note
			var matchingDecks []string
			for _, item := range overlappingDeckInventory {
				if item.query.MatchesParsed(note) {
					matchingDecks = append(matchingDecks, item.name)
				}
			}

			if len(matchingDecks) > 1 {
				violations = append(violations, &Violation{
					Name:         "no-overlapping-deck",
					Message:      fmt.Sprintf("flashcard %q matches multiple decks: %s", note.Title, strings.Join(matchingDecks, ", ")),
					RelativePath: file.RelativePath,
					Line:         note.Line,
				})
			}
		}
	}

	return violations, nil
}

// CheckSchema validates a file against its declared schema if it matches a file type.
func CheckSchema(file *ParsedFile) ([]*Violation, error) {
	var violations []*Violation

	// Check if file matches a file type with a schema
	if file.Type == "" {
		return nil, nil
	}

	fileType := CurrentConfigFile().MustGetFileType(file.Type)
	if fileType.Schema == nil || fileType.Schema.Body == nil {
		// No file type match or no schema defined
		return nil, nil
	}

	// Validate Markdown document against schema
	md := file.Markdown

	root, err := md.GetTopSection()
	if err != nil {
		return nil, err
	}
	sectionViolations := checkMarkdownSection([]*markdown.Section{root}, []*ConfigHeadingSchema{fileType.Schema.Body}, false)
	// Append the file path to each violation
	for _, v := range sectionViolations {
		v.RelativePath = file.RelativePath
	}
	violations = append(violations, sectionViolations...)

	return violations, nil
}

func checkMarkdownSection(sections []*markdown.Section, schemas []*ConfigHeadingSchema, enforceOrder bool) []*Violation {
	// Collect all found violations (don't fail at the first one to improve user experience)
	violations := []*Violation{}

	// Current schema index to enforce order if required
	// If a schema has match, all following sections can only match this schema or the next ones
	currentSchemaIndex := 0

	// Keep track of how many sections matched each schema to enforce required and allow-multiple constraints
	matchingSectionsPerSchema := make(map[int]int)

	for _, section := range sections {
		foundMatch := false
		for i, schema := range schemas[currentSchemaIndex:] {
			if schema.MatchSection(section) {
				foundMatch = true
				matchingSectionsPerSchema[i]++
				if enforceOrder {
					// Prevent using previous schemas for following sections
					currentSchemaIndex = i
				}

				violations = append(violations, checkMarkdownSection(section.Subsections, schema.Children, schema.EnforceOrder)...)
				break
			}
		}
		if !foundMatch {
			violations = append(violations, &Violation{
				Name:    "check-schema",
				Message: fmt.Sprintf("section %q does not match the schema", section.HeadingText),
				Line:    section.FileLineStart,
			})
		}
	}

	for i, schema := range schemas {
		if schema.Required && matchingSectionsPerSchema[i] == 0 {
			violations = append(violations, &Violation{
				Name:    "check-schema",
				Message: fmt.Sprintf("no required %s matching the schema", schema),
				Line:    0, // No known line
			})
		}
		if !schema.AllowMultiple && matchingSectionsPerSchema[i] > 1 {
			violations = append(violations, &Violation{
				Name:    "check-schema",
				Message: fmt.Sprintf("multiple headings matching allowed schema %s", schema),
				Line:    0, // No known line
			})
		}
	}

	return violations
}

func (c *ConfigHeadingSchema) MatchSection(section *markdown.Section) bool {
	if c.Match != "" {
		match, err := regexp.MatchString(c.Match, section.HeadingText.String())
		if err != nil {
			return false
		}
		if !match {
			return false
		}
	}

	if c.MatchType != "" {
		noteType, _, ok := CurrentConfigFile().MatchNoteType(section.HeadingText.String())
		if !ok {
			return false
		}
		match, err := regexp.MatchString(c.MatchType, noteType.Name)
		if err != nil {
			return false
		}
		if !match {
			return false
		}
	}

	return true
}

// CheckAttributes ensures attributes are valid and match the expected type.
func CheckAttributes(file *ParsedFile) ([]*Violation, error) {
	var violations []*Violation

	for _, note := range file.Notes {

		attributeDefinitions := CurrentConfigFile().Attributes

		var fileDefinition *ConfigFileType
		var noteDefinition *ConfigNoteType
		if file.Type != "" {
			fileDefinition = CurrentConfigFile().MustGetFileType(file.Type)
		}
		noteDefinition = CurrentConfigFile().MustGetNoteType(note.Type)

		for _, attributeDefinition := range attributeDefinitions {

			allowedNames := []string{attributeDefinition.Name}
			allowedNames = append(allowedNames, attributeDefinition.Aliases...)

			foundInFile := false // Found in file front matter
			found := false       // Just found (inherited, in file, or in note)

			for _, name := range allowedNames {

				// Attributes can be defined in Front Matter or in the note itself
				// We need to check both (especially if the note override an invalid file attribute, we still want to raise an error).

				fileValue, presentOnFile := file.FileAttributes[name]
				if presentOnFile {
					foundInFile = true
					found = true

					line := text.LineNumber(string(file.Markdown.Content), name+":")
					if valid, err := attributeDefinition.Valid(fileValue); !valid {
						violations = append(violations, &Violation{
							Name:         "check-attributes",
							RelativePath: file.RelativePath,
							Message:      fmt.Sprintf("attribute %q in file %q: %s", name, file.Title, err),
							Line:         line,
						})
					}
				}

				noteValue, presentOnNote := note.NoteAttributes[name]
				if presentOnNote {
					found = true

					line := text.LineNumber(string(file.Markdown.Content), name+":")
					if valid, err := attributeDefinition.Valid(noteValue); !valid {
						violations = append(violations, &Violation{
							Name:         "check-attributes",
							RelativePath: file.RelativePath,
							Message:      fmt.Sprintf("attribute %q on note %q in file %q: %s", name, note.Title, file.RelativePath, err),
							Line:         line,
						})
					}
				}

				// Also check the merged attributes (which include defaults and inherited values)
				mergedValue, presentInMerged := note.Attributes[name]
				if presentInMerged && !presentOnNote && !presentOnFile {
					found = true
					// This is an attribute that comes from defaults or inheritance
					if valid, err := attributeDefinition.Valid(mergedValue); !valid {
						violations = append(violations, &Violation{
							Name:         "check-attributes",
							RelativePath: file.RelativePath,
							Message:      fmt.Sprintf("attribute %q on note %q in file %q: %s", name, note.Title, file.RelativePath, err),
							Line:         note.Line,
						})
					}
				}
			}

			// Check required attributes
			required := noteDefinition.RequiredAttribute(attributeDefinition.Name)
			if required && !found {
				violations = append(violations, &Violation{
					Name:         "check-attributes",
					RelativePath: file.RelativePath,
					Message:      fmt.Sprintf("attribute %q missing on note %q in file %q", attributeDefinition.Name, note.Title, file.RelativePath),
					Line:         note.Line,
				})
			}
			if fileDefinition != nil {
				required = fileDefinition.RequiredAttribute(attributeDefinition.Name)
				if required && !foundInFile {
					violations = append(violations, &Violation{
						Name:         "check-attributes",
						RelativePath: file.RelativePath,
						Message:      fmt.Sprintf("attribute %q missing on file %q", attributeDefinition.Name, file.Title),
						Line:         note.Line,
					})
				}
			}
		}

		// Nothing more to check
		continue
	}

	return violations, nil
}

/* ParsedFile */

func (f *ParsedFile) Lint(ruleNames []string) ([]*Violation, error) {
	var violations []*Violation

	rules := CurrentConfigFile().Linter.Rules
	for _, configRule := range rules {
		fn := LintRulesFn[configRule.Name]

		if len(ruleNames) > 0 && !slices.Contains(ruleNames, configRule.Name) {
			// Skip this rule
			continue
		}

		// Some rules have a query to filter the notes
		var query *Query = nil
		if configRule.Query != "" {
			query, _ = ParseQuery(configRule.Query)
		}

		newViolations, err := fn(f, query, configRule.Args)
		if err != nil {
			return nil, err
		}
		violations = append(violations, newViolations...)
	}

	return violations, nil
}

/* Helper functions */

// IntFromJSON converts a unmarshalled JSON value to an int.
func IntFromJSON(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case string:
		castValue, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("argument %s must be an integer", value)
		}
		return castValue, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("argument %T not supported", value)
	}
}
