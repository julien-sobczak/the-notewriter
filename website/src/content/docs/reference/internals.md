---
title: Internals
---

:::note

Examples are sometimes edited to keep them concise. For example, most OIDs will be presented in a short form instead of their 40-character format.

:::


_The NoteWriter_ is fundamentally a CLI to extract objects from Markdown files.

## _The NoteWriter_ Objects

When running commands like `nt add`, _The NoteWriter_ parses Mardown files to extract different objects.

### `file`

Each parsed file generates an object `file` representing the file as a YAML document.

Ex (Mardown):

```md
---
tags:
- go
---

# Go

## Reference: Golang History

`#history`

`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike,
and Ken Thompson at Google in 2007.


## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./medias/go.png)


## TODO: Conferences

* [Gophercon Europe](https://gophercon.eu/) `#reminder-2023-06-26`
```

Ex (Internal):

```yaml
oid: 6347cbdd           # A uniquely generated OID
relative_path: go.md    # Relative file path from the root of the repository
wikilink: go            # Long wikilink (ex: [[go]] is a valid link to this file)
front_matter: null      # The front matter in YAML
attributes:             # The file attributes (including optional tags)
    tags:
        - go
body: |-                # The raw body (without the Front Matter)
    # Go
    ...
body_line: 6            # The line number of the first body line
size: 463               # The raw file sizze
hash: 1eba21ae87635b6b9a76ca4df89bf2931da64d42 # A SHA-1 using file content as source
mtime: 2023-01-01T12:00:00                     # The last modified time on FS
created_at: 2023-01-01T12:00:00                # The file object creation time
updated_at: 2023-01-01T12:00:00                # The file object modification time
```


### `note`

Each [note](../guides/notes.md), independent of its kind, generates an object `note` representing the note as a YAML document.

Ex (Markdown):

```md
## Reference: Golang History

`#history`

`@source: https://en.wikipedia.org/wiki/Go_(programming_language)`

[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike,
and Ken Thompson at Google in 2007.
```

Ex (Internal):

```yaml
oid: d790d08c                             # Unique generated OID
file_oid: 6347cbdd                        # OID of the file containing this note
parent_note_oid: ""                       # OID of the parent heading when nested under a note
kind: reference                           # Kind of the object
title: 'Reference: Golang History'        # Raw Title without the heading character(s) "#"
short_title: Golang History               # Title without the kind prefix
long_title: Golang History                # Concatenation with all optional parent short titles
relative_path: go.md                      # Relative path of the file containingg this note
wikilink: 'go#Reference: Golang History'  # Long wikilink (ex: [[go#Reference: Golang History]] is a valid link)
attributes:                               # Attributes (including inherited ones)
    source: https://en.wikipedia.org/wiki/Go_(programming_language)
    tags:
        - go
        - history
    title: Golang History
tags:                                     # Tags (= special attribute named "tags")
    - go
    - history
line: 8                                   # Line number of the first line inside the file
content: |-                               # Processed content
    `#history`

    `@source: https://en.wikipedia.org/wiki/Go_(programming_language)`

    [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
content_hash: 0eba86c8b008c0222869ef5358d48ab8241ffc8e # SHA-1 of the property content_raw
created_at: 2023-01-01T12:00:00      # Object creation time
updated_at: 2023-01-01T12:00:00      # Object modification time
```

### `flashcard`

Each [note](../guides/notes.md) of kind `flashcard` generates an additional object `flashcard` representing the flashcard to learn as a YAML document.

Ex (Markdown):

```md
## Flashcard: Golang Logo

What does the **Golang logo** represent?

---

A **gopher**.

![Logo](./medias/go.png)
```

Ex (Internal):

```yaml
oid: 3c268dd8             # Unique generated OID
short_title: Golang Logo  # Short title of the note
file_oid: 6347cbdd        # File OID containing the note
note_oid: 3513929a        # Note OID associated with this flashcard
relative_path: go.md      # Relative path to the file
tags: [go]                # Tags (= special attributes "tags")
interval: 1               # Various SRS settings
...
front: |-                 # Front card in Markdown
    What does the **Golang logo** represent?
back: |-                  # Back card in Markdown
    A **gopher**.

    ![Logo](oid:4a4faba3)
created_at: 2023-01-01T12:00:00  # Object creation time
updated_at: 2023-01-01T12:00:00  # Object modification time
```

### `media`

Each link inside `note` referencing a local file generates an object `media` representing the media file as a YAML document.

Ex (Markdown):

```md
![Logo](./medias/go.png)
```

Ex (Internal):

```yaml
oid: 840dd3bc                    # Unique generated OID
relative_path: medias/go.png     # Relative path to root directory of the repository
kind: picture                    # Object kind
dangling: false                  # True when file is not present on disk
extension: .png                  # File extension
mtime: 2023-01-01T12:00:00       # File last modification time on FS
hash: ef81045f57ea747457769965487ac8211a44ed32 # SHA-1 using file content as source
size: 14220                      # File size in bytes
blobs:                           # List of blobs (= optimized versions)
    - oid: 6545e323              # Unique OID using file content
      mime: image/avif           # Mime type
      attributes: {}             # Optional attributes
      tags:                      # Identify the blob type
        - preview                # (preview = thumbnail)
        - lossy                  # (lossy = lossy conversion)
    - oid: eb49431b
      mime: image/avif
      attributes: {}
      tags:
        - original
        - lossy
created_at: 2023-01-01T12:00:00  # Object creation time
updated_at: 2023-01-01T12:00:00  # Object modification time
```

### `link`

Each link (internal or external) that includes a Go link inside the link's title generates an additional object `link` representing the special link as a YAML document.

Ex (Markdown):

```md
[Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
```

Ex (Internal):

```yaml
oid: 8c9d3ed                      # Unique generated OID
note_oid: d790d08c                # Note OID containing the link definition
relative_path: go.md              # Relative path to the file containing the link
text: Golang                      # Link Text
url: https://go.dev/doc/          # Link target URL
title: ""                         # Link title without the special syntax for the Go link
go_name: go                       # Name of the Go link
created_at: 2023-01-01T12:00:00   # Object creation time
updated_at: 2023-01-01T12:00:00   # Object modification time
```

### `reminder`

Reminders are defined using a special tag syntax and generate additional `reminder` object representing the reminders as YAML document:

Ex (Markdown):

```md
* [Gophercon Europe](https://gophercon.eu/) `#reminder-2023-06-26`
```

Ex (Internal):

```yaml
oid: 9032a26e              # Unique generated OID
file_oid: 6347cbdd         # File OID containing this reminder
note_oid: 9d5ac892         # Note OID containing this reminder
relative_path: go.md       # Relative path to the file containing this reminder
description: |-            # The description of reminder
    '[Gophercon Europe](https://gophercon.eu/)'
tag: '#reminder-2023-06-26'               # The tag
last_performed_at: 0001-01-01T00:00:00Z   # Last time when the reminder what planned (for recurring reminder)
next_performed_at: 2023-06-26T00:00:00Z   # Next time when the reminder is planned (for future reminder)
created_at: 2023-01-01T12:00:00           # Object creation time
updated_at: 2023-01-01T12:00:00           # Object modification time
```


## _The NoteWriter_ Database

:::tip

_The NoteWriter_ internal database (like `nt` commands) is largely inspired by Git. If you have already looked at [Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain), the file names and their organization on disk are very similar. Here are the main differences:

* The content of index files is mostly YAML files (easier to debug but less performant).
* There are no branching, no versioning. _The NoteWriter_ is designed to be used along Git.
* Objects are persisted in pack files immediately to limit the number of files on disk (Git defers the creation of packfiles in a maintenance task).

:::

Above objects are stored (indirectly) in the index. Ex:

```shell
.
├── .nt
│   ├── .gitignore
│   ├── config
│   ├── database.db
│   ├── index
│   └── objects
│       ├── 4a
│       │   └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6.pack
│       ├── af
│       │   └── afe988e5f40e4d1181a86f522e3c1f2e6f0241e3.pack
│       ├── db
│       │   └── dbeaba7026ce4be8aa84ee85992dc9eca31118f7.blob
│       └── e7
│           └── e7b26e89367a22b5f578532a74742e646f843e1f.blob
│
├── .ntignore
├── hello.md
└── me.png
```

This section documents the format of the different files that composed the internal database.

### `.nt/objects/`

This directory contains two kinds of objects:

* Pack Files (`*.pack`): A group of objects present in a single file inside the repository (a Markdown file, a media file).
* Blobs (`*.blob`): The raw bytes for a single media file, the metadata are stored in the media object referencing the blob inside a commit object.

All objects and all blobs are uniquely identified by their OID, a 40-character string similar to the SHA-1 used by Git. The OIDs for pack files and blobs are used to spread the files into subdirectories and avoid having thousands of files directly under `.nt/objects`. For example, the pack file `afe988e5f40e4d1181a86f522e3c1f2e6f0241e3` and the blob `6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4` will be stored like this:

```
.nt
└── objects
    ├── 6e
    │   └── 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4.blob
    └── af
        └── afe988e5f40e4d1181a86f522e3c1f2e6f0241e3.pack
```


#### `.nt/objects/{xx}/{packfile-sha1}`

Pack files are YAML objects.

Ex:

```yaml
oid: 4a03d1ab               # A unique OID for this pack file
file_relative_path: go.md         # Relative path of the source file
file_mtime: 2023-01-01T12:30:00Z  # Mtime of the source file
file_size: 134                    # Size of the source file
ctime: 2023-01-01T12:00:00  # The creation time
objects:                    # The list of all objects added/modified/deleted (a new pack file cannot contain more than a predefined number of objects)
    - oid: d19a2bba                      # The object OID
      kind: file                         # The object kind
      ctime: 2023-01-01T12:00:00         # The creation time of the object
      desc: file "hello.md" [d19a2bba]   # A human-readable description of the object
      data: <value>                      # A base64-encoded representation of the object
    - oid: 6ee8a962
      kind: note
      Ctime: 2023-01-01T12:00:00
      desc: 'note "Reference: Hello" [6ee8a962]'
      data: <value>
```

Each object is self-containing through the `data` attribute. The value is compressed using zlib and encoded in Base 64. You can easily retrieve the uncompressed content:

```shell
# On MacOS
$ brew install qpdf
$ echo "<value>" | base64 -d | zlib-flate -uncompress
oid: 6ee8a962
file_oid: d19a2bba
kind: reference
relative_path: hello.md
wikilink: 'hello#Reference: Hello'
content: Coucou
content_hash: b70f7d0e2acef2e0fa1c6f117e3c11e0d7082232
...
```

The main motivation behind pack files is to limit the number of files on disk (and the number of files to transfer when using a [remote repository](../guides/remote.md)).


### `.nt/index`

The `index` file serves multiple purposes. It contains the staging area (= the list of pack files to include in the next commit) and keeps a list of all pack files to quickly locate an object or a blob.

Ex:

```yaml
committed_at: 2023-01-01T12:00:00   # The last committed timestamp
entries:
  - relative_path: "go.md"          # Relative path of the source file
    packfile_oid: a3455b            # Committed pack file containing
                                    # the last known version of this source file
    mtime: 2022-12-11T02:14:00      # Mtime and size of the source file...
    size: 123                       # ... when the file was committed
  - relative_path: "python.md"
    packfile_oid: 00000             # Never committed
    mtime: 0001-01-01T00:00:00
    size: 0
    # For staged entries

	  staged: true                           # True when a file has been staged
  	staged_packfile_oid: 837291            # Pack file containing the staged version
	  staged_mtime: 2023-01-01T08:42:00      # Mtime and size of the source file...
	  staged_size: 2349                      # ... when the file was staged
	  staged_tombstone: 0001-01-01T00:00:00  # Tombstone for staged deleted files
  ...
objects:
	- oid: 983712
	  kind: note
	  packfile_oid: a3455b
  ...
blobs:
	- oid: 98ab19
	  mime: audio/mpeg
	  packfile_oid: 837291
  ...
```


### `.nt/database.db`

In addition to raw files, _The NoteWriter_ also comprises a SQLite database (populated using the same information as present in object files, included staged ones). This database is used to speed up commands but also to benefit from the [full-text search support](https://www.sqlite.org/fts5.html) when using the desktop application.


## Example

1. Setup a new repository

    ```shell
    $ mkdir notes
    $ cd notes
    $ nt init
    ```

    Inspect the database:

    ```shell
    .
    ├── .nt
    │   ├── .gitignore
    │   └── config
    └── .ntignore
    ```

    Most files are still missing and will be populated only after adding files.


2. Add a new note

    ```shell
    $ nt init
    $ echo "# Reference: Hello\n\nCoucou" > hello.md
    $ nt add hello.md && nt commit
    [bf712c5de01642338ce2d16a37daabeb37daabeb]
     2 objects changes, 2 insertion(s)
     create file "hello.md" [d19a2bba42d44d8a82b18b2edcd4320612a3dfbc]
     create note "Reference: Hello" [6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4]
    ```

    Inspect the database:

    ```shell
    $ tree -a
    .
    ├── .nt
    │   ├── .gitignore
    │   ├── config
    │   ├── database.db
    │   ├── index
    │   └── objects
    │       ├── 4a
    │       │   └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6.pack
    │       └── 67
    │           └── 67937d98e9cba4df937d41348e6d4eec5d11546c.blob
    ├── .ntignore
    └── hello.md
    ```

    Database files have now been created. We have new objects under `objects` representing the unique pack file (`4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6`) containing two objects:

    ```shell
    $ nt cat-file
    oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    file_relative_path: hello.md
    file_mtime: 2023-01-01T11:30:00Z
    file_size: 134
    ctime: 2023-01-01T12:00:00
    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          mtime: 2023-01-01T12:00:00
          desc: file "hello.md" [d19a2bba42d44d8a82b18b2edcd4320612a3dfbc]
          data: aBc...=
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          mtime: 2023-01-01T12:00:00
          desc: 'note "Reference: Hello" [6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4]'
          data: eFg...=
    ```

    And our index has been updated with new entries that are no longer staged:

    ```shell
    $ cat .nt/index
    committed_at: 2023-01-01T12:00:00
    entries:
        - relative_path: hello.md
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
          mtime: 2023-01-01T12:00:00
          size: 1234
    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    blobs:
        - oid: 67937d98e9cba4df937d41348e6d4eec5d11546c
          mime: text/markdown
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    ```

3. Edit the note to reference a new media

    ```shell
    $ cp ~/me.png .
    $ echo "\n"'![](me.png)' >> hello.md
    $ nt add
    ```

    Check that the staging area is not empty:

    ```shell
    $ cat .nt/index
    committed_at: 2023-01-01T12:00:00
    entries:
        - relative_path: hello.md
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
          mtime: 2023-01-01T12:00:00
          size: 1234
          staged: true
          staged_packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
          staged_mtime: 2023-01-02T12:00:00
          staged_size: 1245
        - relative_path: me.png
          packfile_oid: 0000000000000000000000000000000000000000
          mtime: 0001-01-01T00:00:00
          size: 0
          staged: true
          staged_packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
          staged_mtime: 2023-01-02T12:00:00
          staged_size: 1245

    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: 634a860212ee4ad392dd47a375aa88431a494f1c
          kind: media
          packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
    blobs:
        - oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
          mime: text/markdown
          packfile_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
        - oid: 0733657a2ca447beadfbce2832a26035deb1634e
          mime: text/markdown
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: a977d1c4ed76444e9ff05c37f2ce2c3be0fa7b55
          mime: image/avif
          packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
    ```

4. Commit changes

    ```shell
    $ nt commit
    [c34575fba9884d62b5512e2c5fbc274c5fbc274c]
     3 objects changes, 1 insertion(s), 2 modification(s)
     create media me.png [3837a10fbc3a47c7961896febf64463b4a006c79]
     modify file "hello.md" [d19a2bba42d44d8a82b18b2edcd4320612a3dfbc]
     modify note "Reference: Hello" [6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4]

    $ tree -a
    .
    ├── .nt
    │   ├── .gitignore
    │   ├── config
    │   ├── database.db
    │   ├── index
    │   └── objects
    │       ├── 06
    │       │   └── 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1.pack
    │       ├── 07
    │       │   └── 0733657a2ca447beadfbce2832a26035deb1634e.pack
    │       └── 4a
    │           └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6.pack
    ├── .ntignore
    ├── hello.md
    └── me.png
    ```

    The commit has been recorded:

    ```shell
    $ cat .nt/objects/index
    committed_at: 2023-01-02T12:00:00
    entries:
        - relative_path: hello.md
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
          mtime: 2023-01-02T12:00:00
          size: 1245
        - relative_path: me.png
          packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
          mtime: 2023-01-02T12:00:00
          size: 1245

    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: 634a860212ee4ad392dd47a375aa88431a494f1c
          kind: media
          packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
    blobs:
        - oid: 0733657a2ca447beadfbce2832a26035deb1634e
          mime: text/markdown
          packfile_oid: 0733657a2ca447beadfbce2832a26035deb1634e
        - oid: a977d1c4ed76444e9ff05c37f2ce2c3be0fa7b55
          mime: image/avif
          packfile_oid: 06400bd13f8a43c8a5b5f3db41f60dac7bfa78f1
    ```

That's all. You have seen the various files in action.
