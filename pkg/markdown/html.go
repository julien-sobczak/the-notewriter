package markdown

import (
	"strings"

	"github.com/gomarkdown/markdown"
)

func ToHTML(md string) string {
	html := markdown.ToHTML([]byte(md), nil, nil)
	return strings.TrimSpace(string(html))
}
