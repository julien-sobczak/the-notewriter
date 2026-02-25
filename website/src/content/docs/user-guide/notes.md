---
title: Notes
---

## Declaring Notes

Notes are defined using a Markdown heading starting with a known type: `Note`, `Quote`, `Flashcard`, etc.

You may find the default types declared in `.nt/nt.libsonnet`:

```jsonnet title=.nt/nt.libsonnet
{
    DefaultNoteTypes: {
        Note: {
            name: "Note",
        },
        Task: self.Note + {
            name: "Task",
        },
        // ...
    }
}
```

Types identify the purpose of your notes. Different notes will be processed differently. For example, flashcards must be studied, tasks should be planned, and quotes could be rendered for inspiration.

Here is an example using different note types:

```md title=notebook.md
# My Notebook

## Note: A thought

This is a note.

## Quote: A quote

This is a quote.
```

The content of notes can be structured using Markdown headings. For example:

```md
## Note: A Structured Note

A first sentence.

### Subsection 1

A first subsection.

### Subsection 2

A second subsection.
```

The subsections "Subsection 1" and "Subsection 2" are included in the note `A Structured Note`. They are not individual notes.

Notes can also be declared below other notes. For example:

```md
## Note: Go vs Golang

When the language was released by Google in 2009, the team registered the website golang.org because go.org was not available.

### Flashcard: Go vs Golang

Why does **Go** sometimes called **Golang**?

---

`go.org` was not available => `golang.org` 👍
```

This syntax is convenient to inherit [attributes](./attributes.md) and add implicit [links](./links) between your notes.


## Annotating Notes

All notes can end with an annotation (using the Markdown syntax for quotations):

```md
## Quote: Wayne Gretzky on Trying

You miss 100% of the shots you don’t take.

> Trying will always be more effective than doing nothing.
```

These optional comments are useful to explain why a note resonates, or to summarize the key idea using your own words. These comments are highlighted differently (and can be omitted) when rendered in _The NoteWriter Desktop_.


## Embedding Notes

Notes can be embedded inside another notes using the unofficial Markdown syntax `![[wikilink-to-file#note-section]]`


:::important

The wikilink must include a reference to a Markdown heading defining a note. You cannot import a file like this `![[wikilink-to-file]]` but only a note inside a file.

:::

Example:

```md title=notes.md
## Quote: Tim Ferris on Productivity

`@author: Tim Ferris`

Focus on being productive instead of busy.

## Note: On Busyness

Productivity is doing the right thing. Doing less useless things, and doing more important things.

![[#Quote: Tim Ferris on Productivity]]
```

The second note is equivalent to:

```md
## Note: On Busyness

Productivity is doing the right thing. Doing less useless things, and doing more important things.

> Focus on being productive instead of busy.
>
> — Tim Ferris
```

## Ignoring Notes

Like files, some notes can be ignored using the special tag `ignore` (more about tags later):

```md
## Note: A Parsed Note

This note will be indexed.

## Note: An Ignored Note

`#ignore`

This note will be ignored.
```

:::tip

* **Use types to classify your notes**, to make easy to retrieve them or to restrict when searching for specific notes.
* **Use personal comment to remember why a note resonates** and help increase knowledge retention.
* **Use embedded notes** to avoid duplication when quoting other notes.
:::


:::tip[Under the hood]

Notes are the building block of _The NoteWriter_. When adding a file using `nt add`, the application will parse the Markdown content to search for headings matching one of the declared note types to extract a new object for every note. All objects present in a file are stored in the same pack file under `.nt/objects`.

Let's try to update the file created in the previous [page](./files):

```shell
$ echo <<'EOF'
# My Notes

## Note: Example

This is a **note**.

## Quote: Example

To **quote** or not to quote.*
EOF

$ nt add notes.md
$ tree .nt/objects
.nt/objects
├── d2
│   └── d2810d9388293838c8edac1654ecb2bb440f4237.pack
└── da
    └── da79a1a26d429d0250f1ae89ab9f0dd6bbda65d9.pack
```

A new object `da79a1a26d429d0250f1ae89ab9f0dd6bbda65d9.pack` has been created. Objects are immutable. Everytime a file is edited, a new pack file will be recreated and the old pack file will be deleted automatically by the garbage collector (or when running `nt gc` manually).

We now have more objects extracted from the Markdown file:

```shell
$ cat .nt/objects/da/da79a1a26d429d0250f1ae89ab9f0dd6bbda65d9.pack
oid: da79a1a26d429d0250f1ae89ab9f0dd6bbda65d9
file_relative_path: notes.md
file_mtime: 2026-01-01T15:37:51.705068578+01:00
file_size: 101
ctime: 2026-01-01T15:38:00.132144+01:00
kind: objects
objects:
    - oid: 76913e4a281449efa5eade3702a0cda25434f3b6
      kind: file
      desc: file "notes.md" [76913e4a281449efa5eade3702a0cda25434f3b6]
      data: eJyUk...
    - oid: d67cf83411f545279a3b7d9181f1d1901bb2cce3
      kind: note
      desc: 'note "Note: Example" [d67cf83411f545279a3b7d9181f1d1901bb2cce3]'
      data: eJyMj...
    - oid: e012f232479743ad80ad9d8b1c6b60fc3b3bcb88
      kind: note
      desc: 'note "Quote: Example" [e012f232479743ad80ad9d8b1c6b60fc3b3bcb88]'
      data: eJyMk...
```

_The NoteWriter_ has extracted 3 objects: one object `file` representing the Markdown file itself, and two objects `note` representing the notes present in the document. The `data` is compressed in Zlib and encoded in base64 to store it as a YAML attribute. The value can be decompressed using standard utility commands or using the special command `nt cat-file`:

```shell
$ nt cat-file e012f232479743ad80ad9d8b1c6b60fc3b3bcb88
oid: e012f232479743ad80ad9d8b1c6b60fc3b3bcb88
slug: notes-quote-example
type: Quote
title: 'Quote: Example'
relative_path: notes.md
wikilink: 'notes#Quote: Example'
line: 7
body: '> To **quote** or not to quote.*'
created_at: 2026-01-01T15:38:00.132144+01:00
updated_at: 2026-01-01T15:38:00.132144+01:00
indexed_at: 2026-01-01T15:38:00.132144+01:00
```

These objects are also present in the SQLite database. For examples, notes can be found in table `notes`:

```shell
$  sqlite3 .nt/database.db "SELECT oid, slug, title, relative_path FROM note";
d67cf83411f545279a3b7d9181f1d1901bb2cce3|notes-note-example|Note: Example|notes.md
e012f232479743ad80ad9d8b1c6b60fc3b3bcb88|notes-quote-example|Quote: Example|notes.md
```

The SQLite database uses the FTS5 Extension to support full-text searches in the desktop application:
:::


:::note[Useful Commands]

* Use `nt add <path>` after editing a file to parse it again and extract new objects.
* Use `nt cat-file` to describe an object from its object identifier (OID).
:::

