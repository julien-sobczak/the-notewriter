package markdown

/*
When do the post-processing?

- MarkdownFile
  => Works with non-native markdown (replacement characters, HTML comments, etc.)
  => Cannot do some post-processing (ex: implicit quote syntax)
- ParsedFile (_The NoteWriter_ syntax)
  => Can do everything (except note inclusion)
  => Store raw and processed Markdown?

When to do note-inclusion?

At save time => the referenced note can be saved later (two-pass saving algorithm)
When the included note is udated/deleted => Resave depending notes by following links?

Must save original markdown because how do you know you must reevaluate the included note otherwise
=> Add (n *Note) EmbeddedNoteReferences()
*/

// TODO create parser.go with
// type ParsedFile {
//    ParsedNotes []*ParsedNote
//    ParsedMedias []*ParsedMedias
//    ParsedLinks []*ParsedLinks
// }

// TODO on ParsedFile() ParsedMedia(), add SHA1()

/*

File.GetObjects()

All objects (file, note, reminder, link) are present in a given file
Media objects can be referenced from multiple files
Notes are present inside a file (and inside another note)
Flashcard are indissociable from their note.
Reminder references a specific note (or a specific item in a note)
Go Links are present in a note but are independant.

NewParsedFileFromMarkdownFile(mdFile) *ParsedFile


    MarkdownFile ------> ParsedFile ---------> File -------------> PackFile

    I understand        I extract              Core logic          I bundle
    Markdown syntax     _NoteWriter_ objects                       _NoteWriter_ objects

    <---- Stateless ----------------> <------- Stateful ------------------->

	<----- Env agnostic ------------> <----- Env specific (config, ...) --->


Option 1: Parsed when needed (ex: `GetLinks` on `Note`)
* Advantage(s):
  * `ParsedXXX` object transparent

Option 2: Parsed everything immediately (ex: `ParseFile` calls `ParseNote`. `ParseMedia`, etc.)
* Advantage(s):
  * Clear separation of logic (parsing <> database interaction)
  * Unique place to test parsing logic (without interaction with DB) (`parser_test.go`)
  * Easier interface for lint rules
* Drawback(s):
  * Useful parsing? (ex: in Linter => in practice, we can expect a rule to validate almost anything)

Decision: Option 2 wins

MarkdownFile / ParsedFile => Stateless
File/PackFile => Stateful

file := parsedFile.ToFile()
file := NewFileFromParsedFile(parsedFile)
file.Save()

packFile := file.ToFile()
packFile := NewPackFileFromFile(file)
packFile.Save() // Write to .nt/objects

packFile := NewPackFileFromPath(path)
file := packFile.ToFile()
file.Save() // refresh the DB


$ nt add
-> packFile.Save()
-> index.StagingArea

$ nt commit
-> index.StagingArea -> index // update object to packfile OID
-> gc()

$ nt reset
-> read index.StagingArea
  -> updated/deleted objects => reread last packfile based on index + .Save()
  -> added objects => read from DB + .Delete()

PackObject is an entry inside a PackFile

> The packfile is a single file containing the contents of all the objects that were removed from your filesystem. The index is a file that contains offsets into that packfile so you can quickly seek to a specific object.
=> Don't use binary files for debuggablity purposes. Use YAML file instead (even if performance are decreased)
*/
