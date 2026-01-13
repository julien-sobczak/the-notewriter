package markdown

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// Document represents a Markdown document (can be a whole file, or just a snippet)
type Document string

// Null object
var EmptyDocument = Document("")

func (m Document) IsBlank() bool {
	return text.IsBlank(string(m))
}

func (m Document) Hash() string {
	return helpers.Hash([]byte(m))
}

func (m Document) String() string {
	return string(m)
}

// TrimBlankLines removes blank lines at the beginning and end of the document and returns the number of lines removed.
// TrimBlankLines is similar to TrimSpace but returns the count of lines trimmed.
func (m Document) TrimBlankLines() (result Document, countLinesAtStartTrimmed int, countLinesAtEndTrimmed int) {
	var raw string = string(m)

	rawWithoutPrefix := strings.TrimLeftFunc(raw, unicode.IsSpace)
	trimPrefixStart := raw[0 : len(raw)-len(rawWithoutPrefix)]
	countLinesAtStartTrimmed = strings.Count(trimPrefixStart, "\n")

	rawWithoutPrefixAndSuffix := strings.TrimRightFunc(rawWithoutPrefix, unicode.IsSpace)
	trimPrefixEnd := rawWithoutPrefix[len(rawWithoutPrefix)-(len(rawWithoutPrefix)-len(rawWithoutPrefixAndSuffix)):]
	countLinesAtEndTrimmed = strings.Count(trimPrefixEnd, "\n")

	result = Document(rawWithoutPrefixAndSuffix)
	return
}

// TrimSpace removes spaces at the start and end of a markdown document.
func (m Document) TrimSpace() Document {
	return Document(strings.TrimSpace(string(m)))
}

// ToANSI processes markdown text to replace emphasis with ANSI escape codes.
func (m Document) ToANSI() string {
	text := string(m)

	// First handle bold emphasis (**text** and __text__)
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between ** or __
		content := strings.Trim(match, "*_")
		return color.New(color.Bold).Sprint(content)
	})

	// Then handle italic emphasis (*text* and _text_)
	italicRegex := regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between * or _
		content := strings.Trim(match, "*_")
		return color.New(color.Italic).Sprint(content)
	})

	return text
}
