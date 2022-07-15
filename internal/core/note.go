package core

type NoteKind int64

const (
	KindReference  NoteKind = 0
	KindNote       NoteKind = 1
	KindFlashcard  NoteKind = 2
	KindCheatsheet NoteKind = 3
	KindJournal    NoteKind = 4
	KindTodo       NoteKind = 5
)

type Note struct {
	// A unique identifier among all notes
	ID string
	// The kind of note
	Kind NoteKind
	// A relative path to the collection directory
	RelativePath string
	// The FrontMatter for the note file
	FrontMatter map[string]interface{}

	// TODO split Content into a list of notes???? Or create a method GetSubNotes()???
	Content string
}

func (n *Note) Save() error {
	// Persist to disk
	return nil
}
