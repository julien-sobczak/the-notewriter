package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

// Wikilink is an internal link.
// See https://en.wikipedia.org/wiki/Help:Link
type Wikilink struct {
	Link string
	Text string
	Line int
}

// Anchored indicates if a link points to a section in the current file. (ex: [[#A section below]])
func (w *Wikilink) Anchored() bool {
	return strings.HasPrefix(w.Link, "#")
}

func (w *Wikilink) Path() string {
	parts := strings.Split(w.Link, "#")
	return parts[0]
}

func (w *Wikilink) Section() string {
	parts := strings.Split(w.Link, "#")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// Piped indicates if a text is present to describe the link. (ex: [[link|A text]])
func (w *Wikilink) Piped() bool {
	return w.Text != ""
}

// ContainsExtenstion tests if the extension is specified in the link.
func (w *Wikilink) ContainsExtension() bool {
	return text.TrimExtension(w.Link) != w.Link
}

func (w Wikilink) String() string {
	if w.Piped() {
		return fmt.Sprintf("[[%s|%s]]", w.Link, w.Text)
	}
	return fmt.Sprintf("[[%s]]", w.Link)
}

// ParseWikilinks extracts wikilinks from a text.
func ParseWikilinks(text string) []*Wikilink {
	var wikilinks []*Wikilink

	regexMedia := regexp.MustCompile(`\[\[([/\a-zA-Z0-9_-]*?(?:#.*?)?)(?:\|(.*?))?\]\]`)
	matches := regexMedia.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		link := text[match[2]:match[3]]
		title := ""
		if match[4] != -1 {
			title = text[match[4]:match[5]]
		}
		line := len(strings.Split(text[:match[0]], "\n"))

		wikilinks = append(wikilinks, &Wikilink{
			Link: link,
			Text: title,
			Line: line,
		})
	}

	return wikilinks
}
