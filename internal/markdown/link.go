package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex to match links
const regexLinkRaw = `\[(.*?)\][(](\S*)?(?:\s+"(.*?)")?[)]`

var regexLink = regexp.MustCompile(`(?:^|[^!])` + regexLinkRaw) // Golang doesn't support negative lookbehind
var regexEmbeddedLink = regexp.MustCompile(`!` + regexLinkRaw)

type Link struct {
	Text  string
	URL   string
	Title string
	Line  int
}

func (l Link) Internal() bool {
	if strings.HasPrefix(l.URL, "file:") {
		return true
	}
	return !strings.Contains(l.URL, ":")
}

func (l Link) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`[%s](%s`, l.Text, l.URL))
	if l.Title != "" {
		sb.WriteString(fmt.Sprintf(` "%s"`, l.Title))
	}
	sb.WriteString(")")
	return sb.String()
}

/*
 * Document
 */

func (m Document) Links() []Link {
	return m.extractLinks(regexLink)
}

func (m Document) EmbededLinks() []Link {
	return m.extractLinks(regexEmbeddedLink)
}

func (m Document) extractLinks(r *regexp.Regexp) []Link {
	var results []Link

	// Ignore medias inside code blocks (ex: a sample Markdown code block)
	text := m.MustTransform(StripCodeBlocks()).String()

	matches := r.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		linkText := text[match[2]:match[3]]
		linkURL := text[match[4]:match[5]]
		linkTitle := ""
		if match[6] != -1 {
			linkTitle = text[match[6]:match[7]]
		}
		linkLine := len(strings.Split(text[:match[0]+1], "\n")) // Add +1 as the regex matches the previous character

		results = append(results, Link{
			Text:  linkText,
			URL:   linkURL,
			Title: linkTitle,
			Line:  linkLine,
		})
	}

	return results
}
