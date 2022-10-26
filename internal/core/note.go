package core

import (
	"regexp"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
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
)

var regexNote = regexp.MustCompile(`^Note[-:_ ]\s*`)
var regexFlashcard = regexp.MustCompile(`^Flashcard[-:_ ]\s*`)
var regexCheatsheet = regexp.MustCompile(`^Cheatsheet[-:_ ]\s*`)
var regexQuote = regexp.MustCompile(`^Cheatsheet[-:_ ]\s*`)

type Note struct {
	ID int64

	// File containing the note
	FileID int64

	// Type of note
	Kind NoteKind

	// The filepath of the file containing the note (denormalized field)
	RelativePath string

	// Merged Front Matter containing file attributes + note-specific attributes
	FrontMatter map[string]interface{}

	// Comma-separated list of tags
	Tags []string

	// Line number (1-based index) of the note section title
	Line int

	// Content in Markdown format (best for editing)
	Content         string
	ContentMarkdown string
	ContentHTML     string
	ContentText     string

	// Timestamps to track changes
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

func NewNote(f *File, kind NoteKind, title, content string, lineNumber int) *Note {
	return &Note{
		FileID:          f.ID,
		Kind:            kind,
		RelativePath:    f.RelativePath,
		FrontMatter:     f.GetAttributes(),
		Tags:            getTags(f),
		Line:            lineNumber,
		Content:         content,
		ContentMarkdown: markdown.ToMarkdown(content),
		ContentHTML:     markdown.ToHTML(content),
		ContentText:     markdown.ToText(content),
	}
}

func getTags(f *File) []string {
	value := f.GetAttribute("tags")
	if tags, ok := value.([]string); ok {
		return tags
	}
	return nil
}
