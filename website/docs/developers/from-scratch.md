---
sidebar_position: 4
---

# From Scratch

The goal of this document is to write a basic version of _The NoteWriter_ to emphasize the core abstractions and the main logic.

:::info

We will implement a basic version supporting only the commands `nt add` and `nt commit`, and only the objects `File` and `Note` (no flashcards, medias, etc.). We ignore configuration too.

The source code is available in this same repository under the directory `cmd/ntlite/`.
:::

## The Model

_The NoteWriter_ extract objects from Markdown files that will be stored inside `nt/objects` in YAML and inside `nt/database.db` using SQL tables (useful for speed up queries + the full-text search support).

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
type File struct {
	// A unique identifier among all files
	OID string `yaml:"oid"`

	// A relative path to the collection directory
	RelativePath string `yaml:"relative_path"`

	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime"`

	Body string `yaml:"body"`

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}
```

:::info

The complete model `File` contains additional fields like a reference to a parent file, a title extracted from the text, etc.

:::

Basically, we persist various metadata about the file to quickly determine if a file has changed when running the command `ntlite add`. In addition:

* Each object get assigned an OID (a unique 40-character string like the hash of Git objects). This OID is used as the primary key inside the SQL database and can be used with the official command `nt cat-file <oid>` to get the full information about an object.
* Each object includes various timestamps. The creation and last modification dates are mostly informative. The timestamp `LastCheckedAt` is updated every time an object is traversed (even if the object hasn't changed) and is useful to quickly find all deleted objects.
* Each object uses Go struct tags to make easy to serialize them in YAML.
* Each object includes the fields `new` and `stale` to determine if a change must be saved and if the object must be inserted or updated.

### `Note`

Here is the definition of the similar struct `Note`:

```go
type Note struct {
	OID string `yaml:"oid"`

	// File containing the note
	FileOID string `yaml:"file_oid"`

	// Title of the note without leading # characters
	Title string `yaml:"title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	// Content in various formats (best for editing, rendering, writing, etc.)
	Content string `yaml:"content_raw"`
	Hash    string `yaml:"content_hash"`

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}
```

:::info

The complete model `Note` contains a lot more fields. Notes represent the core abstraction. They can have a list of attributes, tags, a parent note (when notes are nested to inherit from parent's attributes), and their content is converted into different representations (Markdown/HTML/Text) to render them easily in various contexts.

:::

### `ParsedXXX`

The structs `File` and `Note` must be populated by parsing Markdown files but to make easy to test the parsing logic, we will use basic structs to ignore some of the complexity (for example, `Note` contains the logic to enrich the Markdown and convert it to HTML). This is the intent behind the structs `ParsedXXX`:

```go
type ParsedFile struct {
	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Stat
	Stat fs.FileInfo

	// The raw content bytes
	Bytes []byte

	// The file content
	Body string
}

type ParsedNote struct {
	// Heading
	Title   string
	// Content inside the heading
	Content string
}
```

The logic to initialize a `ParsedFile` simply uses the standard Go librairies:

```go
// ParseFile contains the main logic to parse a raw note file.
func ParseFile(relativePath string) (*ParsedFile, error) {
	absolutePath := filepath.Join(CurrentCollection().Path, relativePath)

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
```

:::info

_The NoteWriter_ supports attributes and tags using a YAML Front Matter and a special syntax. The actual parser extracts these metadata used to enrich notes and make them easily searchable.

:::

The logic to initialize `ParsedNote` is slightly more elaborate:

```go

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
```

:::info

_The NoteWriter_ supports nested notes (you can define notes at any level in your Markdown documents) which makes the actual parsing logic more obscure. In addition, the actual logic must also ignore code blocks where `#` is a common character that must not be considered as valid Markdown heading.

:::

The target objects can be initalized from these structs easily:

```go
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
```

As explained before, the OIDs uses the same format as Git (SHA1) but are not determined from a hash of the content. If the content of a note (or a flashcard) is edited, we want to update the old object (when Git stores a new file in this case). Therefore, the OID are in fact disguised UUID under the hood:

```go
func NewOID() string {
	// Ex (Git): 5e3f1b351782c017590b4b70fee709bf9c83b050
	// Ex (UUIDv4): 123e4567-e89b-12d3-a456-426655440000

	// Remove `-` + add 8 random characters
	oid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:40]
	return oid
}
```

:::info

SHA1 are only used when storing blobs (aka medias files), not covered in this document.

:::

## The Collection

Now that we know how to parse Markdown files, we need to write the logic to traverse the file system. Most commands will have to process the  complete set of all note files, that are represented by the struct `Collection`:

```go
type Collection struct {
	Path string
}
```

The collection will be useful from many places inside the code to resolve absolute paths (the actual code contains a lot more methods) and is defined as a singleton (preferable compared to a global variable to initialize it lazily).

```go
var (
	collectionOnce      sync.Once
	collectionSingleton *Collection
)

func CurrentCollection() *Collection {
	collectionOnce.Do(func() {
        cwd, err := os.Getwd()
        if err != nil {
            log.Fatal(err)
        }
		collectionSingleton = &Collection{
			Path: cwd,
		}
	})
	return collectionSingleton
}
```

:::tip

The same pattern is used for different global objects: to retrieve the database connection using `CurrentDB()`, the configuration using `CurrentConfig()`, the logger using `CurrentLogger()`, etc. Using singletons can be challenging in some environments, for example when reading the instance from multiple goroutines or to replace with a test double. _The NoteWriter_ is a CLI running short-lived commands and tests are using the same dependencies like SQLite.

:::

We define a convenient method to locate the note files:

```go
func (c *Collection) walk(fn func(path string, stat fs.FileInfo) error) error {
	filepath.WalkDir(c.Path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}

		relativePath, err := filepath.Rel(c.Path, path)
		if err != nil {
			// ignore the file
			return nil
		}

		// We look for only specific extension
		if !info.IsDir() && !strings.HasSuffix(relativePath, ".md") {
			// Nothing to do
			return nil
		}

		// Ignore certain file modes like symlinks
		fileInfo, err := os.Lstat(path) // NB: os.Stat follows symlinks
		if err != nil {
			// Ignore the file
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			// Exclude any file with a mode bit set (device, socket, named pipe, ...)
			// See https://pkg.go.dev/io/fs#FileMode
			return nil
		}

		// A file found to process using the callback
		err = fn(relativePath, fileInfo)
		if err != nil {
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

`Index` represents the content of the database (= a list of known OIDs), including the staging area (= the objects that were added using `ntlite add` but still not committed using `ntlite commit`).

```go
type Index struct {
	Objects     []*IndexObject   `yaml:"objects"`
	StagingArea []*StagingObject `yaml:"staging"`
}

type IndexObject struct {
	OID   string    `yaml:"oid"`
	Kind  string    `yaml:"kind"`
	MTime time.Time `yaml:"mtime"`
}

type StagingObject struct {
	IndexObject
	State State      `yaml:"state"`
	Data  ObjectData `yaml:"data"`
}
```

The index is a YAML file located at `.nt/index`. We define a few functions and methods to load and dump it:

```go
// ReadIndex loads the index file.
func ReadIndex() *Index {
	path := filepath.Join(CurrentCollection().Path, ".nt/index")
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
	path := filepath.Join(CurrentCollection().Path, ".nt/index")
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
	db, err := sql.Open("sqlite3", filepath.Join(CurrentCollection().Path, ".nt/database.db"))
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
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	last_checked_at TEXT,
	mtime TEXT NOT NULL,
	size INTEGER NOT NULL,
	hashsum TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS note (
	oid TEXT PRIMARY KEY,
	file_oid TEXT NOT NULL,
	relative_path TEXT NOT NULL,
	title TEXT NOT NULL,
	content_raw TEXT NOT NULL,
	hashsum TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	last_checked_at TEXT
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
	var err error
	f.UpdatedAt = time.Now()
	f.LastCheckedAt = time.Now()
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
			last_checked_at,
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
		timeToSQL(f.LastCheckedAt),
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
			last_checked_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?
		WHERE oid = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		f.RelativePath,
		f.Body,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
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
	f.LastCheckedAt = time.Now()
	query := `
		UPDATE file
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE note
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	return nil
}
```

That's a lot of code as we are using a low-level library. We have a method for every operation `Insert()`, `Update()`, `Delete()`, and an additional method `Check()` to only update the `LastCheckedAt` timestamp. The method `Save()` determines which method to call based on the attributes `new` and `stale`.

:::info

The method `Save()` for the model `Note` is very similar and omitted for brievity.

:::

Before closing the section, there is still one issue to debate. Using `CurrentDB().Client()` makes easy to execute queries but each query is executed inside a different transaction. When running commands, we will work on many objects at the same time. If a command fails to any reasons, we may want to rollback our changes and only report the error. We need to use transactions.

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

The command `add` updates the database with new objects. For this document, we consider only `File` and `Note` but _The NoteWriter_ manages more object types (`Flashcard`, `Reminder`, `Link`, ...). A common interface between these different types is useful to factorize code. For example, we want to add any type of object to the database in a uniform way. Here are the interfaces:

```go
type Object interface {
	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() string
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// SubObjects returns the objects directly contained by this object.
	SubObjects() []StatefulObject

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}

type StatefulObject interface {
	Object

	// State returns the current state.
	State() State

	// Save persists to DB
	Save() error
}
```

In practice, all objects satisfy the `StatefulObject` interface but we can choose to use one of two types to make explicit if we are interesting in reading the object or updating it.

The implementations of these methods is trivial. We have already covered the method `Save()`. Here are the other methods implemented by the struct `File`:

```go

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

func (f *File) SubObjects() []StatefulObject {
	var objs []StatefulObject

	for _, object := range f.GetNotes() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	return objs
}
```

The last method `SubObjects()` will be particularly useful when processing the collection of notes, since we will create objects of type `File` and use `SubObjects()` to iterate over other sub-objects recursively without having to interact directly with all types of objects.

Here is the code for the command `add`:

```go
func (c *Collection) Add() error {
	db := CurrentDB()

	// Run all queries inside the same transaction
	err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer db.RollbackTransaction()

	// Traverse all files
	err = c.walk(func(relativePath string, stat fs.FileInfo) error {
		file, err := NewOrExistingFile(relativePath)
		if err != nil {
			return err
		}

		if file.State() != None {
			if err := db.StageObject(file); err != nil {
				return fmt.Errorf("unable to stage modified object %s: %v", file.RelativePath, err)
			}
		}
		if err := file.Save(); err != nil {
			return nil
		}

		for _, object := range file.SubObjects() {
			if object.State() != None {
				if err := db.StageObject(object); err != nil {
					return fmt.Errorf("unable to stage modified object %s: %v", object, err)
				}
			}
			if err := object.Save(); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// (Not implemented) Find objects to delete by querying
	// the different tables for rows with last_checked_at < :execution_time

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := db.index.Save(); err != nil {
		return err
	}

	return nil
}
```

We iterate over files using the `walk()` method. We create a new `File` using the newly function `NewOrExistingFile()` whose goal is to check in database if the file is already known and compare for changes:

```go
func NewOrExistingFile(relativePath string) (*File, error) {
	existingFile, err := CurrentCollection().LoadFileByPath(relativePath)
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

func (f *File) update() error {
	absolutePath := filepath.Join(CurrentCollection().Path, f.RelativePath)
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
```

When the `State()` of an object is different from `None` (= the object has changed), we place the object in the staging area (= the objects waiting to be committed). Then, we `Save()` every object to at minimum update their `LastCheckedAt` timestamp.

The only step remaining to be covered in more detail is the method `StageObject()`:

```go
func (db *DB) StageObject(obj StatefulObject) error {
	return db.index.StageObject(obj)
}

func (i *Index) StageObject(obj StatefulObject) error {
	objData, err := NewObjectData(obj)
	if err != nil {
		return err
	}

	// Update staging area
	stagingObject := &StagingObject{
		IndexObject: IndexObject{
			OID:   obj.UniqueOID(),
			Kind:  obj.Kind(),
			MTime: obj.ModificationTime(),
		},
		State: obj.State(),
		Data:  objData,
	}

	i.StagingArea = append(i.StagingArea, stagingObject)

	return nil
}
```

Basically, we append a new `IndexObject` into the slice `StagingArea` defined by the struct `Index`. What is more subtle to understand is the field `Data` where the content of the staged object (can be any type) is serialized. Indeed, the index (and thus the staging area) is serialized in YAML. We serialize the content of all staged objects in `YAML` before compressing it using the package `compress/zlib` and encoding it in Base64 to end up with a simple string in `Data`:

```yaml
staging:
 - oid: 93267c32147a4ab7a1100ce82faab56a99fca1cd
   kind: note
   state: added
   mtime: 2023-01-01T01:12:30Z
   data: eJzEUsFq20AQves...
```

Here is a preview of this code:

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

func (od ObjectData) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(od), nil
}
```

Saving the edited objects is particularly useful for deletions. When running the command `add`, there is a check (commented in the above code) to list all objects which have not be saved (= the objects that no longer exist) and we issue a `DELETE` in database to remove them. This way, the objects disappear from the relational database (and the inverted index) and are not visible from the desktop UI. But if the user decides to run the command `reset` (not supported in this document), we need to restore the content using the field `Data` in the staging area.


### `commit`

The command `commit` only interacts with the database (`nt/objects`) since the relational database was already updated when adding the files.

The goal of this command is to move the objects present inside the staging area to the final objects under `.nt/objects`. The file `.nt/index` must also be updated to empty the staging area and append the new objects in the reference list.

```go
// Commit creates a new commit object and clear the staging area.
func (db *DB) Commit() error {
	// Convert the staging area to object files under .nt/objects
	for _, indexObject := range db.index.StagingArea {
		var object Object

		switch indexObject.Kind {
		case "file":
			var file File
			if err := indexObject.Data.Unmarshal(&file); err != nil {
				return err
			}
			object = &file
		case "note":
			var note Note
			if err := indexObject.Data.Unmarshal(&note); err != nil {
				return err
			}
			object = &note
		}
		objectPath := filepath.Join(CurrentCollection().Path, ".nt/objects", OIDToPath(indexObject.OID))
		if err := os.MkdirAll(filepath.Dir(objectPath), os.ModePerm); err != nil {
			return err
		}
		f, err := os.Create(objectPath)
		if err != nil {
			return err
		}
		defer f.Close()
		err = object.Write(f)
		if err != nil {
			return err
		}
	}
	db.index.ClearStagingArea()

	// Save .nt/index
	if err := db.index.Save(); err != nil {
		return err
	}
	return nil
}
```

The code iterates over the elements present in the slice `StagingArea` and decode/uncompress/unmarshall the objects before creating a new YAML file under `.nt/objects`.

The code ends with a call to the method `ClearStagingArea()` defined like this:


```go

// ClearStagingArea empties the staging area.
func (i *Index) ClearStagingArea() {
	for _, obj := range i.StagingArea {
		i.Objects = append(i.Objects, &obj.IndexObject)
	}
	i.StagingArea = nil
}
```

Objects are migrated from the staging area to the list of all known objects. The command `commit` ends by saving the file `.nt/index`.

We are ready for the next batch of files to add. That's all for now.


:::info

**This minimalist version uses the same logic as the complete version**. Here are a few notable differences:

* _The NoteWriter_ uses Viper to have a more informative CLI output and support arguments to commands.
* _The NoteWriter_ uses a [migration tool](https://github.com/golang-migrate/migrate) to create the database schema. (see `initClient` in `internal/core/database.go`)
* _The NoteWriter_ supports more complex Markdown documents (ex: notes can be nested inside other notes).
* _The NoteWriter_ supports attributes (and tags) defined using a YAML Front Matter and with a special syntax inside notes.
* _The NoteWriter_ processes notes to enrich their content (support note embedding or sugar syntax for quotes).
* _The NoteWriter_ wraps objects inside `.nt/objects` inside `Commit` object, regrouping all changes inside a single file. The OIDs of commits are also saved inside a file `.nt/commit-graph`.
* _The NoteWriter_ supports remotes to push objects and synchronize them between devices. It reuses the file `.nt/commit-graph` to compare the commits and found the ones to push or pull.

:::
