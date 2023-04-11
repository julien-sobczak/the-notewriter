package markdown

import (
	"strings"

	"github.com/gomarkdown/markdown"
)

func ToHTML(md string) string {
	html := markdown.ToHTML([]byte(md), nil, nil)
	return strings.TrimSpace(string(html))
}

func ToInlineHTML(md string) string {
	html := strings.TrimSpace(ToHTML(md))
	if strings.HasPrefix(html, "<p>") && strings.HasSuffix(html, "</p>") {
		html = strings.TrimPrefix(html, "<p>")
		html = strings.TrimSuffix(html, "</p>")
	}
	return html
}
