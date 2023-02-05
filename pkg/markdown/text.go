package markdown

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

// How many spaces to indent headings per level
const indentHeading = 2

// How many spaces to indent code blocks
const indentCode = 4

func ToText(md string) string {
	// Headings
	var res bytes.Buffer
	for _, line := range strings.Split(md, "\n") {
		ok, headingTitle, level := IsHeading(line)
		if ok {
			switch level {
			case 1:
				res.WriteString(headingTitle)
				res.WriteString("\n")
				for i := 0; i < utf8.RuneCountInString(headingTitle); i++ {
					res.WriteRune('=')
				}
				res.WriteString("\n")
			case 2:
				res.WriteString(headingTitle)
				res.WriteString("\n")
				for i := 0; i < utf8.RuneCountInString(headingTitle); i++ {
					res.WriteRune('-')
				}
				res.WriteString("\n")
			default:
				spaces := (level - 2) * indentHeading
				for i := 0; i < spaces; i++ {
					res.WriteRune(' ')
				}
				res.WriteString(headingTitle)
				res.WriteString("\n")
			}
		} else {
			res.WriteString(line)
			res.WriteString("\n")
		}
	}
	txt := res.String()

	// Emphasis
	reBoldAsterisks := regexp.MustCompile(`\*\*(.*?)\*\*`)
	reBoldUnderscores := regexp.MustCompile(`__(.*?)__`)
	reItalicAsterisks := regexp.MustCompile(`\*(.*?)\*`)
	reItalicUnderscores := regexp.MustCompile(`_(.*?)_`)

	txt = reBoldAsterisks.ReplaceAllString(txt, "$1")
	txt = reBoldUnderscores.ReplaceAllString(txt, "$1")
	txt = reItalicAsterisks.ReplaceAllString(txt, "$1")
	txt = reItalicUnderscores.ReplaceAllString(txt, "$1")

	// Quotes
	res.Reset()
	lines := strings.Split(txt, "\n")
	insideQuote := false
	for i, line := range lines {
		if strings.HasPrefix(line, ">") {
			if !insideQuote {
				res.WriteRune('"')
			}
			quotationLine := strings.TrimSpace(strings.TrimPrefix(line, ">"))
			res.WriteString(quotationLine)
			if i == len(lines)-1 || !strings.HasPrefix(lines[i+1], ">") {
				// end the quote
				res.WriteRune('"')
			} else {
				// remember the quote is not finished
				insideQuote = true
			}
		} else {
			insideQuote = false
			res.WriteString(line)
		}
		res.WriteString("\n")
	}
	txt = res.String()

	// Block codes
	res.Reset()
	lines = strings.Split(txt, "\n")
	insideCode := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			insideCode = !insideCode
			continue
		}
		if insideCode {
			for i := 0; i < indentCode; i++ {
				res.WriteRune(' ')
			}
		}
		res.WriteString(line)
		res.WriteString("\n")
	}
	txt = res.String()

	// Links
	reLink := regexp.MustCompile(`(^|[^!])\[(.*?)\]\(.*?\)`)
	reURL := regexp.MustCompile(`<(https?://.*?)>`)
	reEmail := regexp.MustCompile(`<(.*?@.*?[.]\w+)>`)
	txt = reLink.ReplaceAllString(txt, "$1$2")
	txt = reURL.ReplaceAllString(txt, "$1")
	txt = reEmail.ReplaceAllString(txt, "$1")

	return strings.TrimSpace(txt)
}
