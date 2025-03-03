package main

import (
	"database/sql"
	"io"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/note.go

type Note struct {
	OID oid.OID `yaml:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`
	// File containing the note
	FileOID oid.OID `yaml:"file_oid"` // TODO useful?

	// Title of the note without leading # characters
	Title string `yaml:"title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	Content string `yaml:"content_raw"`

	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
	IndexedAt time.Time `yaml:"-"`
}

// NewNote creates a new note.
func NewNote(packFile *PackFile, file *File, parsedNote *ParsedNote) (*Note, error) {
	// Set basic properties
	n := &Note{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		FileOID:      file.OID,
		Title:        parsedNote.Title,
		RelativePath: file.RelativePath,
		Content:      parsedNote.Content,
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
	}

	return n, nil
}

// NewOrExistingNote loads and updates an existing note or creates a new one if new.
func NewOrExistingNote(packFile *PackFile, f *File, parsedNote *ParsedNote) (*Note, error) {
	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingNote, err := CurrentRepository().FindNoteByTitle(f.RelativePath, parsedNote.Title)
	if err != nil {
		return nil, err
	}
	if existingNote != nil {
		existingNote.update(packFile, f, parsedNote)
		return existingNote, nil
	}
	return NewNote(packFile, f, parsedNote)
}

func (n *Note) update(packFile *PackFile, f *File, parsedNote *ParsedNote) {
	stale := false

	// Set basic properties
	if n.FileOID != f.OID {
		n.FileOID = f.OID
		n.RelativePath = f.RelativePath
		stale = true
	}

	if n.Title != parsedNote.Title {
		n.Title = parsedNote.Title
		stale = true
	}
	if n.Content != parsedNote.Content {
		n.Content = parsedNote.Content
		stale = true
	}

	n.PackFileOID = packFile.OID
	n.IndexedAt = packFile.CTime

	if stale {
		n.UpdatedAt = packFile.CTime
	}
}

func (n *Note) Kind() string {
	return "note"
}

func (n *Note) UniqueOID() oid.OID {
	return n.OID
}

func (n *Note) ModificationTime() time.Time { // TODO useful?
	return n.UpdatedAt
}

/* Index Management */

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

/* DB Management */

func (n *Note) Save() error {
	query := `
		INSERT INTO note(
			oid,
			packfile_oid,
			file_oid,
			relative_path,
			title,
			content_raw,
			created_at,
			updated_at,
			indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			file_oid = ?,
			relative_path = ?,
			title = ?,
			content_raw = ?,
			updated_at = ?,
			indexed_at = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		n.OID,
		n.PackFileOID,
		n.FileOID,
		n.RelativePath,
		n.Title,
		n.Content,
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
		// Update
		n.PackFileOID,
		n.FileOID,
		n.RelativePath,
		n.Title,
		n.Content,
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindNoteByTitle(relativePath, title string) (*Note, error) { // TODO useful?
	var n Note
	var createdAt string
	var updatedAt string
	var indexedAt string

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			relative_path,
			title,
			content_raw,
			created_at,
			updated_at,
			indexed_at
		FROM note
		WHERE relative_path = ? and title = ?;`, relativePath, title).
		Scan(
			&n.OID,
			&n.PackFileOID,
			&n.FileOID,
			&n.RelativePath,
			&n.Title,
			&n.Content,
			&createdAt,
			&updatedAt,
			&indexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.IndexedAt = timeFromSQL(indexedAt)

	return &n, nil
}
