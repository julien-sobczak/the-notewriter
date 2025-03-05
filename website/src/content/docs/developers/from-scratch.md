---
title: From Scratch
---


The goal of this document is to write a basic version of _The NoteWriter_ to emphasize the core abstractions and the main logic.

:::note

We will implement a basic version supporting only the commands `nt add` and `nt commit`, and only the objects `File` and `Note` (no flashcards, medias, etc.). We ignore configuration too.

The source code is available in this same repository under the directory `cmd/ntlite/`.
:::

## The Model

_The NoteWriter_ extract objects from Markdown files that will be stored inside `nt/objects` in YAML and inside `nt/database.db` using SQL tables (useful to speed up queries and benefit from the full-text search support).

For example:

```md title=notes.md
# My Notes

## Note: Example 1

A first note.

## Note: Example 2

A second note.
```

This document generates 3 objects: 1 _file_ (`notes.md`) and 2 _notes_ (`Note: Example 1` and `Note: Example 2`).

### `File`

Here is the definition of the object `File` simplified for this document:

```go
import "github.com/julien-sobczak/the-notewriter/pkg/oid"

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
```

:::note

The complete model `File` contains additional fields like a reference to a parent file, a title extracted from the text, etc.

:::

Basically, we persist various metadata about the file to quickly determine if a file has changed when running the command `ntlite add`. In addition:

* Each object get assigned an OID (a unique 40-character string like the hash of Git objects). This OID is used as the primary key inside the SQL database and can be used with the official command `nt cat-file <oid>` to get the full information about an object.
* Each object uses Go struct tags to make easy to serialize them in YAML.


### `Note`

Here is the definition of the similar struct `Note`:

```go
import "github.com/julien-sobczak/the-notewriter/pkg/oid"

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
```

:::note

The complete model `Note` contains a lot more fields. Notes are the main building blocks of the application. They can have a list of attributes, tags, a parent note (when notes are nested to inherit from parent attributes).

:::

### `ParsedXXX`

The structs `File` and `Note` must be populated by parsing Markdown files but to make easy to test the parsing logic, we will use basic structs to ignore some of the complexity (id generation, database management, serialization). This is the intent behind the structs `ParsedXXX`:

```go
import "github.com/julien-sobczak/the-notewriter/internal/markdown"

type ParsedFile struct {
	Markdown *markdown.File

	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Notes inside the file
	Notes []*ParsedNote
}

type ParsedNote struct {
	// Heading
	Title   string
	Content string
}
```

The logic to initialize a `ParsedFile` is relatively trivial, in particular when using the custom abstraction `markdown.File` (we hide the logic to parse a Markdown document, this component is ommitted from this document as there is nothing specific to _The NoteWriter_):

```go
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
```

:::note

_The NoteWriter_ supports attributes and tags using a YAML Front Matter and a special syntax. The actual parser extracts these metadata to enrich notes and make them easily searchable.

_The NoteWriter_ also supports nested notes (you can define notes at any level in your Markdown documents) which makes the actual parsing logic slightly more complicated. In addition, the actual logic must also ignore code blocks where `#` is a common character that must not be considered as valid Markdown heading.

:::

`ParsedFile` and `ParsedNote` makes it easy to create `File` and `Note`:

```go
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

func NewNote(file *File, parsedNote *ParsedNote) *Note {
	return &Note{
		OID:          oid.New(),
		Title:        parsedNote.Title,
		RelativePath: file.RelativePath,
		Content:      parsedNote.Content,
	}
}
```

OIDs for objects are not determined from a hash of the content (unlike OIDs for pack files and blobs that we will introduce later). If the content of a note (or a flashcard) is edited, we want to update the old object even if the note was slighly edited or moved. (This differs from Git which stores a new file in this case).


### `PackFile`

Objects like `File` or `Note` are not persisted directly on disk. A repository may contains thousands of notes. We don't want to create thousands of files on disk. Objects are instead packaged inside pack files (similar in principle to [Git packfiles](https://git-scm.com/book/en/v2/Git-Internals-Packfiles)). Objects extracted from the same file are packed inside the same pack file. If a Markdown file contains thousands of notes, a single pack file will be stored on disk.

_The NoteWriter_ extracts different kinds of objects. We cover `File` and `Note` in this document but the actual code support even more object kinds. All these objects satisfy a common interface `Object`:

```go
// Object groups method common to all kinds of managed objects.
type Object interface {
	// Kind returns the object kind to determine which kind of object to create.
	Kind() string // "file", "note"
	// UniqueOID returns the OID of the object.
	UniqueOID() oid.OID
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}
```

Adding support for these methods is trivial. Here is the code for `File`:

```go
func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() oid.OID {
	return f.OID
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
```

We now have an abstraction to work a set of different objects. We can resume our discussion of pack files.

A pack file is basically a container for `Object`. Pack files are stored as YAML files, useful for readability and debugging purposes. To avoid large YAML files, objects inside pack files are not simply serialized in YAML but are converted to `ObjectData` and wrapped into a `PackObject` to preserve essential attributes:

```go
type PackFile struct {
	OID              oid.OID       `yaml:"oid" json:"oid"`
	FileRelativePath string        `yaml:"file_relative_path" json:"file_relative_path"`
	FileMTime        time.Time     `yaml:"file_mtime" json:"file_mtime"`
	FileSize         int64         `yaml:"file_size" json:"file_size"`
	PackObjects      []*PackObject `yaml:"objects" json:"objects"`
}

type PackObject struct {
	OID   oid.OID    `yaml:"oid" json:"oid"`
	Kind  string     `yaml:"kind" json:"kind"`
	Data  ObjectData `yaml:"data" json:"data"`
}
```

Objects are appended using the method `AppendObject`:

```go
// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj Object) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:   obj.UniqueOID(),
		Kind:  obj.Kind(),
		Data:  data,
	})
	return nil
}
```

Pack ojects contains a concise text representation of an object in `Data`. The actual code serializes the object in YAML, compressed it using zlib and encoded the result in Base64 to have a concise text representation. The code is not as complex as it may sound:

```go
import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
)

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
```

The result looks like this:

```yaml
oid: 23334328153429ce5ba99acd83181b06c44f30af
file_relative_path: go.md
file_mtime: 2023-01-01T12:30:00Z
file_size: 1
objects:
    - oid: "8e41f9862553483ca0c8a2b1c1e4ffd1ae413847"
      kind: note
      data: eJykj0+L...0l7ORQ==
```

:::tip

Pack objects can easily be decoded using `nt cat-file <oid>`.

:::

We can now instantiate a pack file from a `ParsedFile`:

```go
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
	file := NewFile(parsedFile)
	file.PackFileOID = packFile.OID
	objects = append(objects, file)

	// Process the note(s)
	for _, parsedNote := range parsedFile.Notes {
		note := NewNote(file, parsedNote)
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
```

Unlike objects, the OID for a pack file is determined from the source file. We determine a OID based on a hash of the file content:

```go
func Hash(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}
```

When a file is edited, we want to recreate a new pack file. The old pack file will be garbage collected.



## The Repository

Now that we know how to parse Markdown files, we need to write the logic to traverse the file system. Most commands need to process the collection of Markdown files, represented by the struct `Repository`:

```go
type Repository struct {
	Path string // The directory containing .nt/
}
```

The repository will be useful from many places inside the code to resolve absolute paths (the actual code contains a lot more methods) and is defined as a singleton (preferable compared to a global variable to initialize it lazily).

```go
var (
	repositoryOnce      sync.Once
	repositorySingleton *Repository
)

func CurrentRepository() *Repository {
	repositoryOnce.Do(func() {
        cwd, err := os.Getwd() // For this tutorial, simply use $CWD
        if err != nil {
            log.Fatal(err)
        }
		repositorySingleton = &Repository{
			Path: cwd,
		}
	})
	return repositorySingleton
}
```

:::tip

The same pattern is used for different global objects: to retrieve the database connection using `CurrentDB()`, the configuration using `CurrentConfig()`, the logger using `CurrentLogger()`, etc. Using singletons can be challenging in some environments, for example when reading the instance from multiple goroutines or to replace with a test double. _The NoteWriter_ is a CLI running short-lived commands and tests are using the same dependencies like SQLite.

:::

We define a convenient method to locate the note files:

```go
func (r *Repository) Walk(fn func(md *markdown.File) error) error {
	filepath.WalkDir(r.Path, func(path string, info fs.DirEntry, err error) error {
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

	return nil
}
```

We will reuse this method several times later but now, we need to have a look at the database.

## The Database

```go
type DB struct {
	// .nt/index
	index *Index
	// .nt/database.sql
	client *sql.DB
}
```

`Index` represents the content of the database (= the inventory of pack files and known OIDs), including the staging area (= the objects that were added using `ntlite add` but still not committed using `ntlite commit`).

```go
type Index struct {
	// Last commit date
	CommittedAt time.Time `yaml:"committed_at"`
	// List of files known in the index
	Entries []*IndexEntry `yaml:"entries"`
}

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
```

The index is a YAML file located at `.nt/index`. We define a few functions and methods to load and dump it:

```go
// ReadIndex loads the index file.
func ReadIndex() *Index {
	path := filepath.Join(CurrentRepository().Path, ".nt/index")
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return &Index{}
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
```

The other attribute of `DB` is the connection to the SQLite database instance located at `.nt/database.db`:

```go
func InitClient() *sql.DB {
	db, err := sql.Open("sqlite3", filepath.Join(CurrentRepository().Path, ".nt/database.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Create the schema
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS file (
	oid TEXT PRIMARY KEY,
	relative_path TEXT NOT NULL,
	body TEXT NOT NULL,
	mtime TEXT NOT NULL,
	size INTEGER NOT NULL,
	hashsum TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS note (
	oid TEXT PRIMARY KEY,
	relative_path TEXT NOT NULL,
	title TEXT NOT NULL,
	content_raw TEXT NOT NULL
);`)
	if err != nil {
		log.Fatalf("Error while initializing database: %v", err)
	}

	return db
}
```

We will use the standard `database/sql` Go package to interact with the database. We will also expose a singleton to make easy to retrieve the connection:

```go
var (
	dbOnce       sync.Once
	dbSingleton  *DB
)

func CurrentDB() *DB {
	dbOnce.Do(func() {
		dbSingleton = &DB{
			index: ReadIndex(),
			client: InitClient(),
		}
	})
	return dbSingleton
}

// Client returns the client to use to query the database.
func (db *DB) Client() *sql.DB {
    // This method will be completed later in this document
	return db.client
}
```

Using this connection, we can now add methods on our model to persist the objects in the database:

```go
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
```

:::note

The method `Save()` for the model `Note` is very similar and omitted for brievity.

:::

Before closing the section, there is still one issue to debate. Using `CurrentDB().Client()` makes easy to execute queries but each query is executed inside a different transaction. When running commands, we will work on many objects at the same time. If a command fails for any reasons, we want to rollback our changes and only report the error. We need to use transactions.

### Transactions

The standard type `sql.DB` exposes a method `BeginTx` that returns a variable of type `*sql.Tx` useful to `Rollback()` or `Commit()` the transaction. This object `sql.Tx` exposes also different methods to query the database, the same methods as offered by `sql.DB`, except there is no common interface between these two types. Ideally, we would like our methods `Save()` to work if there are a transaction in progress or not. To solve this issue, we define an interface:

```go
// Queryable provides a common interface between sql.DB and sql.Tx to make methods compatible with both.
type SQLClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}
```

We define only the few methods used by the application.

We also rework the method `Client()` on `DB` to use this type and to return the default connection when no transaction was started (`*sql.DB`) or the current transaction (`*sql.Tx`):

```go
type DB struct {
	index *Index
	client *sql.DB
	tx *sql.Tx // NEW
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
```

We will now implement the basic commands where these transactions will be indispensable.

## The Commands

### `add`

The command `add` updates the index and the database with new stateful objects:

```go
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
```

We iterate over files using the `Walk()` method. We create a new `ParsedFile` to instantiate a `PackFile` with the function `NewPackFileFromParsedFile` we've covered previously.

The new pack files are then saved in database into the method `UpsertPackFiles`:

```go
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
```

And staged into the index using the method `Stage`:

```go
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

func (i *IndexEntry) Stage(newPackFile *PackFile) {
	i.Staged = true
	i.StagedPackFileOID = newPackFile.OID
	i.StagedMTime = newPackFile.FileMTime
	i.StagedSize = newPackFile.FileSize
}
```

The index is save on disk and the changes in the database committed.


### `commit`

The command `commit` only interacts with the database (`nt/objects`) since the relational database was already updated when adding the files.

The goal of this command is to clear staged objects present inside the index:

```go
func (r *Repository) Commit() error {
	return CurrentIndex().Commit()
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
```

The code iterates over the elements present marked as `Staged` and clear the staged fields.

We are ready for the next batch of files to add. That's all for now.

:::note

**This minimalist version uses the same logic as the complete version**. Here are a few notable differences:

* _The NoteWriter_ uses Viper to have a more informative CLI output and support arguments to commands.
* _The NoteWriter_ uses a [migration tool](https://github.com/golang-migrate/migrate) to create the database schema. (see `initClient` in `internal/core/database.go`)
* _The NoteWriter_ supports more complex Markdown documents (ex: notes can be nested inside other notes), attributes (and tags) using a YAML Front Matter and with a special syntax inside notes.
* _The NoteWriter_ processes notes to enrich their content (support note embedding or sugar syntax for quotes).
* _The NoteWriter_ supports more commands like remotes to push objects and synchronize them between devices. It reuses the file `.nt/index` to compare different trees and find the missing pack files to push/pull.

:::
