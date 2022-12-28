package markdown

import (
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

func ToMarkdown(markdownText string) string {
	regexTags := regexp.MustCompile("^(\\s*`#\\w+`)+\\s*$")

	markdownText = regexTags.ReplaceAllString(markdownText, "")
	markdownText = text.SquashBlankLines(markdownText)
	markdownText = strings.TrimSpace(markdownText)

	return markdownText
}

// IsHeading returns if a givne line is a Markdown heading and its level.
func IsHeading(line string) (bool, string, int) {
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
