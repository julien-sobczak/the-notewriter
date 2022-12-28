package core

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

type NoteKind int

const (
	KindFree       NoteKind = 0
	KindReference  NoteKind = 1
	KindNote       NoteKind = 2
	KindFlashcard  NoteKind = 3
	KindCheatsheet NoteKind = 4
	KindQuote      NoteKind = 5
	KindJournal    NoteKind = 6
	KindTodo       NoteKind = 7
	KindArtwork    NoteKind = 8
	KindSnippet    NoteKind = 9
)

var regexReference = regexp.MustCompile(`(?i)^Reference[-:_ ]\s*(.*)$`)
var regexNote = regexp.MustCompile(`^(?i)Note[-:_ ]\s*(.*)$`)
var regexFlashcard = regexp.MustCompile(`^(?i)Flashcard[-:_ ]\s*(.*)$`)
var regexCheatsheet = regexp.MustCompile(`^(?i)Cheatsheet[-:_ ]\s*(.*)$`)
var regexQuote = regexp.MustCompile(`^(?i)Quote[-:_ ]\s*(.*)$`)
var regexTodo = regexp.MustCompile(`^(?i)Todo[-:_ ]\s*(.*)$`)
var regexArtwork = regexp.MustCompile(`^(?i)Artwork[-:_ ]\s*(.*)$`)
var regexSnippet = regexp.MustCompile(`^(?i)Snippet[-:_ ]\s*(.*)$`)

type Note struct {
	ID int64

	// File containing the note
	FileID int64
	File   *File // Lazy-loaded

	// ParentNoteID
	ParentNoteID int64
	ParentNote   *Note // Lazy-loaded

	// Type of note
	Kind NoteKind

	// Original title of the note without leading # characters
	Title string
	// Short title of the note without the kind prefix
	ShortTitle string

	// The filepath of the file containing the note (denormalized field)
	RelativePath string

	// Note-specific attributes. Use GetAttributes() to get all merged attributes
	Attributes map[string]interface{}

	// Note-specific tags. Use GetTags() to get all merged tags
	Tags []string

	// Line number (1-based index) of the note section title
	Line int

	// Content in various formats (best for editing, rendering, writing, etc.)
	RawContent      string
	Content         string // Same as RawContent without tags, attributes, etc.
	ContentMarkdown string
	ContentHTML     string
	ContentText     string

	// Timestamps to track changes
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

func NewNote(f *File, title string, content string, lineNumber int) *Note {
	rawContent := strings.TrimSpace(content)

	_, kind, shortTitle := isSupportedNote(title)

	n := &Note{
		FileID:       f.ID,
		File:         f,
		ParentNoteID: -1,
		ParentNote:   nil,
		Kind:         kind,
		Title:        title,
		ShortTitle:   shortTitle,
		RelativePath: f.RelativePath,
		Line:         lineNumber,
	}

	n.updateContent(rawContent)

	return n
}

func (n *Note) updateContent(rawContent string) {
	n.RawContent = rawContent

	var tags []string
	var attributes map[string]interface{}
	var content string

	content, tags = extractTags(rawContent)
	content, attributes = extractAttributes(content)
	content = strings.TrimSpace(content)

	n.Tags = tags
	n.Attributes = attributes
	n.Content = n.expandSyntaxSugar(content)
	n.ContentMarkdown = markdown.ToMarkdown(n.Content)
	n.ContentHTML = markdown.ToHTML(n.Content)
	n.ContentText = markdown.ToText(n.Content)
}

func (n *Note) SetParent(parent *Note) {
	if parent == nil {
		return
	}
	n.ParentNote = parent
	n.ParentNoteID = parent.ID
}

// GetFile returns the containing file, loading it from database if necessary.
func (n *Note) GetFile() *File {
	if n.FileID == -1 {
		return nil
	}
	if n.File == nil {
		// FIXME lazy load from database
		panic("oops")
	}
	return n.File
}

// GetParentNote returns the parent note, loading it from database if necessary.
func (n *Note) GetParentNote() *Note {
	if n.ParentNoteID == -1 {
		return nil
	}
	if n.ParentNote == nil {
		// FIXME lazy load from database
		panic("oops")
	}
	return n.ParentNote
}

func (n *Note) GetAttributes() map[string]interface{} {
	if n.ParentNoteID == -1 {
		return mergeAttributes(n.GetFile().GetAttributes(), n.Attributes)

	}
	return mergeAttributes(n.GetParentNote().GetAttributes(), n.Attributes)
}

func (n *Note) GetAttribute(name string) interface{} {
	if value, ok := n.GetAttributes()[name]; ok {
		return value
	}
	return nil
}

func (n *Note) GetAttributeString(name, defaultValue string) string {
	value := n.GetAttribute(name)
	if value == nil {
		return defaultValue
	}
	return fmt.Sprintf("%v", value)
}

func (n *Note) GetTags() []string {
	if n.ParentNoteID == -1 {
		return mergeTags(n.GetFile().GetTags(), n.Tags)
	}
	return mergeTags(n.GetParentNote().GetTags(), n.Tags)
}

func mergeTags(tags ...[]string) []string {
	var result []string
	for _, items := range tags {
		for _, item := range items {
			found := false
			for _, existingItem := range result {
				if existingItem == item {
					found = true
					break
				}
			}
			if !found {
				result = append(result, item)
			}
		}
	}
	return result
}

func mergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
	// Implementation: THe code is obscure due to untyped elements.
	// We don't want to always replace old values when the old value is a slice
	// that can accept these new values too.
	//
	// Examples:
	//   ---
	//   tags: [a]
	//   references: []
	//   ---
	//
	//   `#b`
	//   <!-- references: https://example.org -->
	//
	// Should be the same as:
	//   ---
	//   tags: [a, b]
	//   references: [https://example.org]
	//   ---
	//
	// Most of the code tries to manage this use case.

	result := make(map[string]interface{})
	empty := true
	for _, m := range attributes {
		for newKey, newValue := range m {
			// Check if the attribute was already defined
			if currentValue, ok := result[newKey]; ok {

				// If the tyoe is a slice, append the new value instead of overriding
				switch x := currentValue.(type) {
				case []interface{}:
					switch y := newValue.(type) {
					case []interface{}:
						result[newKey] = append(x, y...)
					case []string:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					default:
						result[newKey] = append(x, newValue)
					}
				case []string:
					switch y := newValue.(type) {
					case []interface{}:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					case []string:
						result[newKey] = append(x, y...)
					default:
						result[newKey] = append(x, fmt.Sprintf("%v", newValue))
					}

				default:
					// override
					result[newKey] = newValue
				}
			} else {
				result[newKey] = newValue
			}
			empty = false
		}
	}
	if empty {
		return nil
	}
	return result
}

func (n Note) String() string {
	return fmt.Sprintf("[%v] %s", n.Kind, n.Title)
}

func (n *Note) expandSyntaxSugar(rawContent string) string {
	if n.Kind == KindQuote {
		// Turn every text line into a quote
		// Add the attribute name or author in suffix
		// Ex:
		//   ---
		//   name: Walt Disney
		//   ---
		//   "The way to get started is to quit"
		//   "talking and begin doing."
		//
		// Becomes:
		//
		//   > The way to get started is to quit
		//   > talking and begin doing.
		//   > — Walt Disney

		var res bytes.Buffer
		previousLineWasQuotation := false
		for _, line := range strings.Split(rawContent, "\n") {
			if text.IsBlank(line) {
				if previousLineWasQuotation {
					name := n.GetAttributeString("name", n.GetAttributeString("author", ""))
					if !text.IsBlank(name) {
						res.WriteString("> -- " + name + "\n")
					}
					res.WriteString(line + "\n")
					previousLineWasQuotation = false
				}
				res.WriteString(line + "\n")
			} else {
				res.WriteString("> " + strings.TrimSpace(line) + "\n")
				previousLineWasQuotation = true
			}
		}

		if previousLineWasQuotation {
			name := n.GetAttributeString("name", n.GetAttributeString("author", ""))
			if !text.IsBlank(name) {
				res.WriteString("> -- " + name + "\n")
			}
		}

		return res.String()
	}

	return rawContent
}

func extractTags(content string) (string, []string) {
	var tags []string

	reTags := regexp.MustCompile("`#(\\w+)`")
	reOnlyTags := regexp.MustCompile("^(`#\\w+`\\s*)+$")
	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		matches := reTags.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tags = append(tags, match[1])
		}
		if !reOnlyTags.MatchString(line) {
			res.WriteString(line + "\n")
		}
	}

	return text.SquashBlankLines(res.String()), tags
}

func extractAttributes(content string) (string, map[string]interface{}) {
	attributes := make(map[string]interface{})

	reAttribute := regexp.MustCompile(`<!--\s*(\w+)\s*:\s*(.*?)\s*-->\s*$`)

	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		match := reAttribute.FindStringSubmatch(line)
		if match != nil {
			attributes[match[1]] = match[2]
		} else {
			res.WriteString(line + "\n")
		}
	}
	return text.SquashBlankLines(res.String()), attributes
}

// TODO add SetParent method and traverse the hierachy to merge attributes/tags
// TOOD add GetMedias() method to extract medias from notes
