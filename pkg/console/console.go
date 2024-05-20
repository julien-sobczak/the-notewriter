package console

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type ProgressLog struct {
	output        io.Writer
	showBar       bool
	showPercent   bool
	maxSteps      int
	maxCharacters int
}

func NewProgressLog(maxSteps int, options ...func(*ProgressLog)) *ProgressLog {
	result := &ProgressLog{
		output:        os.Stdout,
		showPercent:   false,
		showBar:       true,
		maxSteps:      maxSteps,
		maxCharacters: 80,
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func ToWriter(w io.Writer) func(*ProgressLog) {
	return func(s *ProgressLog) {
		s.output = w
	}
}

func HideBar() func(*ProgressLog) {
	return func(s *ProgressLog) {
		s.showBar = false
	}
}

func ShowPercent() func(*ProgressLog) {
	return func(s *ProgressLog) {
		s.showPercent = true
	}
}

func LineLength(characters int) func(*ProgressLog) {
	return func(s *ProgressLog) {
		s.maxCharacters = characters
	}
}

func (l *ProgressLog) Log(currentStep int, message string) {
	// Determine the percent
	i100 := currentStep * 100 / l.maxSteps

	// We show between 0 and 10 '#' depending on the percent
	i10 := i100 / 10

	// Build the line step by step
	var sb strings.Builder

	if l.showBar {
		sb.WriteString(strings.Repeat("#", i10))
		sb.WriteString(strings.Repeat(" ", 10-i10))
		sb.WriteRune(' ') // Add a space after the progress bar
	}

	if l.showPercent {
		sb.WriteString(fmt.Sprintf("(%3d%%) ", i100))
	} else {
		sb.WriteString(fmt.Sprintf("(%d/%d) ", currentStep, l.maxSteps))
	}

	sb.WriteString(message)

	line := sb.String()
	if len(line) > l.maxCharacters {
		line = line[0:l.maxCharacters]
	}
	line += strings.Repeat(" ", l.maxCharacters-len(line))

	// Show the result
	fmt.Fprint(l.output, line, "\r")
}

func (l *ProgressLog) Clear(newMessage string) {
	line := newMessage + strings.Repeat(" ", l.maxCharacters-len(newMessage))
	if len(line) > l.maxCharacters {
		line = line[0:l.maxCharacters]
	}

	// Rewrite the last line
	fmt.Fprint(l.output, line)

	if newMessage == "" {
		fmt.Fprint(l.output, "\r")
	} else {
		// Move to next line
		fmt.Fprint(l.output, "\n")
	}
}
