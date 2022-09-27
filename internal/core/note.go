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

}
