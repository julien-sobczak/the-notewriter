package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// Regex to match wikilinks
var regexWikilink = regexp.MustCompile(`\[\[([/\a-zA-Z0-9_-]*?(?:#.*?)?)(?:\|(.*?))?\]\]`)

// Wikilink is an internal link.
// See https://en.wikipedia.org/wiki/Help:Link
type Wikilink struct {
	Link string
	Text string
	Line int
}

// MatchWikilink tests if a text is a wikilink.
func MatchWikilink(txt string) bool {
	match := regexWikilink.FindStringSubmatch(txt)
	return match != nil
}

// NewWikilink instantiates a new wikilink.
func NewWikilink(link string) (*Wikilink, error) {
	match := regexWikilink.FindStringSubmatch(link)
	if match == nil {
		return nil, fmt.Errorf("invalid wikilink %q", link)
	}
	return &Wikilink{
		Link: match[1],
		Text: match[2],
	}, nil
}

// Anchored indicates if a link points to a section in the current file. (ex: [[#A section below]])
func (w *Wikilink) Anchored() bool {
	return strings.HasPrefix(w.Link, "#")
}

// Path returns the link without the optional fragment.
func (w *Wikilink) Path() string {
	parts := strings.Split(w.Link, "#")
	return parts[0]
}

// Section returns the fragment part of the link.
func (w *Wikilink) Section() string {
	parts := strings.Split(w.Link, "#")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// Internal returns if the link points to the current file.
func (w *Wikilink) Internal() bool {
	return w.Path() == ""
}

// External returns if the link points to a different file.
func (w *Wikilink) External() bool {
	return !w.Internal()
}

// Piped indicates if a text is present to describe the link. (ex: [[link|A text]])
func (w *Wikilink) Piped() bool {
	return w.Text != ""
}

// ContainsExtenstion tests if the extension is specified in the link.
func (w *Wikilink) ContainsExtension() bool {
	return text.TrimExtension(w.Path()) != w.Path()
}

func (w Wikilink) String() string {
	if w.Piped() {
		return fmt.Sprintf("[[%s|%s]]", w.Link, w.Text)
	}
	return fmt.Sprintf("[[%s]]", w.Link)
}
