package text

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

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

// IsBlank returns if a text is blank.
func IsBlank(text string) bool {
	return len(strings.TrimSpace(text)) == 0
}

// IsNumber returns if a text is a number.
func IsNumber(text string) bool {
	_, err := strconv.Atoi(text)
	return err == nil
}
