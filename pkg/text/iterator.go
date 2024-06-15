package text

import (
	"strings"
)

type Line struct {
	Text   string
	Number int
	next   *Line
	prev   *Line
}

// Null Object pattern.
// Useful to check l.Next().Next().IsBlank() => true even if l is the last line
var MissingLine = Line{
	Text:   "",
	Number: -1,
}

func (l Line) IsBlank() bool {
	return IsBlank(l.Text)
}

func (l Line) Next() Line {
	if l.next == nil {
		return MissingLine
	}
	return *l.next
}

func (l Line) Prev() Line {
	if l.prev == nil {
		return MissingLine
	}
	return *l.prev
}

func (l Line) IsFirst() bool {
	return l.prev == nil
}

func (l Line) IsLast() bool {
	return l.next == nil
}

// LineIterator implements the Iterator pattern to iterate over text lines.
type LineIterator struct {
	index int
	lines []*Line
}

func (l *LineIterator) HasNext() bool {
	return l.index < len(l.lines)
}

// Same as Next but does not move the iterator
func (l *LineIterator) Peek() Line {
	if l.HasNext() {
		line := l.lines[l.index]
		return *line
	}
	return MissingLine
}

func (l *LineIterator) Next() Line {
	if l.HasNext() {
		line := l.lines[l.index]
		l.index++
		return *line
	}
	return MissingLine
}

// SkipBlankLines moves the iterator to the next non-blank line (or return the null line otherwise).
func (l *LineIterator) SkipBlankLines() {
	for l.HasNext() {
		current := l.Peek()
		if current == MissingLine {
			break
		}
		if current.IsBlank() {
			l.Next()
		} else {
			break
		}
	}
}

func NewLineIteratorFromText(text string) *LineIterator {
	rawLines := strings.Split(text, "\n")

	var lines []*Line

	for i, line := range rawLines {
		current := &Line{
			Number: i + 1,
			Text:   line,
		}
		lines = append(lines, current)
	}

	for i, line := range lines {
		if i > 0 {
			line.prev = lines[i-1]
		}
		if i < len(lines)-1 {
			line.next = lines[i+1]
		}
	}

	return &LineIterator{
		index: 0,
		lines: lines,
	}
}
