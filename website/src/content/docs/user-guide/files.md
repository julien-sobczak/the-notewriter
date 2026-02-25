---
title: Files
---

## A Markdown File

Notes are written using Markdown files (GitHub Flavored Markdown is supported).

```md title=notes.md
# My Notebook

A file is a Markdown document structured using headings.

## Notes

### Note: Example

Only headings starting with a known prefix such as `Note:` or `Flashcard:` are
considered as notes and processed by _The NoteWriter_.

### Note: Another Example

Notes can use **Markdown syntax** such as code blocks, quotes, images, tables.
```

A YAML Front Matter can also present at the top of a file to declare common [attributes](./attributes):

```md title=review.md
---
draft: true
---

# Note: My Review

It's a book containing pages.
```

## Many Markdown Files

You can organize your Markdown files using a tree structure. Check the [directory `examples/`](https://github.com/julien-sobczak/the-notewriter/tree/main/examples) to have a better idea of what a note repository could look like.


## Ignore Markdown Files

You may want to exclude some files from being parsed by _The NoteWriter_. Two solutions exist:

* Use the special [tag](./tags.md) `ignore`:

```md title=ignore.md
---
tags: ignore
---

This file will be ignored when adding files using `nt add`.
```

* Exclude files by declaring glob patterns inside a file `.ntignore` present at the root of your repository:

``` title=.ntignore
/archives/**/*.md  # Any Markdown file under archives/
                   # will be ignored
```

:::tip

* **Use Markdown files as notebooks**. Group your notes inside a file like you would group them in a physical notebook.
* **Use _standard_ Markdown headings to group similar notes** under the same section.
* **Use _special_ Markdown headings to define the notes** that will be parsed by _The NoteWriter_.
* **Ignore files** that must not be processed.

:::


## Working with Files

After editing your Markdown files, you will use the command `nt add` to add them to _The NoteWriter_. This command parses your files and extracts different objects, covered in the following pages, and make them accessible in the desktop application.

```shell
$ echo "# My Notes" > nodes.md
$ nt add .
# The file is now visible in the desktop application
```

When using [remotes](./remotes) to push your notes (ex: to your phone), only committed objects will be pushed.

```shell
$ nt commit
# Objects can now be pushed
$ nt push phone # Expect a remote "phone" to have been declared
```

In practice, you will use the command `nt` along the command `git` and will add/commit your changes at the same time.

:::tip[Under the hood]
_The NoteWriter_ design was inspired by [Git internals](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects). When adding files with Git, the command `git add` traverses the working directory to find updates to extract a diff that is stored in the directory `.git/objects`.

Running `nt add` also traverses the working directory to find updated Markdown files to parse. For every file, _The Notewriter_ creates a new object:

```shell
$ echo "# My Notes" > nodes.md

$ git add .
$ tree .git/
.git
├── HEAD
├── config
├── index
├── objects
│   ├── fa
│   │   └── 47a5e460cc48457b1bfafacc2c78f533200e8d
│   ├── info
│   └── pack
└── refs
    ├── heads
    └── tags

$ nt add .
$ tree .nt/
.nt
├── config.jsonnet
├── database.db
├── index
├── nt.libsonnet
└── objects
    └── d2
        └── d2810d9388293838c8edac1654ecb2bb440f4237.pack
```

Both Git and _The NoteWriter_ needs to determine if a file has been edited. Both use a file `index` to determine the last known object ID (OID) and a timestamp when the file was last updated.

Extracted objects are stored into the directory `objects/`:

```shell
$ cat .nt/objects/d2/d2810d9388293838c8edac1654ecb2bb440f4237.pack
oid: d2810d9388293838c8edac1654ecb2bb440f4237
file_relative_path: notes.md
file_mtime: 2026-01-01T14:46:46.176291231+01:00
file_size: 11
ctime: 2026-01-01T14:48:00.339699+01:00
kind: objects
objects:
    - oid: 76913e4a281449efa5eade3702a0cda25434f3b6
      kind: file
      ctime: 2025-12-24T14:46:46.176291231+01:00
      desc: file "notes.md" [76913e4a281449efa5eade3702a0cda25434f3b6]
```

The design of _The NoteWriter_ was inspired by Git but not the implementation. Git has different requirements such as working with large mono-repositories with millions of files. Performance is not as important when processing hundreds of note files. For example, Git `index` is an optimized binary file when _The NoteWriter_ `index` is a YAML human-redable file easier to debug:

```shell
$ cat .nt/index
entries:
    - relative_path: notes.md
      kind: objects
      staged: true
      staged_packfile_oid: d2810d9388293838c8edac1654ecb2bb440f4237
      staged_mtime: 2026-01-01T14:46:46.176291231+01:00
      staged_itime: 2026-01-01T14:48:00.34468+01:00
      staged_size: 11
objects:
    - oid: 76913e4a281449efa5eade3702a0cda25434f3b6
      kind: file
      packfile_oid: d2810d9388293838c8edac1654ecb2bb440f4237
```

The file `index` contains also the staging area.

Unlike Git, _The NoteWriter_ stores objects into a second database, the SQLite file `.nt/database.db`, which is used extensively by _The NoteWriter Desktop_.

On this example, the file `notes.md` has been staged but still not committed:

```shell
$ nt commit
$ cat .nt/index
entries:
    - relative_path: notes.md
      kind: objects
      packfile_oid: d2810d9388293838c8edac1654ecb2bb440f4237
      mtime: 2026-01-01T14:46:46.176291231+01:00
      itime: 2026-01-01T14:48:00.34468+01:00
      size: 11
objects:
    - oid: 76913e4a281449efa5eade3702a0cda25434f3b6
      kind: file
      packfile_oid: d2810d9388293838c8edac1654ecb2bb440f4237
```

The index has been updated to mark all staged objects as committed and reasy to be pushed.
:::



:::note[Useful Commands]

* Use `nt add <path>` to add files or directories in the index. Notes inside these files will be processed and accessible in _The NoteWriter Desktop_.
* Use `nt commit` to commit staged files. Notes inside these files could then be pushed using `nt push`.
:::
