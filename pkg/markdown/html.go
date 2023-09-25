package markdown

import (
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
)

func ToHTML(md string) string {
	renderer := newCustomizedRender()
	html := markdown.ToHTML([]byte(md), nil, renderer)
	result := strings.TrimSpace(string(html))
	// Support GLM Task lists
	result = supportTaskLists(result)
	return result
}

func ToInlineHTML(md string) string {
	html := strings.TrimSpace(ToHTML(md))
	if strings.HasPrefix(html, "<p>") && strings.HasSuffix(html, "</p>") {
		html = strings.TrimPrefix(html, "<p>")
		html = strings.TrimSuffix(html, "</p>")
	}
	return html
}

/* Extensions */

func supportTaskLists(html string) string {
	// See https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#task-lists
	rePendingTasks := regexp.MustCompile(`<li>\[ \]\s+(.*)</li>`)
	reCompletedTasks := regexp.MustCompile(`<li>\[x\]\s+(.*)</li>`)
	html = rePendingTasks.ReplaceAllString(html, `<li><input type="checkbox" /> $1</li>`)
	html = reCompletedTasks.ReplaceAllString(html, `<li><input type="checkbox" checked="checked" /> $1</li>`)
	return html
}

/* Customizations */
/* Read https://blog.kowalczyk.info/article/cxn3/advanced-markdown-processing-in-go.html */

func newCustomizedRender() *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags | mdhtml.HrefTargetBlank,
		RenderNodeHook: myRenderHook,
	}
	return mdhtml.NewRenderer(opts)
}

func myRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if image, ok := node.(*ast.Image); ok {
		renderMedia(w, image, entering)
		return ast.GoToNext, true
	}
	return ast.GoToNext, false
}

func renderMedia(w io.Writer, image *ast.Image, entering bool) {
	if entering {
		imageEnter(w, image)
	} else {
		imageExit(w, image)
	}
}

func imageEnter(w io.Writer, image *ast.Image) {
	attrs := mdhtml.BlockAttrs(image)
	src := string(image.Destination)
	oid := strings.TrimPrefix(src, "oid:")

	s := mdhtml.TagWithAttributes("<media", attrs)
	s = s[:len(s)-1] // hackish: strip off ">" from end
	io.WriteString(w, s+` oid="`+oid+`" alt="`)
}

func imageExit(w io.Writer, image *ast.Image) {
	if image.Title != nil {
		io.WriteString(w, `" title="`)
		mdhtml.EscapeHTML(w, image.Title)
	}
	io.WriteString(w, `" />`)
}
