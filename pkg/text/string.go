package text

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
)

// ExtractLines extract the given lines (1-based indices).
func ExtractLines(text string, start, end int) string {
	lines := strings.Split(text, "\n")
	start = start - 1
	if start < 0 {
		start = 0
	}
	if end == -1 || end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// SquashBlankLines replaces successive blank lines by a single empty one.
func SquashBlankLines(text string) string {
	var result bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(text))

	previousLineEmpty := false
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			if previousLineEmpty {
				continue
			}
			previousLineEmpty = true
		} else {
			previousLineEmpty = false
		}
		result.WriteString(line)
		result.WriteRune('\n')
	}

	return result.String()
}

// PrefixLines add a prefix to every line. All lines ends with \n in the result.
func PrefixLines(text string, prefix string) string {
	var result bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()
		result.WriteString(prefix)
		result.WriteString(line)
		result.WriteRune('\n')
	}

	return result.String()
}

// IsBlank returns if a text is blank.
func IsBlank(text string) bool {
	return len(strings.TrimSpace(text)) == 0
}

// IsNumber returns if a text is a number.
func IsNumber(text string) bool {
	_, err := strconv.Atoi(text)
	return err == nil
}

// TrimLinePrefix removes the prefix from every line
func TrimLinePrefix(text string, prefix string) string {
	var result bytes.Buffer
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		result.WriteString(strings.TrimPrefix(line, prefix))
		result.WriteRune('\n')
	}
	return result.String()
}

// TrimExtension removes the extension from a file name or file path.
func TrimExtension(path string) string {
	path = strings.TrimSuffix(path, string(filepath.Separator))
	return strings.TrimSuffix(path, filepath.Ext(path))
}

// LineNumber returns the number of the first line containing the subtring
// or -1 if not found.
func LineNumber(text string, sub string) int {
	i := strings.Index(text, sub)
	if i == -1 {
		return -1
	}
	return len(strings.Split(text[0:i], "\n"))
}

// Repeat repeats the same text the given number of times.
func Repeat(text string, n int) string {
	var result bytes.Buffer
	for i := 0; i < n; i++ {
		result.WriteString(text)
	}
	return result.String()
}
