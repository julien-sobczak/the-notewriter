package markdown

import (
	"strings"
	"unicode"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// Document represents a Markdown document (can be a whole file, or just a snippet)
type Document string

// Null object
var EmptyDocument = Document("")

// Lines returns the lines present in the Markdown document
func (m Document) Lines() []string {
	return strings.Split(string(m), "\n")
}

func (m Document) IsBlank() bool {
	return text.IsBlank(string(m))
}

func (m Document) Hash() string {
	return helpers.Hash([]byte(m))
}

func (m Document) Iterator() *text.LineIterator {
	return text.NewLineIteratorFromText(string(m))
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

// Wikilinks searches for wikilinks inside a Markdown document
func (m Document) Wikilinks() []*Wikilink {
	var results []*Wikilink

	// Ignore medias inside code blocks (ex: a sample Markdown code block)
	text := m.MustTransform(StripCodeBlocks()).String()

	matches := regexWikilink.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		link := text[match[2]:match[3]]
		title := ""
		if match[4] != -1 {
			title = text[match[4]:match[5]]
		}
		line := len(strings.Split(text[:match[0]], "\n"))

		results = append(results, &Wikilink{
			Link: link,
			Text: title,
			Line: line,
		})
	}

	return results
}

/*
 * Helpers
 */

// IsHeading returns if a given line is a Markdown heading and its level.
func IsHeading(line string) (bool, string, int) { // FIXME move to core/markdown.go?
	if !strings.HasPrefix(line, "#") {
		return false, "", 0
	}
	if strings.HasPrefix(line, "###### ") {
		return true, strings.TrimPrefix(line, "###### "), 6
	} else if strings.HasPrefix(line, "##### ") {
		return true, strings.TrimPrefix(line, "##### "), 5
	} else if strings.HasPrefix(line, "#### ") {
		return true, strings.TrimPrefix(line, "#### "), 4
	} else if strings.HasPrefix(line, "### ") {
		return true, strings.TrimPrefix(line, "### "), 3
	} else if strings.HasPrefix(line, "## ") {
		return true, strings.TrimPrefix(line, "## "), 2
	} else if strings.HasPrefix(line, "# ") {
		return true, strings.TrimPrefix(line, "# "), 1
	}

	return false, "", 0
}

// TODO document
func (m *Document) ToCleanMarkdown() Document { // FIXME formerly markdown.ToMarkdown
	return m.MustTransform(AlignHeadings(), SquashBlankLines()).TrimSpace()
}