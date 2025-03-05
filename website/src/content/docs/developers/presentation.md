---
title: Presentation
---

_The NoteWriter_ is a CLI to extract objects from Markdown files.

Users edit Markdown files (with a few extensions). _The NoteWriter_ CLI parses these files to extract different objects (notes, flashcards, reminders, etc.).


## Code Organization

```
.
├── cmd             # Viper commands
├── internal        # The NoteWriter-specific code
│   ├── core        # Main logic
│   ├── markdown    # Markdown-specific parsing
│   ├── medias      # Media processing
│   └── testutil    # Test utilities
└── pkg             # "Reusable" code (not specific to The NoteWriter)
```

:::tip

Start with commands under `cmd/` when inspecting code to quickly locate the interesting lines of code.

:::

The repository also contains additional directories not directly related to the implementation:

```
.
├── build      # Binary built using the Makefile
├── example    # A demo repository of notes
└── website    # This documentation
```


## Implementation


### Core (`internal/core`)

Most of the code (and most of the tests) is present in this package.

A `Repository` (`repository.go`) is the parent container. A _repository_ traverses directories to find `markdown.File` (`markdown/file.go`) using the method `Walk`:

```go
r := core.CurrentRepository() // look for the top directory containing .nt/
err := r.Walk(pathSpecs, func(md *markdown.File) error {
    relativePath, err := r.GetFileRelativePath(md.AbsolutePath)
    if err != nil {
        return err
    }
    fmt.Printf("Found Markdown file to process: %s", relativePath)
}
```

:::note

_The NoteWriter_ relies heavily on [singletons](https://en.wikipedia.org/wiki/Singleton_pattern). Main abstractions (`Repository`, `DB`, `Index`, `Config`) can be retrieved using methods `CurrentRepository()`, `CurrentDB()`, `CurrentIndex()`, `CurrentConfig()` to easily find a note, persist changes in database, or read configuration settings anywhere in the code. (Singletons are only initialized on first use.)

**This strongly differs from most enterprise applications** where layers and dependency injection are used to have a clean separation of concerns.

**_The NoteWriter_ is a CLI to execute short-lived commands** (one execution = one "transaction") where traditional applications process transactions in parallel (one request = one transaction).

:::

Markdown files are parsed (`parser.go`) to extract objects:

```go
parsedFile, _ := core.ParseFile(relativePath, mdFile)
```

* `ParsedFile` represents the Markdown file.
* `ParsedNote` (see field `Notes` in `ParsedFile`) represents the nodes defined using Markdown headings with a given kind.
* `ParsedFlashcard` (see optional field `Flashcard` in `ParsedNote`) are notes using the kind `Flashcard`.
* `ParsedLink` (see field `GoLinks` in `ParsedNote`) are Go links present in link titles using the defined convention.
* `ParsedReminder` (see fields `Reminders` in `ParsedNote`) are special tags defining one or more successive dates in the future when a note must be reviewed.
* `ParsedMedias` (see fields `Medias` in `ParsedFile) are references to local medias files present in the same repository using the Markdown image syntax inside a note.

Almost the entire parsing and enriching logic (`parser.go`) such as attribute management is processed during this initial parsing.

Parsed objects are then converted into different `Object` (`object.go`). A `ParsedFile` becomes a `File` (`file.go`), a `ParsedNote` becomes a `Note` (`note.go`), and so on.


All objects must satisfy this interface:

```go title=core/object.go
type Dumpable interface {
	ToYAML() string
	ToJSON() string
	ToMarkdown() string
}

type Object interface {
	Dumpable

	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() oid.OID
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// Relations returns the relations where the current object is the source.
	Relations() []*Relation

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}
```

This interface makes easy to factorize common logic betwen objects (ex: all objects can reference other objects and be dumped to YAML inside `.nt/objects`).

Each _object_ is uniquely defined by an OID (a 40-character string) randomly generated from a UUID (see `pkg/oid/oid.go`), except in tests where the generation is reproducible.

:::tip

Use the [command `nt cat-file <oid>`](../reference/commands/nt-cat-file.md) to find the object from an OID.

:::

Each _object_ can be `Read()` from a YAML document and `Write()` to a YAML document using the common Go abstractions `io.Reader` and `io.Writer`.

All objects extracted from Markdown files must also be stored in a relational database using SQLite.

```go title=core/object.go
// StatefulObject to represent the subset of updatable objects persisted in database.
type StatefulObject interface {
	Object

	// Save persists to DB
	Save() error
	// Delete removes from DB
	Delete() error
}
```

The methods `Save()` and `Delete()` are commonly implemented using the singleton `CurrentDB()` to retrieve a connection to the database. `Save()` relies on the "upsert" support in SQLite.

Some _objects_ (`File` and `Media`) also satisfy the interface `FileObject`:

```go title=core/object.go
// FileObject represents an object present as a file in the repository.
type FileObject interface {
	// UniqueOID of the object representing the file
	UniqueOID() oid.OID

	// Relative path to repository
	FileRelativePath() string
	// Timestamp of last content modification
	FileMTime() time.Time
	// Size of the file
	FileSize() int64
	// MD5 Checksum
	FileHash() string

	Blobs() []*BlobRef
}
```

In addition to methods exposing various information about the underlying source file, these `FileObject` could also include blobs (`BlobRef`):

* `File` generates a unique blob containing the original Markdown file. Retrieving the original file is useful in some edge cases.
* `Media` generates different blobs corresponding to the different binary files generated from the source [medias](#medias).


Now that all objects have been generated, they need to be persisted. Stateful objects are grouped into different `PackFile` (`packfile.go`). We create a pack file per path inside the repository. This means that we create one pack file for every Markdown file (`*.md`) and one pack file for every media (`.png`, `*.mp3`, ...) referenced by these files:

```go
var newPackFiles []*core.PackFile
for _, parsedMedia := range parsedFile.Medias {
	packMedia, err := core.NewPackFileFromParsedMedia(parsedMedia)
	if err != nil {
		return err
	}
	newPackFiles = append(newPackFiles, newPackFiles)
}

packFile, err := NewPackFileFromParsedFile(parsedFile)
if err != nil {
	return err
}
newPackFiles = append(newPackFiles, packFile)
```

Pack files are stored in the index (under the directory `.nt/objects`). They are especially useful to limit the number of files on disk. A markdown file containing thousands of notes will be saved as a single pack file on disk.

The `Index` (`index.go`) is used to keep track of all saved pack files. This inventory is physically saved in `.nt/index`. New pack files can be added easily:

```go
idx := CurrentIndex()
idx.Stage(newPackFiles...)
idx.Save()
```

Working with pack files is not ideal. We want to search notes using keywords, or quickly find the URL of a go link. In complement to the index, we saved pack files (`database.go`) inside a SQLite database (stored in `.nt/database.db`). The `Database` is basically a component to save/delete pack files, with additional methods to answer common queries.

In pratice, adding new pack files in the index and in the database looks more like this:

```go
// We saved pack files on disk before starting a new transaction to keep it short
if err := db.BeginTransaction(); err != nil {
	return err
}
db.UpsertPackFiles(packFilesToUpsert...)
db.Index().Stage(packFilesToUpsert...)

// Don't forget to commit
if err := db.CommitTransaction(); err != nil {
	return err
}
// And to persist the index
if err := db.Index().Save(); err != nil {
	return err
}
```

We now have a good overview of the different steps required to convert a Markdown file to a list of extracted objects and persisted on disk and database. The following schema illustrates these steps:

```
     📄       (parsing)
markdown.File     ➡     core.ParsedFile   ➡  core.File  ⏋➡
                        core.ParsedNote      core.Note  ⏌
                        core.ParsedMedia     core.Media ⏋
						...                  ...
```


### Linter `internal/core/lint.go`

The [command `nt lint`](../reference/commands/nt-lint.md) check for violations. All files are inspected (rules may have changed even if files haven't been modified). The linter reuses the method `Walk` to traverse the _repository_. The linter doesn't bother with stateful objects and reuses the types `ParsedFile`, `ParsedNote`, `ParsedMedia` to find errors.

Each rule is defined using the type `LintRule`:

```go title=internal/core/lint.go
type LintRule func(*ParsedFile, []string) ([]*Violation, error)
```

For example, we can write a custom rule (not supported) to validate a file doesn't contains more than 100 notes.

```go
func MyCustomRule(file *ParsedFile, args []string) ([]*Violation, error) {
	var violations []*Violation

	notes := ParseNotes(file.Body)
    if len(notes) > 100 {
        violations = append(violations, &Violation{
            Name:         "my-custom-rule",
            RelativePath: file.RelativePath,
            Message:      "too many notes",
        })
	}

	return violations, nil
}
```

Each rule must be declared in the global variable `LintRules` in the same file:

```go title=internal/core/lint.go
var LintRules = map[string]LintRuleDefinition{
    // ...
	"my-custon-rule": {
		Eval: MyCustomRule,
	},
}
```


### Media (`internal/medias`)

Medias are static files included in notes using the image syntax:

```md
![](audio.wav)
![](picture.png)
![](video.mp4)
```

When processing these _medias_, _The NoteWriter_ will create blobs inside the directory `.nt/objects/`. The OID is the SHA1 determined from the file content.

Images, videos, sounds are processed. _The NoteWriter_ will optimize these medias like this:

* Images are converted to AVIF in different sizes ("preview" = mobile and grid view, "large" = full-size view, "original" = original size).
* Audios are converted to MP3.
* Videos are converted to WebM and a preview image is generated from the first frame.

The AVIF, MP3, and WebM formats are used for their great compression performance and their support (including on mobile devices).

By default, _The NoteWriter_ uses the [external command `ffmpeg`](https://ffmpeg.org/) (`internal/medias/ffmpeg`) to convert and resize medias. All converters must satisfy this interface:

```go title=internal/medias/converters.go
type Converter interface {
	ToAVIF(src, dest string, dimensions Dimensions) error
	ToMP3(src, dest string) error
	ToWebM(src, dest string) error
}
```

For example, we can draft a note including a large picture:

```shell
$ mkdir notes
$ cd notes
$ echo "# My Notes\n\n## Artwork: Whale\n\n![](medias/whale.jpg)" > notes.md
$ nt init
$ nt add .
$ nt commit
[9ef2100625aa4d5c913b8010516fb9a1cd6add98]
 3 objects changes, 3 insertion(s)
 create file "notes.md" [10a76fcada5a4336bb427b68f23d9690b5ebec33]
 create note "Artwork: Whale" [fed3aa2ace7a4fcb889af7f149bda0d6c802cf43]
 create media medias/whale.jpg [72f94476596d47568e617292ab93e02b64032159]
$ notes nt cat-file 72f94476596d47568e617292ab93e02b64032159
oid: 72f94476596d47568e617292ab93e02b64032159
relative_path: medias/whale.jpg
kind: picture
dangling: false
extension: .jpg
mtime: 2023-01-01T12:00
hash: 27198c1682772f01d006b19d4a15018463b7004a
size: 6296968
blobs:
    - oid: 5ac8980e0206c51e113191f1cfa4aab3e40b671a
      mime: image/avif
      tags:
        - preview
        - lossy
    - oid: 40100b2a68ecf7048566a901d6766be8f85ed186
      mime: image/avif
      tags:
        - large
        - lossy
    - oid: 7b4bf88e47e7f782ae9b11e89414d4f66782eeea
      mime: image/avif
      tags:
        - original
        - lossy
created_at: 2023-01-01T12:00
updated_at: 2023-01-01T12:00
```

You can open the generated file. Ex (MacOS):

```shell
$ open -a Preview .nt/objects/5a/5ac8980e0206c51e113191f1cfa4aab3e40b671a
```


## Testing

_The NoteWriter_ works with files. Testing the application by mocking interactions with the file system would be cumbersome.

:::tip

Almost all tests interacts with the file system and executes SQL queries on a SQLite database instance. Their execution time on a SSD machine are relatively low (~10s to run ~500 tests).

Only external commands like `ffmpeg` are impersonated by the test binary file (popular technique used by Golang to test `exec` package).

:::

The package `internal/testutil` exposes various functions to duplicate a `testdata` directory. These functions are reused by functions inside `internal/core/testing.go` to provide a valid repository:

1. Copy Markdown files present under `internal/core/testdata` (aka golden files).
2. Init a valid `.nt` directory and ensure `CurrentRepository()` reads from this repository.
3. Return the temporary directory (automatically cleaned after the test completes)

Example (`SetUpRepositoryFromGoldenDirNamed`):

```go
package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandAdd(t *testing.T) {
    SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

    err := CurrentRepository().Add("go.md")
    require.NoError(t, err)
}
```

Various methods exist:

* `SetUpRepositoryFromGoldenFile` initializes a repository containing a single file named after the test (`TestCommandAdd` => `testdata/TestCommandAdd.md`).
* `SetUpRepositoryFromGoldenFileNamed` is identical to previous function but accepts the file name.
* `SetUpRepositoryFromGoldenDir` initializes a repository from a directory named after the test (`TestCommandAdd` => `testdata/TestCommandAdd/`).
* `SetUpRepositoryFromGoldenDirNamed` is identical to previous function but accepts the directory name.

:::tip

Many tests share the same fixture like `internal/core/testdata/TestMinimal/` (= minimal number of files to demonstrate the maximum of features). Indeed, writing new Markdown files for every test would represent many lines of Markdown to maintain. The recommendation is to reuse `TestMinimal` as much as possible when the logic is independant but create a custom test fixture when testing special cases.

Here are the common fixtures:

* `TestMinimal`: A basic set of files using most of the features.
* `TestMedias`: A basic set of files using all supported medias file types.
* `TestPostProcessing`: A basic set exposing all post-processing rules applies to raw notes.
* `TestLint`: A basic set exposing violations for every rules.
* `TestRelations`: A basic set of inter-referenced notes.

:::

In addition, several utilities are sometimes required to make tests reproductible:

* `core.FreezeNow()` and `core.FreezeAt(time.Time)` ensure successive calls to `clock.Now()` returns a precise timestamp.
* `oid.UseNext(...oid.OID)`, `oid.UseFixed(oid.OID)`, and `oid.UseSequence()` ensure generated OIDs are deterministic (using respectively a predefined sequence of OIDs, the same OIDs, or OIDs incremented by 1).

Use these methods is safe. Test helpers `SetUpXXX` restore the initial configuration using `t.Cleanup()`.

```go
func TestHelpers(t *testing.T) {
    SetUpRepositoryFromTempDir(t) // empty repository

    oid.UseSequence(t) // 0000000000000000000000000000000000000001
                       // 0000000000000000000000000000000000000002
                       // ...
    FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
    // clock.Now() will now always return 2023-01-1T12:30:00Z

    ...
}
```


## F.A.Q.

### How to migrate SQL schema

When the method `CurrentDB().Client()` is first called, the SQL database is read to initialize the connection. Then, the code uses [`golang-migrate`](https://github.com/golang-migrate/migrate/v4) to determine if migrations (`internal/core/sql/*.sql`) must be run.


### How to use transactions with SQLite

Use `CurrentDB().Client()` to retrieve a valid connection to the SQLite database stored in `.nt/database.db`.

```go title=internal/core/note.go
func (r *Repository) CountNotes() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
```

Sometimes, you may want to use transactions. For example, when using `nt add`, if an error occurs when reading a corrupted file, we want to rollback changes to left the database intact. The `DB` exposes methods `BeginTransaction()`, `RollbackTransaction()`, and `CommitTransaction()` for this purpose. Other methods continue to use `CurrentDB().Client()` to create the connection; if a transaction is currently in progress, it will be returned instead and queries will be part of the ongoing transaction.

```go title=internal/core/repository.go
func (r *Repository) Add(pathSpces PathSpecs) error {
	// Run all queries inside the same transaction
	err = db.BeginTransaction()
	if err != nil {
		return err
	}
	defer db.RollbackTransaction()

	// Traverse all given path to add files
	r.Walk(pathSpecs, func(md *markdown.File) error {
		// Do changes in database
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}

	return nil
}
```

Often, the commands update the relational SQLite database and various files inside `.nt` like `.nt/index`. The implemented approach is to write files just after committing the SQL transaction to minimize the risk:

```go title=internal/core/repository.go
func (r *Repository) Add(paths ...string) error {
    ...

	if err := CurrentDB().CommitTransaction(); err != nil {
		return err
	}
	if err := CurrentIndex().Save(); err != nil {
		return err
	}

    return nil
}
```
