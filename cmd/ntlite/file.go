package main

import (
	"database/sql"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/file.go

type File struct {
	// A unique identifier among all files
	OID string `yaml:"oid"`

	// A relative path to the repository root directory
	RelativePath string `yaml:"relative_path"`

	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime"`

	Body string `yaml:"body"`

	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
	DeletedAt time.Time `yaml:"deleted_at,omitempty"`
	IndexedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingFile(relativePath string) (*File, error) {
	existingFile, err := CurrentRepository().LoadFileByPath(relativePath)
	if err != nil {
		log.Fatal(err)
	}

	if existingFile != nil {
		existingFile.update()
		return existingFile, nil
	}

	parsedFile, err := ParseFile(relativePath)
	if err != nil {
		return nil, err
	}
	return NewFileFromParsedFile(parsedFile), nil
}

func NewFileFromParsedFile(parsedFile *ParsedFile) *File {
	return &File{
		OID:          NewOID(),
		RelativePath: parsedFile.RelativePath,
		Size:         parsedFile.Stat.Size(),
		Hash:         Hash(parsedFile.Bytes),
		MTime:        parsedFile.Stat.ModTime(),
		Body:         parsedFile.Body,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		stale:        true,
		new:          true,
	}
}

func (f *File) update() error {
	absolutePath := filepath.Join(CurrentRepository().Path, f.RelativePath)
	parsedFile, err := ParseFile(absolutePath)
	if err != nil {
		return err
	}

	// Check if local file has changed
	if f.MTime != parsedFile.Stat.ModTime() || f.Size != parsedFile.Stat.Size() {
		f.Size = parsedFile.Stat.Size()
		f.Hash = Hash(parsedFile.Bytes)
		f.MTime = parsedFile.Stat.ModTime()
		f.Body = parsedFile.Body
		f.stale = true
	}

	return nil
}

type ParsedFile struct {
	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Stat
	Stat fs.FileInfo

	// The raw file bytes
	Bytes []byte

	// The body
	Body string
}

// ParseFile contains the main logic to parse a raw note file.
func ParseFile(relativePath string) (*ParsedFile, error) {
	absolutePath := filepath.Join(CurrentRepository().Path, relativePath)

	lstat, err := os.Lstat(absolutePath)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	body := strings.TrimSpace(string(contentBytes))
	return &ParsedFile{
		AbsolutePath: absolutePath,
		RelativePath: relativePath,
		Stat:         lstat,
		Bytes:        contentBytes,
		Body:         body,
	}, nil
}

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() string {
	return f.OID
}

func (f *File) State() State {
	if !f.DeletedAt.IsZero() {
		return Deleted
	}
	if f.new {
		return Added
	}
	if f.stale {
		return Modified
	}
	return None
}

func (f *File) ModificationTime() time.Time {
	return f.MTime
}

func (f *File) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(f)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *File) GetNotes() []*Note {
	parsedNotes := ParseNotes(f.Body)
	if len(parsedNotes) == 0 {
		return nil
	}

	var notes []*Note
	for _, parsedNote := range parsedNotes {
		note := NewOrExistingNote(f, parsedNote)
		notes = append(notes, note)
	}

	return notes
}

func (f *File) SubObjects() []StatefulObject {
	var objs []StatefulObject

	for _, object := range f.GetNotes() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	return objs
}

func (f *File) Save() error {
	var err error
	f.UpdatedAt = time.Now()
	f.IndexedAt = time.Now()
	switch f.State() {
	case Added:
		err = f.Insert()
	case Modified:
		err = f.Update()
	case Deleted:
		err = f.Delete()
	default:
		err = f.Check()
	}
	if err != nil {
		return err
	}
	f.new = false
	f.stale = false
	return nil
}

func (f *File) Insert() error {
	query := `
		INSERT INTO file(
			oid,
			relative_path,
			body,
			created_at,
			updated_at,
			indexed_at,
			mtime,
			size,
			hashsum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := CurrentDB().Client().Exec(query,
		f.OID,
		f.RelativePath,
		f.Body,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.IndexedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Update() error {
	query := `
		UPDATE file
		SET
			relative_path = ?,
			body = ?,
			updated_at = ?,
			indexed_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?
		WHERE oid = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		f.RelativePath,
		f.Body,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.IndexedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.OID,
	)
	return err
}

func (f *File) Delete() error {
	query := `DELETE FROM file WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID)
	return err
}

func (f *File) Check() error {
	client := CurrentDB().Client()
	f.IndexedAt = time.Now()
	query := `
		UPDATE file
		SET indexed_at = ?
		WHERE oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.IndexedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE note
		SET indexed_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.IndexedAt), f.OID); err != nil {
		return err
	}
	return nil
}

func (r *Repository) LoadFileByPath(relativePath string) (*File, error) {
	var f File
	var createdAt string
	var updatedAt string
	var lastIndexedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			relative_path,
			body,
			created_at,
			updated_at,
			indexed_at,
			mtime,
			size,
			hashsum
		FROM file
		WHERE relative_path = ?;`, relativePath).
		Scan(
			&f.OID,
			&f.RelativePath,
			&f.Body,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
			&mTime,
			&f.Size,
			&f.Hash,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.IndexedAt = timeFromSQL(lastIndexedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}
