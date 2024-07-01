package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
)

type LintResult struct {
	AnalyzedFiles int
	AffectedFiles int
	Warnings      []*Violation
	Errors        []*Violation
}

// Append merges new violations into the current result.
func (r *LintResult) Append(violations ...*Violation) {
	lintFile := CurrentConfig().LintFile
	for _, violation := range violations {
		if lintFile.Severity(violation.Name) == "warning" {
			r.Warnings = append(r.Warnings, violation)
		} else {
			r.Errors = append(r.Errors, violation)
		}
	}
}

func (r LintResult) String() string {
	var res strings.Builder
	res.WriteString(fmt.Sprintf("%d invalid files on %d analyzed files (%d errors, %d warnings)\n",
		r.AffectedFiles,
		r.AnalyzedFiles,
		len(r.Errors),
		len(r.Warnings)))
	for _, violation := range r.Errors {
		res.WriteString(fmt.Sprintf("[WARNING] %s (%s:%d)\n", violation.Message, violation.RelativePath, violation.Line))
	}
	for _, violation := range r.Warnings {
		res.WriteString(fmt.Sprintf("[WARNING] %s (%s:%d)\n", violation.Message, violation.RelativePath, violation.Line))
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

	// Every slug must be unique
	"no-duplicate-slug": {
		Eval: NoDuplicateSlug,
	},

	// Enforce a minimum number of lines between notes
	"min-lines-between-notes": {
		Eval: MinLinesBetweenNotes,
	},

	// Enforce a maximum number of lines between notes
	"max-lines-between-notes": {
		Eval: MaxLinesBetweenNotes,
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

	// No ambiguity in wikilinks
	"no-ambiguous-wikilink": {
		Eval: NoAmbiguousWikilink,
	},

	// Attributes must satisfy their schema if defined
	"check-attribute": {
		Eval: CheckAttribute,
	},

	// At least one tag on quotes (must match the optional pattern).
	"require-quote-tag": {
		Eval: RequireQuoteTag,
	},
}

/* Schemas */

// GetSchemaAttributeTypes returns all declared attributes with their JSON types.
func GetSchemaAttributeTypes() map[string]string {
	results := make(map[string]string)
	for _, schema := range CurrentConfig().LintFile.Schemas {
		for _, attribute := range schema.Attributes {
			// Potential conflicting types are already checked after config parsing.
			results[attribute.Name] = attribute.Type
		}
	}
	return results
}

// GetSchemaAttributeType returns the type for the given attribute
// and defaults to string if no declaration is found.
func GetSchemaAttributeType(name string) string {
	declaredTypes := GetSchemaAttributeTypes()
	declaredType, ok := declaredTypes[name]
	if !ok {
		return "string"
	}
	return declaredType
}

// GetSchemaAttributes calculates the list of declared attributes for a given note.
func GetSchemaAttributes(relativePath string, kind NoteKind) []*ConfigLintSchemaAttribute {
	// We must find the most specific definition for every attributes.
	//
	// Ex:
	// schemas:
	// - name: Attributes
	//   attributes:
	//   - name: author
	//     type: string
	//
	// - name: Books
	//   path: references/books/
	//   attributes:
	//   - name: author
	//     required: true
	//
	// We must use the second schema when both apply.

	var matchingSchemas []ConfigLintSchema
	for _, schema := range CurrentConfig().LintFile.Schemas {
		if schema.Path != "" && !strings.HasPrefix(relativePath, schema.Path) {
			// Path does not match
			continue
		}
		if schema.Kind != "" && string(kind) != schema.Kind {
			// Kind does not match
			continue
		}
		matchingSchemas = append(matchingSchemas, schema)
	}
	if len(matchingSchemas) == 0 {
		// No attributes defined in schemas
		return nil
	}

	// Sort from most specific to least specific
	slices.SortFunc(matchingSchemas, func(a, b ConfigLintSchema) bool {
		// Most specific path first
		if a.Path != b.Path {
			return strings.HasPrefix(a.Path, b.Path)
		}
		if a.Kind != "" && b.Kind == "" {
			return true
		} else if a.Kind == "" && b.Kind != "" {
			return false
		}
		return false // both have same priority... (NB: SortFunc is not stable...)
	})

	resultsMap := make(map[string]*ConfigLintSchemaAttribute)
	// Iterate from least to most specific so that more specific definitions override previous ones.
	for i := len(matchingSchemas) - 1; i >= 0; i-- {
		schema := matchingSchemas[i]
		for _, definition := range schema.Attributes {
			resultsMap[definition.Name] = definition
		}
	}

	// Return values
	var results []*ConfigLintSchemaAttribute
	for _, definition := range resultsMap {
		results = append(results, definition)
	}
	// Sort by name
	slices.SortFunc(results, func(a, b *ConfigLintSchemaAttribute) bool {
		return a.Name < b.Name
	})
	return results
}

// NonInheritableAttributes returns the attributes that must not be inherited.
func NonInheritableAttributes(relativePath string, kind NoteKind) []string {
	var results []string
	definitions := GetSchemaAttributes(relativePath, kind)
	for _, definition := range definitions {
		if !*definition.Inherit {
			results = append(results, definition.Name)
		}
	}
	return results
}

// FilterNonInheritableAttributes removes from the list all non-inheritable attributes.
func FilterNonInheritableAttributes(attributes map[string]interface{}, relativePath string, kind NoteKind) map[string]interface{} {
	nonInheritableAttributes := NonInheritableAttributes(relativePath, kind)
	result := make(map[string]interface{})
	for key, value := range attributes {
		if slices.Contains(nonInheritableAttributes, key) {
			// non-inheritable
			continue
		}
		result[key] = value
	}
	return result
}

/* Rules */

// NoDuplicateNoteTitle implements the rule "no-duplicate-note-title".
func NoDuplicateNoteTitle(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	uniqueNoteTitles := make(map[string]bool)
	for _, note := range file.Notes {
		cleanTitle := note.Title.MustTransform(markdown.StripEmphasis()).String()
		if _, ok := uniqueNoteTitles[cleanTitle]; ok {
			violations = append(violations, &Violation{
				Name:         "no-duplicate-note-title",
				Message:      fmt.Sprintf("duplicated note with title %q", note.ShortTitle),
				RelativePath: file.RelativePath,
				Line:         file.FileLineNumber(note.Line),
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
func NoDuplicateSlug(file *ParsedFile, args []string) ([]*Violation, error) {
	slugInventoryOnce.Do(func() {
		slugInventory = make(map[string]bool)
	})

	var violations []*Violation

	for _, note := range file.Notes {
		// Collect relevant attributes
		fileSlug := file.Slug
		attributeSlug := ""
		if slugRawValue, ok := note.NoteAttributes["slug"]; ok {
			if slugStringValue, ok := slugRawValue.(string); ok {
				attributeSlug = slugStringValue
			}
		}

		// Determine the note
		slug := DetermineNoteSlug(fileSlug, attributeSlug, note.Kind, string(note.ShortTitle))

		// Check if not already in use
		if _, ok := slugInventory[slug]; ok {
			violations = append(violations, &Violation{
				Name:         "no-duplicate-slug",
				Message:      fmt.Sprintf("duplicated slug %q", slug),
				RelativePath: file.RelativePath,
				Line:         file.FileLineNumber(note.Line),
			})
		} else {
			if markdown.Slug(slug) != slug {
				// Slug does not match the expected format
				// (important to use slug in URLs)
				violations = append(violations, &Violation{
					Name:         "no-duplicate-slug",
					Message:      fmt.Sprintf("invalid slug format %q", slug),
					RelativePath: file.RelativePath,
					Line:         file.FileLineNumber(note.Line),
				})
			}
			slugInventory[slug] = true
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

	lines := file.Markdown.Body.Lines()

	for i, note := range file.Notes {
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
					Line:         file.FileLineNumber(note.Line),
				})
			}
		}
	}

	return violations, nil
}

// MaxLinesBetweenNotes implements the rule "min-lines-between-notes".
func MaxLinesBetweenNotes(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	if len(args) != 1 {
		return nil, errors.New("only a single argument is required")
	}
	maxLines, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, fmt.Errorf("argument %s must be an integer", args[0])
	}

	lines := file.Markdown.Body.Lines()

	for _, note := range file.Notes {
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
				Line:         file.FileLineNumber(note.Line),
			})
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

	for _, note := range file.Notes {
		if note.Kind == KindFree {
			// Free notes can used any syntax
			continue
		}
		if !re.MatchString(string(note.Title)) {
			violations = append(violations, &Violation{
				Name:         "note-title-match",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("note title %q does not match regex %q", note.Title, args[0]),
				Line:         file.FileLineNumber(note.Line),
			})
		}
	}

	return violations, nil
}

// NoFreeNote implements the rule "no-free-note".
func NoFreeNote(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	for _, note := range file.Notes {
		if note.Kind == KindFree {
			violations = append(violations, &Violation{
				Name:         "no-free-note",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("free note %q not allowed", note.Title),
				Line:         file.FileLineNumber(note.Line),
			})
		}
	}

	return violations, nil
}

// NoDanglingMedia implements the rule "no-dangling-media".
func NoDanglingMedia(file *ParsedFile, args []string) ([]*Violation, error) {
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
	paths := []string{CurrentConfig().RootDirectory}
	err := CurrentRepository().walkNew(paths, func(md *markdown.File) error {
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
func NoDeadWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
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
func NoExtensionWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
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
func NoAmbiguousWikilink(file *ParsedFile, args []string) ([]*Violation, error) {
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

// RequireQuoteTag implements the rule "require-quote-tag"
func RequireQuoteTag(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	if len(args) > 1 {
		return nil, errors.New("only a single argument is allowed")
	}
	regexPattern := regexp.MustCompile(".*")
	if len(args) == 1 {
		regexArgument, err := regexp.Compile(args[0])
		if err != nil {
			return nil, fmt.Errorf("argument %s must be a valid regular expression", args[0])
		}
		regexPattern = regexArgument
	}

	for _, note := range file.Notes {
		if note.Kind != KindQuote {
			continue
		}

		attributes := file.FileAttributes.Merge(note.NoteAttributes)
		tags := note.NoteTags
		if attributeValue, ok := attributes["tags"]; ok {
			if attributeTags, ok := attributeValue.([]interface{}); ok {
				for _, attributeTag := range attributeTags {
					if attributeTagStr, ok := attributeTag.(string); ok {
						tags = append(tags, attributeTagStr)
					}
				}
			}
		}

		atLeastOneTagMatch := false
		for _, tag := range tags {
			if regexPattern.MatchString(tag) {
				atLeastOneTagMatch = true
				break
			}
		}

		if !atLeastOneTagMatch {
			violations = append(violations, &Violation{
				Name:         "require-quote-tag",
				RelativePath: file.RelativePath,
				Message:      fmt.Sprintf("quote %q does not have tags", note.Title),
				Line:         file.FileLineNumber(note.Line),
			})
		}
	}

	return violations, nil
}

// CheckAttribute implements the rule "check-attribute"
func CheckAttribute(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	for _, note := range file.Notes {

		definitions := GetSchemaAttributes(file.RelativePath, note.Kind)
		for _, definition := range definitions {

			allowedNames := []string{definition.Name}
			allowedNames = append(allowedNames, definition.Aliases...)

			found := false

			for _, name := range allowedNames {

				fileValue, presentOnFile := file.FileAttributes[name]
				noteValue, presentOnNote := note.NoteAttributes[name]

				// Check type
				if presentOnFile {
					found = true

					// FIXME reuse CastFn
					line := text.LineNumber(string(file.Markdown.Content), name+":")
					switch definition.Type {
					case "string[]":
						if !IsArray(fileValue) && !IsPrimitive(fileValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not an array and cannot be converted", name, file.RelativePath),
								Line:         line,
							})
						}
					case "string":
						if !IsString(fileValue) && !IsPrimitive(fileValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a string and cannot be converted", name, file.RelativePath),
								Line:         line,
							})
						} else if definition.Pattern != "" {
							// Check pattern
							regexAttribute := regexp.MustCompile(definition.Pattern)
							// Convert value to string
							fileStringValue := fmt.Sprintf("%s", fileValue)
							if !regexAttribute.MatchString(fileStringValue) {
								violations = append(violations, &Violation{
									Name:         "check-attribute",
									RelativePath: file.RelativePath,
									Message:      fmt.Sprintf("attribute %q in file %q does not match pattern %q", name, file.RelativePath, definition.Pattern),
									Line:         line,
								})
							}
						}
					case "object":
						if !IsObject(fileValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not an object", name, file.RelativePath),
								Line:         line,
							})
						}
					case "number":
						if !IsNumber(fileValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a number", name, file.RelativePath),
								Line:         line,
							})
						}
					case "boolean":
						fallthrough
					case "bool":
						if !IsBool(fileValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a bool", name, file.RelativePath),
								Line:         line,
							})
						}
					}
				}
				if presentOnNote {
					found = true
					line := file.Markdown.BodyLine + note.Line - 1 + text.LineNumber(note.Body.String(), "@"+name)
					switch definition.Type {
					case "string[]":
						if !IsArray(noteValue) && !IsPrimitive(noteValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not an array and cannot be converted", name, file.RelativePath),
								Line:         line,
							})
						}
					case "string":
						if !IsString(noteValue) && !IsPrimitive(noteValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a string and cannot be converted", name, file.RelativePath),
								Line:         line,
							})
						} else if definition.Pattern != "" {
							// Check pattern
							regexAttribute := regexp.MustCompile(definition.Pattern)
							// Convert value to string
							noteStringValue := fmt.Sprintf("%s", noteValue)
							if !regexAttribute.MatchString(noteStringValue) {
								violations = append(violations, &Violation{
									Name:         "check-attribute",
									RelativePath: file.RelativePath,
									Message:      fmt.Sprintf("attribute %q in note %q in file %q does not match pattern %q", name, note.Title, file.RelativePath, definition.Pattern),
									Line:         line,
								})
							}
						}
					case "object":
						if !IsObject(noteValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not an object", name, file.RelativePath),
								Line:         line,
							})
						}
					case "number":
						if !IsNumber(noteValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a number", name, file.RelativePath),
								Line:         line,
							})
						}
					case "boolean":
						fallthrough
					case "bool":
						if !IsBool(noteValue) {
							violations = append(violations, &Violation{
								Name:         "check-attribute",
								RelativePath: file.RelativePath,
								Message:      fmt.Sprintf("attribute %q in file %q is not a bool", name, file.RelativePath),
								Line:         line,
							})
						}
					}
				}

			}

			// Check required
			if *definition.Required && !found {
				violations = append(violations, &Violation{
					Name:         "check-attribute",
					RelativePath: file.RelativePath,
					Message:      fmt.Sprintf("attribute %q missing on note %q in file %q", definition.Name, note.Title, file.RelativePath),
					Line:         file.FileLineNumber(note.Line),
				})
			}

			// Nothing more to check
			continue
		}
	}

	return violations, nil
}

/* ParsedFile */

func (f *ParsedFile) Lint(ruleNames []string) ([]*Violation, error) {
	var violations []*Violation

	rules := CurrentConfig().LintFile.Rules
	for _, configRule := range rules {
		rule := LintRules[configRule.Name]

		if len(ruleNames) > 0 && !slices.Contains(ruleNames, configRule.Name) {
			// Skip this rule
			continue
		}

		// Check path restrictions
		matchAllIncludes := true
		for _, include := range configRule.Includes {
			if !include.Match(f.RelativePath) {
				matchAllIncludes = false
			}
		}
		if !matchAllIncludes {
			continue
		}

		newViolations, err := rule.Eval(f, configRule.Args)
		if err != nil {
			return nil, err
		}
		violations = append(violations, newViolations...)
	}

	return violations, nil
}
