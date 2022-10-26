package markdown

import (
	"regexp"
	"strings"
)

func ToMarkdown(markdownText string) string {
	regexTags := regexp.MustCompile("^(\\s*`#\\w+`)+\\s*$")

	markdownText = regexTags.ReplaceAllString(markdownText, "")
	markdownText = SquashBlankLines(markdownText)
	markdownText = strings.TrimSpace(markdownText)

	return markdownText
}
