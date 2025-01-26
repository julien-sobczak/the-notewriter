package main

import (
	"database/sql"
	"io"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/note.go

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Title   string
	Content string
}

// ParseNotes extracts the notes from a file body.
func ParseNotes(fileBody string) []*ParsedNote {
	var noteTitle string
	var noteContent strings.Builder

	var results []*ParsedNote

	for _, line := range strings.Split(fileBody, "\n") {
		// Minimalist implementation. Only search for ## headings
		if strings.HasPrefix(line, "## ") {
			if noteTitle != "" {
				results = append(results, &ParsedNote{
					Title:   noteTitle,
					Content: strings.TrimSpace(noteContent.String()),
				})
			}
			noteTitle = strings.TrimPrefix(line, "## ")
			noteContent.Reset()
			continue
		}

		if noteTitle != "" {
			noteContent.WriteString(line)
			noteContent.WriteRune('\n')
		}
	}
	if noteTitle != "" {
		results = append(results, &ParsedNote{
			Title:   noteTitle,
			Content: strings.TrimSpace(noteContent.String()),
		})
	}

	return results
}

type Note struct {
	OID string `yaml:"oid"`

	// File containing the note
	FileOID string `yaml:"file_oid"`

	// Title of the note without leading # characters
	Title string `yaml:"title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	Content string `yaml:"content_raw"`
	Hash    string `yaml:"content_hash"`

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastIndexedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

// NewOrExistingNote loads and updates an existing note or creates a new one if new.
func NewOrExistingNote(f *File, parsedNote *ParsedNote) *Note {
	note, _ := CurrentRepository().FindNoteByTitle(f.RelativePath, parsedNote.Title)
	if note != nil {
		note.update(f, parsedNote)
		return note
	}

	return NewNoteFromParsedNote(f, parsedNote)
}

func NewNoteFromParsedNote(f *File, parsedNote *ParsedNote) *Note {
	return &Note{
		OID:          NewOID(),
		FileOID:      f.OID,
		Title:        parsedNote.Title,
		RelativePath: f.RelativePath,
		Content:      parsedNote.Content,
		Hash:         Hash([]byte(parsedNote.Content)),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		stale:        true,
		new:          true,
	}
}

func (n *Note) update(f *File, parsedNote *ParsedNote) {
	if f.OID != n.FileOID {
		n.FileOID = f.OID
		n.stale = true
	}

	if n.Content != parsedNote.Content {
		n.Content = parsedNote.Content
		n.Hash = Hash([]byte(parsedNote.Content))
		n.stale = true
	}
}

func (n *Note) Kind() string {
	return "note"
}

func (n *Note) UniqueOID() string {
	return n.OID
}

func (n *Note) ModificationTime() time.Time {
	return n.UpdatedAt
}

func (n *Note) State() State {
	if !n.DeletedAt.IsZero() {
		return Deleted
	}
	if n.new {
		return Added
	}
	if n.stale {
		return Modified
	}
	return None
}

func (n *Note) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(n)
	if err != nil {
		return err
	}
	return nil
}

func (n *Note) Write(w io.Writer) error {
	data, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (n *Note) SubObjects() []StatefulObject {
	// Usually return flashcards, medias, links, etc.
	return nil
}

func (n *Note) Save() error {
	var err error
	n.UpdatedAt = time.Now()
	n.LastIndexedAt = time.Now()
	switch n.State() {
	case Added:
		err = n.Insert()
	case Modified:
		err = n.Update()
	case Deleted:
		err = n.Delete()
	default:
		err = n.Check()
	}
	if err != nil {
		return err
	}
	n.new = false
	n.stale = false
	return nil
}

func (n *Note) Insert() error {
	query := `
		INSERT INTO note(
			oid,
			file_oid,
			relative_path,
			title,
			content_raw,
			hashsum,
			created_at,
			updated_at,
			last_indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := CurrentDB().Client().Exec(query,
		n.OID,
		n.FileOID,
		n.RelativePath,
		n.Title,
		n.Content,
		n.Hash,
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.LastIndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) Update() error {
	query := `
		UPDATE note
		SET
			file_oid = ?,
			relative_path = ?,
			title = ?,
			content_raw = ?,
			hashsum = ?,
			updated_at = ?,
			last_indexed_at = ?
		WHERE oid = ?;
	`

	_, err := CurrentDB().Client().Exec(query,
		n.FileOID,
		n.RelativePath,
		n.Title,
		n.Content,
		n.Hash,
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.LastIndexedAt),
		n.OID,
	)

	return err
}

func (n *Note) Delete() error {
	query := `DELETE FROM note WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, n.OID)
	return err
}

func (n *Note) Check() error {
	n.LastIndexedAt = time.Now()
	query := `
		UPDATE note
		SET last_indexed_at = ?
		WHERE oid = ?;`
	if _, err := CurrentDB().Client().Exec(query, timeToSQL(n.LastIndexedAt), n.OID); err != nil {
		return err
	}
	// Mark all sub-objects as checked too
	return nil
}

func (r *Repository) FindNoteByTitle(relativePath, title string) (*Note, error) {
	var n Note
	var createdAt string
	var updatedAt string
	var lastIndexedAt string

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			file_oid,
			relative_path,
			title,
			content_raw,
			hashsum,
			created_at,
			updated_at,
			last_indexed_at
		FROM note
		WHERE relative_path = ? and title = ?;`, relativePath, title).
		Scan(
			&n.OID,
			&n.FileOID,
			&n.RelativePath,
			&n.Title,
			&n.Content,
			&n.Hash,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.LastIndexedAt = timeFromSQL(lastIndexedAt)

	return &n, nil
}
