package markdown

import (
	"bufio"
	"bytes"
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
