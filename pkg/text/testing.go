package text

import "strings"

// UnescapeTestContent supports content using a special character instead of backticks.
func UnescapeTestContent(content string) string {
	// We support a special syntax for backticks in content.
	// Backticks are used to define note attributes (= common syntax with The NoteWriter) but
	// multiline strings in Golang cannot contains backticks.

	// We allow the ” character instead as suggested here: https://stackoverflow.com/a/59900008
	//
	// Example: ”@slug: toto” will become `@slug: toto`
	result := strings.ReplaceAll(content, "”", "`")

	// We allow the ‛ character
	// Example: ‛@slug: toto‛ will become `@slug: toto`
	result = strings.ReplaceAll(result, "‛", "`")

	return result
}
