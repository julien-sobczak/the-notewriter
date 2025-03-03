package main

import (
	"database/sql"
	"io"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/file.go

type File struct {
	// A unique identifier among all files
	OID oid.OID `yaml:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`
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
	IndexedAt time.Time `yaml:"-"`
}

func NewFile(packFile *PackFile, parsedFile *ParsedFile) (*File, error) {
	file := &File{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		RelativePath: parsedFile.RelativePath,
		Size:         parsedFile.Markdown.Size,
		MTime:        parsedFile.Markdown.MTime,
		Hash:         helpers.Hash(parsedFile.Markdown.Content),
		Body:         parsedFile.Markdown.Body.String(),
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
	}

	return file, nil
}

func NewOrExistingFile(packFile *PackFile, parsedFile *ParsedFile) (*File, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingFile, err := CurrentRepository().LoadFileByPath(parsedFile.RelativePath)
	if err != nil {
		return nil, err
	}
	if existingFile != nil {
		err := existingFile.update(packFile, parsedFile)
		return existingFile, err
	}
	return NewFile(packFile, parsedFile)
}

func (f *File) update(packFile *PackFile, parsedFile *ParsedFile) error { // TODO useful?
	stale := false

	md := parsedFile.Markdown

	// Check if local file has changed
	if f.MTime != md.MTime || f.Size != md.Size {
		stale = true

		f.Size = md.Size
		f.MTime = md.MTime
		f.Hash = helpers.Hash(md.Content)
		f.Body = md.Body.String()
	}

	f.PackFileOID = packFile.OID
	f.IndexedAt = packFile.CTime

	if stale {
		f.UpdatedAt = packFile.CTime
	}

	return nil
}

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() oid.OID {
	return f.OID
}

func (f *File) ModificationTime() time.Time { // TODO useful?
	return f.MTime
}

/* Index Management */

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

/* DB Management */

func (f *File) Save() error {
	query := `
		INSERT INTO file(
			oid,
			packfile_oid,
			relative_path,
			body,
			created_at,
			updated_at,
			indexed_at,
			mtime,
			size,
			hashsum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			relative_path = ?,
			body = ?,
			updated_at = ?,
			indexed_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		f.OID,
		f.PackFileOID,
		f.RelativePath,
		f.Body,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.IndexedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		// Update
		f.PackFileOID,
		f.RelativePath,
		f.Body,
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

func (r *Repository) LoadFileByPath(relativePath string) (*File, error) { // TODO useful?
	var f File
	var createdAt string
	var updatedAt string
	var indexedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			packfile_oid,
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
			&f.PackFileOID,
			&f.RelativePath,
			&f.Body,
			&createdAt,
			&updatedAt,
			&indexedAt,
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
	f.IndexedAt = timeFromSQL(indexedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}
