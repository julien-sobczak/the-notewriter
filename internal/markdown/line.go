package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// Line represents a line in a Markdown document
type Line struct {
	Text            string
	Number          int // 1-based line number
	InsideCodeBlock bool
	Prev            *Line
	Next            *Line
}

// Lines represents a collection of lines
type Lines []*Line

/* LineIterator implementation */

// LineIterator allows iterating over lines
type LineIterator struct {
	index int
	lines []*Line
}

func (d Document) Iterator() *LineIterator {
	return d.Lines().Iterator()
}

func (l Lines) Iterator() *LineIterator {
	return &LineIterator{
		index: -1,
		lines: l,
	}
}

func (it *LineIterator) HasNext() bool {
	return it.index+1 < len(it.lines)
}

func (it *LineIterator) Next() *Line {
	if !it.HasNext() {
		return nil
	}
	it.index++
	line := it.lines[it.index]
	return line
}

func (it *LineIterator) Peek() *Line {
	if !it.HasNext() {
		return nil
	}
	return it.lines[it.index+1]
}

func (it *LineIterator) SkipBlankLines() {
	for it.HasNext() {
		line := it.Peek()
		if line.IsBlank() {
			it.Next()
		} else {
			break
		}
	}
}

func (it *LineIterator) NextNonBlankLine() *Line {
	for it.HasNext() {
		line := it.Next()
		if !line.IsBlank() {
			return line
		}
	}
	return nil
}

func (it *LineIterator) SkipHeading() {
	ok, _, level := it.lines[it.index].IsHeading()
	if !ok {
		// No heading to skip
		return
	}
	if level > 2 {
		// Basic heading, the next call to Next() is enough
		return
	}
	if it.HasNext() {
		line := it.Peek()
		if regexHeading1Alternate.MatchString(line.Text) || regexHeading2Alternate.MatchString(line.Text) {
			it.Next()
		}
	}
}

/* Lines implementation */

// Lines returns the lines present in the Markdown document
func (m Document) Lines() Lines {
	var lines Lines

	rawLines := strings.Split(string(m), "\n")

	insideBlock := false
	blockRegex := regexp.MustCompile("(`{3,})(\\w*).*")
	blockBackticks := 0

	for i, text := range rawLines {
		matches := blockRegex.FindStringSubmatch(text)
		lineInBlock := insideBlock
		if matches != nil {
			backticksCount := len(matches[1])
			if !insideBlock {
				insideBlock = true
				lineInBlock = true
				blockBackticks = backticksCount
			} else if backticksCount >= blockBackticks {
				lineInBlock = true  // Still inside the code block
				insideBlock = false // but next line will be outside
				blockBackticks = 0
			}
		}
		lines = append(lines, &Line{
			Text:            text,
			Number:          i + 1,
			InsideCodeBlock: lineInBlock,
		})
	}

	// Set Previous and Next pointers
	for i := 0; i < len(lines); i++ {
		if i > 0 {
			lines[i].Prev = lines[i-1]
		}
		if i < len(lines)-1 {
			lines[i].Next = lines[i+1]
		}
	}

	return lines
}

func (l Lines) LastNumber() int {
	if len(l) == 0 {
		return 0
	}
	return l[len(l)-1].Number
}

/* Line implementation */

func (l Line) HasNext() bool {
	return l.Next != nil
}

func (l Line) HasPrev() bool {
	return l.Prev != nil
}

// IsBlank returns true if the line is blank (only whitespace)
func (l Line) IsBlank() bool {
	return text.IsBlank(l.Text)
}

// Regular expressions for alternate heading styles
var regexHeading1Alternate = regexp.MustCompile(`^={3,}$`)
var regexHeading2Alternate = regexp.MustCompile(`^-{3,}$`)

func (l Line) IsTopHeading() bool {
	isHeading, _, level := l.IsHeading()
	return isHeading && level == 1
}

// IsBlockquote returns the text of the blockquote and true if the line is a blockquote
func (l Line) IsBlockquote() (string, bool) {
	if !strings.HasPrefix(l.Text, ">") {
		return l.Text, false
	}
	return strings.TrimSpace(strings.TrimPrefix(l.Text, ">")), true
}

// IsHeading returns true if the line is a heading, along with the heading text and level (1-6)
func (l Line) IsHeading() (bool, string, int) {
	// Test first alternate heading styles
	if !l.IsBlank() && l.HasNext() {
		nextLine := l.Next
		if regexHeading1Alternate.MatchString(nextLine.Text) {
			return true, strings.TrimSpace(l.Text), 1
		}
		if regexHeading2Alternate.MatchString(nextLine.Text) {
			return true, strings.TrimSpace(l.Text), 2
		}
	}

	// No match, test the basic heading styles
	lineText := l.Text
	if !strings.HasPrefix(lineText, "#") {
		return false, "", 0
	}
	if strings.HasPrefix(lineText, "###### ") {
		return true, strings.TrimPrefix(lineText, "###### "), 6
	} else if strings.HasPrefix(lineText, "##### ") {
		return true, strings.TrimPrefix(lineText, "##### "), 5
	} else if strings.HasPrefix(lineText, "#### ") {
		return true, strings.TrimPrefix(lineText, "#### "), 4
	} else if strings.HasPrefix(lineText, "### ") {
		return true, strings.TrimPrefix(lineText, "### "), 3
	} else if strings.HasPrefix(lineText, "## ") {
		return true, strings.TrimPrefix(lineText, "## "), 2
	} else if strings.HasPrefix(lineText, "# ") {
		return true, strings.TrimPrefix(lineText, "# "), 1
	}

	return false, "", 0
}

func (l Line) IsHorizontalRule() bool {
	// Support more than three dashes, asterisks or underscores
	return l.Matches(`^((-\s*){3,}|(\*\s*){3,}|(_\s*){3,})\s*$`)
}

func (l Line) Matches(pattern string) bool {
	matched, err := regexp.MatchString(pattern, l.Text)
	if err != nil {
		return false
	}
	return matched
}

func (l Line) String() string {
	return fmt.Sprintf("%d: %s", l.Number, l.Text)
}
