package markdown

import (
	"bytes"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

func ToMarkdown(markdownText string) string {
	markdownText = AlignHeadings(markdownText)
	markdownText = text.SquashBlankLines(markdownText)

	return strings.TrimSpace(markdownText)
}

// AlignHeadings unindents headings.
//
// Ex: (### Note: Blabla)
//
//	blabla
//	#### Blablabla
//	blablabla
//	##### Blablablabla
//	blablablabla
//
// Becomes:
//
//	blabla
//	## Blablabla
//	blablabla
//	### Blablablabla
//	blablablabla
func AlignHeadings(text string) string {
	// Search for top subheading level
	minHeadingLevel := -1
	for _, line := range strings.Split(text, "\n") {
		ok, _, level := IsHeading(line)
		if ok {
			if minHeadingLevel == -1 || level < minHeadingLevel {
				minHeadingLevel = level
			}
		}
	}

	if minHeadingLevel == -1 { // no heading found = nothing to do
		return text
	}

	// Up level to simulate a standalone Markdown document
	var res bytes.Buffer
	levelHeading := map[int]string{
		1: "#",
		2: "##",
		3: "###",
		4: "####",
		5: "#####",
		6: "######",
		7: "#######",
	}
	for _, line := range strings.Split(text, "\n") {
		ok, headingTitle, level := IsHeading(line)
		if ok {
			newLevel := level - minHeadingLevel + 2 // The top sub-heading should be ##
			res.WriteString(levelHeading[newLevel])
			res.WriteString(" ")
			res.WriteString(headingTitle)
		} else {
			res.WriteString(line)
		}
		res.WriteString("\n")
	}
	return res.String()
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
