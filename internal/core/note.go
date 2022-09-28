package core

import "time"

type NoteKind int

const (
	KindReference  NoteKind = 0
	KindNote       NoteKind = 1
	KindFlashcard  NoteKind = 2
	KindCheatsheet NoteKind = 3
	KindQuote      NoteKind = 4
	KindJournal    NoteKind = 5
)

type Note struct {
	ID int64

	// File containing the note
	FileID int64

	// Type of note
	Kind NoteKind

	// The filepath of the file containing the note (denormalized field)
	Filepath string

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
