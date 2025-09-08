package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
)

func init() {
	RegisterPreprocessor("date-extractor", DateExtractorPreprocessor)
	RegisterPreprocessor("quote-rewriter", QuoteRewriterPreprocessor)
	RegisterPreprocessor("generator", GeneratorPreprocessor)
	RegisterPreprocessor("flashcard-extractor", FlashcardExtractorPreprocessor)
	RegisterPreprocessor("list-items", ListItemsPreprocessor)
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

	insideQuote := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed != "" && !OnlyTagsAndAttributes(trimmed) && !strings.HasPrefix(trimmed, ">") {
			rewrittenLines = append(rewrittenLines, "> "+line)
			insideQuote = true
		} else {
			if insideQuote {
				// The quote is complete. Add the attribution
				attribution := note.Attributes.Attribution()
				if attribution != "" {
					rewrittenLines = append(rewrittenLines, "> "+attribution)
				}
			}
			insideQuote = false
			rewrittenLines = append(rewrittenLines, line)
		}
	}
	if insideQuote {
		attribution := note.Attributes.Attribution()
		if attribution != "" {
			rewrittenLines = append(rewrittenLines, "> "+attribution)
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
	parts := note.Body.SplitByHorizontalRules()

	if len(parts) == 1 {
		return flashcardWithClozeDeletionExtractor(file, note)
	}

	// Not possible to have more than 2 parts
	if len(parts) > 2 {
		return nil, errors.New("too many flashcard separator")
	}
	// 2 parts are required for cards (except for cloze deletions)
	if len(parts) < 2 {
		return nil, errors.New("missing flashcard separator")
	}

	front := parts[0]
	back := parts[1]

	note.Flashcards = append(note.Flashcards, &ParsedFlashcard{
		ShortTitle: note.ShortTitle,
		Slug:       note.Slug,
		Front:      front,
		Back:       back,
	})

	if note.NoteTags.Includes("reversed") {
		note.Flashcards = append(note.Flashcards, &ParsedFlashcard{
			ShortTitle: note.ShortTitle,
			Slug:       note.Slug + "-reversed",
			Front:      back,
			Back:       front,
		})
	}

	return []*ParsedNote{note}, nil
}

// flashcardWithClozeDeletionExtractor extracts flashcards using the cloze deletion syntax.
func flashcardWithClozeDeletionExtractor(_ *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	body := note.Body.String()

	// We use Anki syntax for cloze deletions using curly braces inside brackets.
	// Ex: [{c1::This is a cloze deletion::optional hint}]
	// See https://docs.ankiweb.net/editing.html#cloze-deletion
	// This syntax avoids conflicts with wikilinks ([[...]]) and templating systems ({{...}}).

	clozePattern := regexp.MustCompile(`\[\{(\w+)::([^\}:]+?)(?:::(.+?))?\}\]`)
	if !clozePattern.MatchString(body) {
		return nil, errors.New("missing cloze deletion syntax")
	}

	// Find all unique cloze groups
	clozes := clozePattern.FindAllStringSubmatch(body, -1)
	groupSet := []string{} // Use a set to keep group names in the order of their first appearance
	for _, sub := range clozes {
		if !slices.Contains(groupSet, sub[1]) {
			groupSet = append(groupSet, sub[1])
		}
	}

	// For each group, create a flashcard
	for _, group := range groupSet {
		front := clozePattern.ReplaceAllStringFunc(body, func(m string) string {
			sub := clozePattern.FindStringSubmatch(m)
			if sub[1] == group {
				if sub[3] != "" {
					return fmt.Sprintf("[%s]", sub[3])
				}
				return "[...]"
			}
			return sub[2]
		})
		back := clozePattern.ReplaceAllStringFunc(body, func(m string) string {
			sub := clozePattern.FindStringSubmatch(m)
			return sub[2]
		})
		note.Flashcards = append(note.Flashcards, &ParsedFlashcard{
			ShortTitle: note.ShortTitle,
			Slug:       note.Slug,
			Front:      markdown.Document(front),
			Back:       markdown.Document(back),
		})
	}

	return []*ParsedNote{note}, nil
}

// ListItemsPreprocessor extracts list items from Markdown content and populates the Items field.
// It parses Markdown lists, extracts tags, attributes, and emojis from the text, and creates
// a nested structure representing the list hierarchy.
func ListItemsPreprocessor(file *ParsedFile, note *ParsedNote) ([]*ParsedNote, error) {
	items := extractListItems(note.Body.String(), note.Line)
	note.Items = ListItems{Children: items}
	return []*ParsedNote{note}, nil
}

// extractListItems parses Markdown content and extracts list items with their metadata
func extractListItems(content string, baseLineNumber int) []*ListItem {
	lines := strings.Split(content, "\n")
	items := []*listItemWithIndent{}
	
	for i, line := range lines {
		lineNumber := baseLineNumber + i
		
		// Check if this line is a list item
		if item := parseListItemWithIndent(line, lineNumber); item != nil {
			items = append(items, item)
		}
	}
	
	// Process nested items
	return processNestedItemsWithIndent(items)
}

// listItemWithIndent is a temporary struct to track indentation during parsing
type listItemWithIndent struct {
	*ListItem
	IndentLevel int
}

// parseListItemWithIndent parses a single line and returns a listItemWithIndent if it's a list item, nil otherwise
func parseListItemWithIndent(line string, lineNumber int) *listItemWithIndent {
	// Regex to match various list item formats: *, -, +, 1., 2., etc.
	listPattern := regexp.MustCompile(`^(\s*)([\*\-\+]|\d+\.)\s+(.*)$`)
	matches := listPattern.FindStringSubmatch(line)
	
	if matches == nil {
		return nil
	}
	
	indentation := len(matches[1])
	text := matches[3]
	
	// Extract tags, attributes, and emojis from the text
	tags := extractTags(text)
	attributes := extractAttributes(text)
	emojis := extractEmojis(text)
	
	// Filter tags from text but keep emojis and attributes
	cleanText := filterTags(text)
	// Filter attributes only if preservedShorthand is false (for now, always filter)
	cleanText = filterAttributes(cleanText)
	
	return &listItemWithIndent{
		ListItem: &ListItem{
			Line:       lineNumber,
			Text:       strings.TrimSpace(cleanText),
			Tags:       tags,
			Attributes: attributes,
			Emojis:     emojis,
			Children:   []*ListItem{},
		},
		IndentLevel: indentation,
	}
}

// processNestedItemsWithIndent processes flat list of items and creates nested structure based on indentation
func processNestedItemsWithIndent(items []*listItemWithIndent) []*ListItem {
	if len(items) == 0 {
		return []*ListItem{}
	}
	
	result := []*ListItem{}
	stack := []*listItemWithIndent{} // Stack to track parent items at different indent levels
	
	for _, item := range items {
		// Pop stack until we find the right parent level
		for len(stack) > 0 && stack[len(stack)-1].IndentLevel >= item.IndentLevel {
			stack = stack[:len(stack)-1]
		}
		
		if len(stack) == 0 {
			// This is a top-level item
			result = append(result, item.ListItem)
		} else {
			// This is a child of the last item in the stack
			parent := stack[len(stack)-1]
			parent.ListItem.Children = append(parent.ListItem.Children, item.ListItem)
		}
		
		// Push current item to stack for potential children
		stack = append(stack, item)
	}
	
	return result
}

// extractTags finds and extracts tags in the format `#tag` from text
func extractTags(text string) []string {
	tagPattern := regexp.MustCompile("`([^`]+)`") // Matches `#tag` pattern
	matches := tagPattern.FindAllStringSubmatch(text, -1)
	
	var tags []string
	for _, match := range matches {
		tag := match[1]
		if strings.HasPrefix(tag, "#") {
			tags = append(tags, strings.TrimPrefix(tag, "#"))
		}
	}
	
	return tags
}

// extractAttributes extracts attributes from the text and returns them as a map
func extractAttributes(text string) map[string]interface{} {
	attributes := make(map[string]interface{})
	
	// Extract star rating (★★★★★) and convert to numeric rating
	starPattern := regexp.MustCompile("★+")
	if matches := starPattern.FindAllString(text, -1); len(matches) > 0 {
		rating := len([]rune(matches[0])) * 2 // Each star = 2 points, so 5 stars = 10
		attributes["rating"] = rating
	}
	
	return attributes
}

// extractEmojis finds all emojis in the text using a more robust approach
func extractEmojis(text string) []string {
	var emojis []string
	runes := []rune(text)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		// Check for flag sequences (regional indicators)
		if r >= 0x1F1E0 && r <= 0x1F1FF && i+1 < len(runes) {
			next := runes[i+1]
			if next >= 0x1F1E0 && next <= 0x1F1FF {
				// This is a country flag emoji (two regional indicators)
				emojis = append(emojis, string([]rune{r, next}))
				i++ // Skip the next rune since we processed it
				continue
			}
		}
		
		// Check for other emoji ranges
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x2600 && r <= 0x26FF) ||   // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) {    // Dingbats
			emojis = append(emojis, string(r))
		}
	}
	
	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, emoji := range emojis {
		if !seen[emoji] {
			unique = append(unique, emoji)
			seen[emoji] = true
		}
	}
	
	return unique
}

// filterTags removes tags in backticks from the text
func filterTags(text string) string {
	tagPattern := regexp.MustCompile("`#[^`]+`")
	return tagPattern.ReplaceAllString(text, "")
}

// filterAttributes removes attribute shorthand patterns from the text
func filterAttributes(text string) string {
	// Remove star ratings
	starPattern := regexp.MustCompile(`\s*★+\s*`)
	return starPattern.ReplaceAllString(text, "")
}

/* Helpers */

// CommandExists checks if a command exists in the system's PATH.
func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
