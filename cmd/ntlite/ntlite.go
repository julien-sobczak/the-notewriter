package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Missing command")
	}
	command := os.Args[1]
	switch command {
	case "add":
		CurrentRepository().Add()
	case "commit":
		CurrentRepository().Commit()
	default:
		log.Fatalf("Unsupported command %q", command)
	}
}

/* Parser */

// Lite version of internal/core/parser.go

type ParsedFile struct {
	Markdown *markdown.File

	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Notes inside the file
	Notes []*ParsedNote
}

// ParseFile contains the main logic to parse a raw note file.
func ParseFile(relativePath string, md *markdown.File) *ParsedFile {
	result := &ParsedFile{
		Markdown:     md,
		AbsolutePath: md.AbsolutePath,
		RelativePath: relativePath,
	}

	// Extract sub-objects
	result.Notes = result.extractNotes()
	return result
}

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Title   string
	Content string
}

func (p *ParsedFile) extractNotes() []*ParsedNote {
	// All notes collected until now
	var notes []*ParsedNote

	sections, err := p.Markdown.GetSections()
	if err != nil {
		return nil
	}

	for _, section := range sections {
		// Minimalist implementation. Only search for ## headings
		if section.HeadingLevel != 2 {
			continue
		}

		title := section.HeadingText
		body := section.ContentText

		notes = append(notes, &ParsedNote{
			Title:   title.String(),
			Content: strings.TrimSpace(body.String()),
		})
	}

	return notes
}

/* Object */

// Object groups method common to all kinds of managed objects.
type Object interface {
	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() oid.OID

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}

// StatefulObject enriches Object with methods to persist in a database.
type StatefulObject interface {
	Object

	// Save persists to DB
	Save() error
}

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
}

func NewFile(parsedFile *ParsedFile) *File {
	return &File{
		OID:          oid.New(),
		RelativePath: parsedFile.RelativePath,
		Size:         parsedFile.Markdown.Size,
		MTime:        parsedFile.Markdown.MTime,
		Hash:         helpers.Hash(parsedFile.Markdown.Content),
		Body:         parsedFile.Markdown.Body.String(),
	}
}

func NewOrExistingFile(parsedFile *ParsedFile) (*File, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingFile, err := CurrentRepository().FindMatchingFile(parsedFile)
	if err != nil {
		return nil, err
	}
	if existingFile != nil {
		existingFile.update(parsedFile)
		return existingFile, nil
	}
	return NewFile(parsedFile), nil
}

func (f *File) update(parsedFile *ParsedFile) {
	md := parsedFile.Markdown

	// Check if local file has changed
	if f.MTime != md.MTime || f.Size != md.Size {
		f.Size = md.Size
		f.MTime = md.MTime
		f.Hash = helpers.Hash(md.Content)
		f.Body = md.Body.String()
	}
}

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() oid.OID {
	return f.OID
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
			mtime,
			size,
			hashsum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			relative_path = ?,
			body = ?,
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
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		// Update
		f.PackFileOID,
		f.RelativePath,
		f.Body,
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindMatchingFile(parsedFile *ParsedFile) (*File, error) {
	var f File
	var mTime string

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			packfile_oid,
			relative_path,
			body,
			mtime,
			size,
			hashsum
		FROM file
		WHERE relative_path = ?;`, parsedFile.RelativePath).
		Scan(
			&f.OID,
			&f.PackFileOID,
			&f.RelativePath,
			&f.Body,
			&mTime,
			&f.Size,
			&f.Hash,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

// Lite version of internal/core/note.go

type Note struct {
	OID oid.OID `yaml:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	// Title of the note without leading # characters
	Title string `yaml:"title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	Content string `yaml:"content_raw"`
}

// NewNote creates a new note.
func NewNote(file *File, parsedNote *ParsedNote) *Note {
	return &Note{
		OID:          oid.New(),
		Title:        parsedNote.Title,
		RelativePath: file.RelativePath,
		Content:      parsedNote.Content,
	}
}

// NewOrExistingNote loads and updates an existing note or creates a new one if new.
func NewOrExistingNote(f *File, parsedNote *ParsedNote) (*Note, error) {
	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingNote, err := CurrentRepository().FindMatchingNote(f, parsedNote)
	if err != nil {
		return nil, err
	}
	if existingNote != nil {
		existingNote.update(f, parsedNote)
		return existingNote, nil
	}
	return NewNote(f, parsedNote), nil
}

func (n *Note) update(f *File, parsedNote *ParsedNote) {
	if n.RelativePath != f.RelativePath {
		n.RelativePath = f.RelativePath
	}

	if n.Title != parsedNote.Title {
		n.Title = parsedNote.Title
	}
	if n.Content != parsedNote.Content {
		n.Content = parsedNote.Content
	}
}

func (n *Note) Kind() string {
	return "note"
}

func (n *Note) UniqueOID() oid.OID {
	return n.OID
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
			relative_path,
			title,
			content_raw)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			relative_path = ?,
			title = ?,
			content_raw = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		n.OID,
		n.PackFileOID,
		n.RelativePath,
		n.Title,
		n.Content,
		// Update
		n.PackFileOID,
		n.RelativePath,
		n.Title,
		n.Content,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindMatchingNote(f *File, parsedNote *ParsedNote) (*Note, error) {
	return r.FindNoteByTitle(f.RelativePath, parsedNote.Title)
}

func (r *Repository) FindNoteByTitle(relativePath string, title string) (*Note, error) {
	var n Note

	// Query for a value based on a single row.
	if err := CurrentDB().Client().QueryRow(`
		SELECT
			oid,
			packfile_oid,
			relative_path,
			title,
			content_raw
		FROM note
		WHERE relative_path = ? and title = ?;`, relativePath, title).
		Scan(
			&n.OID,
			&n.PackFileOID,
			&n.RelativePath,
			&n.Title,
			&n.Content,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &n, nil
}

/* Pack File Management */

/* ObjectData */

// ObjectData serializes any Object to base64 after zlib compression.
type ObjectData []byte // alias to serialize to YAML easily

// NewObjectData creates a compressed-string representation of the object.
func NewObjectData(obj Object) (ObjectData, error) {
	b := new(bytes.Buffer)
	if err := obj.Write(b); err != nil {
		return nil, err
	}
	in := b.Bytes()

	zb := new(bytes.Buffer)
	w := zlib.NewWriter(zb)
	w.Write(in)
	w.Close()
	return ObjectData(zb.Bytes()), nil
}

func (od ObjectData) MarshalYAML() (any, error) {
	return base64.StdEncoding.EncodeToString(od), nil
}

func (od *ObjectData) UnmarshalYAML(node *yaml.Node) error {
	value := node.Value
	ba, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	*od = ba
	return nil
}

func (od ObjectData) Unmarshal(target interface{}) error {
	if target == nil {
		return fmt.Errorf("cannot unmarshall in nil target")
	}
	src := bytes.NewReader(od)
	dest := new(bytes.Buffer)
	r, err := zlib.NewReader(src)
	if err != nil {
		return err
	}
	io.Copy(dest, r)
	r.Close()

	if f, ok := target.(*File); ok {
		f.Read(dest)
		return nil
	}
	if n, ok := target.(*Note); ok {
		n.Read(dest)
		return nil
	}
	return fmt.Errorf("unsupported type %T", target)
}

/* PackFile */

type PackFile struct {
	OID              oid.OID       `yaml:"oid" json:"oid"`
	FileRelativePath string        `yaml:"file_relative_path" json:"file_relative_path"`
	FileMTime        time.Time     `yaml:"file_mtime" json:"file_mtime"`
	FileSize         int64         `yaml:"file_size" json:"file_size"`
	PackObjects      []*PackObject `yaml:"objects" json:"objects"`
}

type PackObject struct {
	OID  oid.OID    `yaml:"oid" json:"oid"`
	Kind string     `yaml:"kind" json:"kind"`
	Data ObjectData `yaml:"data" json:"data"`
}

// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj Object) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:  obj.UniqueOID(),
		Kind: obj.Kind(),
		Data: data,
	})
	return nil
}

// ReadObject recreates the core object from a commit object.
func (p *PackObject) ReadObject() Object {
	switch p.Kind {
	case "file":
		file := new(File)
		p.Data.Unmarshal(file)
		return file
	case "note":
		note := new(Note)
		p.Data.Unmarshal(note)
		return note
	}
	return nil
}

// LoadPackFileFromPath reads a pack file file on disk.
func LoadPackFileFromPath(path string) (*PackFile, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	result := new(PackFile)
	if err := result.Read(in); err != nil {
		return nil, err
	}
	in.Close()
	return result, nil
}

// Read populates a pack file from an object file.
func (p *PackFile) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&p)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps a pack file to an object file.
func (p *PackFile) Write(w io.Writer) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// SaveTo writes a new pack file to the given location.
func (p *PackFile) Save() error {
	path := filepath.Join(CurrentRepository().Path, ".nt", "objects/"+p.OID.RelativePath(".pack"))
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Write(f)
}

func NewPackFileFromParsedFile(parsedFile *ParsedFile) (*PackFile, error) {
	// Use the hash of the parsed file as OID (if a file changes = new OID)
	packFileOID := oid.MustParse(Hash([]byte(parsedFile.Markdown.Content)))

	packFile := &PackFile{
		OID: packFileOID,

		// Init file properties
		FileRelativePath: parsedFile.RelativePath,
		FileMTime:        parsedFile.Markdown.MTime,
		FileSize:         parsedFile.Markdown.Size,
	}

	// Create objects
	var objects []Object

	// Process the File
	file, err := NewOrExistingFile(parsedFile)
	if err != nil {
		return nil, err
	}
	file.PackFileOID = packFile.OID
	objects = append(objects, file)

	// Process the note(s)
	for _, parsedNote := range parsedFile.Notes {
		note, err := NewOrExistingNote(file, parsedNote)
		if err != nil {
			return nil, err
		}
		note.PackFileOID = packFile.OID
		objects = append(objects, note)
	}

	// Fill the pack file
	for _, obj := range objects {
		if statefulObj, ok := obj.(StatefulObject); ok {
			if err := packFile.AppendObject(statefulObj); err != nil {
				return nil, err
			}
		}
	}

	return packFile, nil
}

/* Index */

// Lite version of internal/core/index.go

// The index file is used to determine if an object is new
// and to quickly locate the pack file containing the object otherwise.
type Index struct {
	// Last commit date
	CommittedAt time.Time `yaml:"committed_at"`
	// List of files known in the index
	Entries []*IndexEntry `yaml:"entries"`
}

// IndexEntry is a file entry in the index.
type IndexEntry struct {
	// Path to the file in working directory
	RelativePath string `yaml:"relative_path"`

	// Pack file OID representing this file under .nt/objects
	PackFileOID oid.OID `yaml:"packfile_oid"`
	// File last modification date
	MTime time.Time `yaml:"mtime"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size" json:"size"`

	// True when a file has been staged
	Staged            bool      `yaml:"staged"`
	StagedPackFileOID oid.OID   `yaml:"staged_packfile_oid"`
	StagedMTime       time.Time `yaml:"staged_mtime"`
	StagedSize        int64     `yaml:"staged_size"`
}

func (i *IndexEntry) Stage(newPackFile *PackFile) {
	i.Staged = true
	i.StagedPackFileOID = newPackFile.OID
	i.StagedMTime = newPackFile.FileMTime
	i.StagedSize = newPackFile.FileSize
}

func (i *IndexEntry) Commit() {
	if !i.Staged {
		return
	}
	i.Staged = false
	i.PackFileOID = i.StagedPackFileOID
	i.MTime = i.StagedMTime
	i.Size = i.StagedSize
	// Clear staged values
	i.StagedPackFileOID = ""
	i.StagedMTime = time.Time{}
	i.StagedSize = 0
}

// ReadIndex loads the index file.
func ReadIndex() *Index {
	path := filepath.Join(CurrentRepository().Path, ".nt/index")
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return &Index{
			Entries: []*IndexEntry{},
		}
	}
	if err != nil {
		log.Fatalf("Unable to open index: %v", err)
	}
	index := new(Index)
	if err := index.Read(in); err != nil {
		log.Fatalf("Unable to read index: %v", err)
	}
	in.Close()
	return index
}

// Save persists the index on disk.
func (i *Index) Save() error {
	path := filepath.Join(CurrentRepository().Path, ".nt/index")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.Write(f)
}

// Read reads an index from the file.
func (i *Index) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&i)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps the index to a file.
func (i *Index) Write(w io.Writer) error {
	data, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// GetEntry returns the entry associated with a file path.
func (i *Index) GetEntry(path string) *IndexEntry {
	for _, entry := range i.Entries {
		if entry.RelativePath == path {
			return entry
		}
	}
	return nil
}

// Stage indexes new pack files.
func (i *Index) Stage(packFiles ...*PackFile) error {
	for _, packFile := range packFiles {
		entry := i.GetEntry(packFile.FileRelativePath)
		if entry == nil {
			entry = &IndexEntry{
				PackFileOID:  packFile.OID,
				RelativePath: packFile.FileRelativePath,
				MTime:        packFile.FileMTime,
				Size:         packFile.FileSize,
			}
			i.Entries = append(i.Entries, entry)
		}
		entry.Stage(packFile)
	}
	return nil
}

// Commit persists the staged changes to the index.
func (i *Index) Commit() error {
	for _, entry := range i.Entries {
		if entry.Staged {
			entry.Commit()
		}
	}
	return i.Save()
}

/* Repository */

// Lite version of internal/core/repository.go

var (
	repositoryOnce      sync.Once
	repositorySingleton *Repository
)

type Repository struct {
	Path string
}

func CurrentRepository() *Repository {
	repositoryOnce.Do(func() {
		var root string
		// Useful in tests when working with repositories in tmp directories
		if path, ok := os.LookupEnv("NT_HOME"); ok {
			root = path
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			root = cwd
		}
		repositorySingleton = &Repository{
			Path: root,
		}
	})
	return repositorySingleton
}

func (r *Repository) Walk(fn func(md *markdown.File) error) error {
	return filepath.WalkDir(r.Path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." || path == ".." {
			return nil
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}

		// We look for Markdown files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// A file found to process!
		md, err := markdown.ParseFile(path)
		if err != nil {
			return err
		}

		if err := fn(md); err != nil {
			return err
		}

		return nil
	})
}

// Add implements the command `nt add`.`
func (r *Repository) Add() error {
	db := CurrentDB()
	index := CurrentIndex()

	var traversedPaths []string
	var packFilesToUpsert []*PackFile

	// Traverse all given paths to detected updated medias/files
	err := r.Walk(func(mdFile *markdown.File) error {
		relativePath, err := filepath.Rel(r.Path, mdFile.AbsolutePath)
		if err != nil {
			log.Fatalf("Unable to determine relative path: %v", err)
		}

		traversedPaths = append(traversedPaths, relativePath)

		entry := index.GetEntry(relativePath)
		if entry != nil && !mdFile.MTime.After(entry.MTime) {
			// Nothing changed = Nothing to parse
			return nil
		}

		// Reparse the new version
		parsedFile := ParseFile(relativePath, mdFile)

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}
		if err := packFile.Save(); err != nil {
			return err
		}
		packFilesToUpsert = append(packFilesToUpsert, packFile)

		return nil
	})
	if err != nil {
		return err
	}

	// We saved pack files on disk before starting a new transaction to keep it short
	if err := db.BeginTransaction(); err != nil {
		return err
	}
	if err := db.UpsertPackFiles(packFilesToUpsert...); err != nil {
		return err
	}
	if err := index.Stage(packFilesToUpsert...); err != nil {
		return err
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := index.Save(); err != nil {
		return err
	}

	return nil
}

// Commit implements the command `nt commit`
func (r *Repository) Commit() error {
	return CurrentIndex().Commit()
}

/* Database Management */

// Lite version of internal/core/database.go

const schema = `
CREATE TABLE file (
	oid TEXT PRIMARY KEY,
	packfile_oid TEXT NOT NULL,
	relative_path TEXT NOT NULL,
	body TEXT NOT NULL,
	mtime TEXT NOT NULL,
	size INTEGER NOT NULL,
	hashsum TEXT NOT NULL
);

CREATE TABLE note (
	oid TEXT PRIMARY KEY,
	packfile_oid TEXT NOT NULL,
	relative_path TEXT NOT NULL,
	title TEXT NOT NULL,
	content_raw TEXT NOT NULL
);
`

var (
	dbOnce      sync.Once
	dbSingleton *DB
)

type DB struct {
	// .nt/index
	index *Index
	// .nt/database.sql
	client *sql.DB
	// In-progress transaction
	tx *sql.Tx
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		dbSingleton = &DB{
			index:  ReadIndex(),
			client: InitClient(),
		}
	})
	return dbSingleton
}

func CurrentIndex() *Index {
	return CurrentDB().index
}

// Client returns the client to use to query the database.
func (db *DB) Client() SQLClient {
	if db.tx != nil {
		// Execute queries in current transaction
		return db.tx
	}
	// Basic client = no transaction
	return db.client
}

func InitClient() *sql.DB {
	db, err := sql.Open("sqlite3", filepath.Join(CurrentRepository().Path, ".nt/database.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Create schema
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("Error while initializing database: %v", err)
	}

	return db
}

// Queryable provides a common interface between sql.DB and sql.Tx to make methods compatible with both.
type SQLClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

/* Transaction Management */

// BeginTransaction starts a new transaction.
func (db *DB) BeginTransaction() error {
	tx, err := db.client.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	db.tx = tx
	return nil
}

// RollbackTransaction aborts the current transaction.
func (db *DB) RollbackTransaction() error {
	if db.tx == nil {
		return errors.New("no transaction started")
	}
	err := db.tx.Rollback()
	db.tx = nil
	return err
}

// CommitTransaction ends the current transaction.
func (db *DB) CommitTransaction() error {
	if db.tx == nil {
		return errors.New("no transaction started")
	}
	err := db.tx.Commit()
	if err != nil {
		return err
	}
	db.tx = nil
	return nil
}

// UpsertPackFiles inserts or updates pack files in the database.
func (db *DB) UpsertPackFiles(packFiles ...*PackFile) error {
	for _, packFile := range packFiles {
		for _, object := range packFile.PackObjects {
			obj := object.ReadObject()
			if statefulObj, ok := obj.(StatefulObject); ok {
				if err := statefulObj.Save(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

/* Helpers */

// OIDToPath converts an oid to a file path.
func OIDToPath(oid oid.OID) string {
	// We use the first two characters to spread objects into different directories
	// (same as .git/objects/) to avoid having a large unpractical directory.
	return oid.String()[0:2] + "/" + oid.String() + ".pack"
}

// Hash is an utility to determine a MD5 hash (acceptable as not used for security reasons).
func Hash(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// timeToSQL converts a time struct to a string representation compatible with SQLite.
func timeToSQL(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	dateStr := date.Format(time.RFC3339Nano)
	return dateStr
}

// timeToSQL parses a string representation of a time to a time struct.
func timeFromSQL(dateStr string) time.Time {
	date, err := time.Parse(time.RFC3339Nano, dateStr)
	if err != nil {
		return time.Time{}
	}
	return date
}
