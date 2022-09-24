package core

import (
	"bytes"
	"regexp"
	"strings"
)

// CompactYAML removes leading spaces in front of sequences.
//
// Ex:
//
//   doc:
//     - toto: tata
//
// Becomes
//
//   doc:
//   - toto: tata
func CompactYAML(doc string) string {
	// Identing sequences using zero-space (compact form) is not supported:
	// https://github.com/go-yaml/yaml/issues/661
	var buf bytes.Buffer
	r, _ := regexp.Compile(`^(\s*)  (- .*)$`)
	insideSequence := false
	var leadingSpaces string // the spaces prefix for successive lines in the sequence
	for _, line := range strings.Split(strings.TrimSuffix(doc, "\n"), "\n") {
		if r.MatchString(line) {
			rs := r.FindStringSubmatch(line)
			buf.WriteString(rs[1] + rs[2])
			buf.WriteString("\n")
			insideSequence = true
			leadingSpaces = rs[1] + "    "
		} else if insideSequence && strings.HasPrefix(line, leadingSpaces) {
			buf.WriteString(line[Indent:])
			buf.WriteString("\n")
		} else {
			buf.WriteString(line)
			buf.WriteString("\n")
			insideSequence = false
			leadingSpaces = ""
		}
	}
	return buf.String()
}
