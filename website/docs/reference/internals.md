---
sidebar_position: 1
---

# Internals

:::info

Examples are sometimes edited to keep them concise. For example, most OIDs will be presented in a short form instead of their 40-character format.

:::


_The NoteWriter_ is fundamentally a CLI to extract objects from files.

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
mode: 420               # The UNIX file mode
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
content_raw: |-                           # Content as present in file
    `#history`

    `@source: https://en.wikipedia.org/wiki/Go_(programming_language)`

    [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
content_hash: 0eba86c8b008c0222869ef5358d48ab8241ffc8e # SHA-1 of the property content_raw
title_markdown: '# Golang History'   # Short title in Markdown
title_html: <h1>Golang History</h1>  # Short title in HTML
title_text: |-                       # Short title in plain text
    Golang History
    ==============
content_markdown: |-                 # Processed content in Markdown
    [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.'
content_html: |-                     # Processed content in HTML
    <p><a href="https://go.dev/doc/" title="#go/go">Golang</a> was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.</p>
content_text: |-                     # Processed content in plain text
    Golang was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
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
front_markdown: |-        # Front card in Markdown
    What does the **Golang logo** represent?
back_markdown: |-         # Back card in Markdown
    A **gopher**.

    ![Logo](oid:4a4faba3)
front_html: |-            # Front card in HTML
    <p>What does the <strong>Golang logo</strong> represent?</p>
back_html: |-             # Back card in HTML
    <p>A <strong>gopher</strong>.</p>

    <p><img src="oid:4a4faba3" alt="Logo" /></p>
front_text: |-            # Front card in plain text
    What does the Golang logo represent?
back_text: |-             # Back card in HTML
    A gopher.

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
mode: 420                        # UNIX file modes
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
description_raw: |-        # The raw description of reminder
    '[Gophercon Europe](https://gophercon.eu/)'
description_markdown: |-   # The processed description in Markdown
    '[Gophercon Europe](https://gophercon.eu/)'
description_html: |-       # The processed description in HTML
    <p><a href="https://gophercon.eu/">Gophercon Europe</a></p>
description_text: |-       # The processed description in plain text
    Gophercon Europe
tag: '#reminder-2023-06-26'               # The tag
last_performed_at: 0001-01-01T00:00:00Z   # Last time when the reminder what planned (for recurring reminder)
next_performed_at: 2023-06-26T00:00:00Z   # Next time when the reminder is planned (for future reminder)
created_at: 2023-01-01T12:00:00           # Object creation time
updated_at: 2023-01-01T12:00:00           # Object modification time
```


## _The NoteWriter_ Database

:::tip

_The NoteWriter_ internal database (like commands) is largely inspired by Git. If you have already looked at [Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain), the file names and their organization on disk is very similar. Here are major differences:

* The content of index files is mostly YAML files (easier to debug).
* There are no branching, no versioning. _The NoteWriter_ is designed to be used with Git.
* There are no equivalent to Git packfiles (composite file packaging previously-created Git objects to improve performance). _The NoteWriter_ packages objects inside `commit` files for similar reasons but the implementation differs as the packaging is done immediately.

:::

Above objects are stored (indirectly) in the index. Ex:

```shell
.
├── .nt
│   ├── .gitignore
│   ├── config
│   ├── database.db
│   ├── index
│   ├── objects
│   │   ├── 4a
│   │   │   └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
│   │   ├── af
│   │   │   └── afe988e5f40e4d1181a86f522e3c1f2e6f0241e3
│   │   ├── db
│   │   │   └── dbeaba7026ce4be8aa84ee85992dc9eca31118f7
│   │   ├── e7
│   │   │   └── e7b26e89367a22b5f578532a74742e646f843e1f
│   │   └── info
│   │       └── commit-graph
│   └── refs
│       └── main
├── .ntignore
├── hello.md
└── me.png
```

This section documents the format of the different files that composed the internal database.

### `.nt/objects/`

This directory contains two kinds of objects:

* Commits (a kind of "packfile" regrouping all updated objects in a single commit).
* Blobs (the raw bytes for a single media file, the metadata are stored in the media object referencing the blob inside a commit object).

All objects (leafs such as `note` or composite such as `commit`  commits) and all blobs are uniquely identified by their OID, a 40-character string similar to the SHA-1 used by Git. This OID is used to spread the files into subdirectories and avoid having thousands of files directly under `.nt/objects`. For example, the commit `afe988e5f40e4d1181a86f522e3c1f2e6f0241e3` and the blob `6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4` will be stored like this:

```
.nt
└── objects
    ├── 6e
    │   └── 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
    └── af
        └── afe988e5f40e4d1181a86f522e3c1f2e6f0241e3
```

The directory `objects` contains an additional file `.nt/objects/info/commit-graph` containing the linear sequence of commits applied in this repository.

#### `.nt/objects/{xx}/{commit-sha1}`

Commits are YAML objects.

Ex:

```yaml
oid: 4a03d1ab               # A unique OID for this commit (as present in commit-graph)
ctime: 2023-01-01T12:00:00  # The commit creation time
objects:                    # The list of all objects added/modified/deleted by this commit
    - oid: d19a2bba                      # The object OID
      kind: file                         # The object kind
      state: added                       # The action introduced by this commit
      mtime: 2023-01-01T12:00:00         # The modification time of the action
      desc: file "hello.md" [d19a2bba]   # A human-readable description of the object
      data: <value>                      # A base64-encoded representation of the object
    - oid: 6ee8a962
      kind: note
      state: added
      mtime: 2023-01-01T12:00:00
      desc: 'note "Reference: Hello" [6ee8a962]'
      data: <value>
```

Each object is self-containing through the `data` attribute and compressed using zlib and encoded in Base 64. You can easily retrieve the uncompressed content:

```shell
# On MacOS
$ brew install qpdf
$ echo "<value>" | base64 -d | zlib-flate -uncompress
oid: 6ee8a962
file_oid: d19a2bba
kind: reference
relative_path: hello.md
wikilink: 'hello#Reference: Hello'
content_raw: Coucou
content_hash: b70f7d0e2acef2e0fa1c6f117e3c11e0d7082232
...
```

The main motivation behind `commit` objects is to limit the number of files on disk (and the number of files to transfer when using a [remote repository](../guides/remote.md)).

#### `.nt/objects/info/commit-graph`

The `commit-graph` file lists in a sequential order all commits that was processed in this repository. The list is useful when retrieving new objects from a [remote](../guides/remote.md) to quickly determine the missing commits to replay.

Ex:

```yaml
updated_at: 2023-01-01T12:00:00  # Date of the last applied commit
commits:                         # List of commits
    - 4a03d1ab                   #     - Older commit
    - dbeaba70                   #     - ...
    - afe988e5                   #     - Last commit
```

### `.nt/index`

The `index` file serves multiple purposes. It contains the staging area (= the list of objects to include in the next commit) and keeps a list of all objects to quickly locate the commit containing them.

Ex:

```yaml
objects:                                      # List of all known objects in repository
                                              # (do not include commit objects)
    - oid: d19a2bba                           # Object OID
      kind: file                              # Object Kind
      mtime: 2023-01-01T12:00:00              # Object modification time
      commit_oid: 4a03d1ab                    # Commit OID containing the last version
    - oid: 6ee8a962
      kind: note
      mtime: 2023-01-01T12:00:00
      commit_oid: 4a03d1ab
    - oid: 3837a10f
      kind: media
      mtime: 2023-01-01T12:00:00
      commit_oid: dbeaba70
orphan_blobs: []
staging: # The Staging Area (= nt add)
    # NB: Objects in staging area uses
    # the same format as object in commit files
    # (make easy to create new commit files)
    - commitobject:
        oid: d19a2bba42
        kind: file
        state: modified
        mtime: 2023-01-01T12:00:00
        desc: file "hello.md" [d19a2bba]
        data: <value>
        previous_commit_oid: afe988e5
    - commitobject:
        oid: 6ee8a962
        kind: note
        state: modified
        mtime: 2023-01-01T12:00:00
        desc: 'note "Reference: Hello" [6ee8a962]'
        data: <value>
        previous_commit_oid: afe988e5
```

### `.nt/refs/`

In the way that Git Branches are simply aliases to commit SHA-1, references contains the a commit OID.

* `.nt/refs/main` is the reference for the local repository (= the last commit OID present in `commit-graph`)
* `.nt/refs/remote` is only present when using a [remote](../guides/remote.md). It contains the last known commit when the remote was last checked.

### `.nt/database.db`

In addition to raw files, _The NoteWriter_ also comprises a SQLite database (populated using the same information as present in object files). This database is used to speed up commands but also to benefit from the [full-text search support](https://www.sqlite.org/fts5.html) when using the desktop application.


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
    [4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6]
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
    │   ├── objects
    │   │   ├── 4a
    │   │   │   └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    │   │   └── info
    │   │       └── commit-graph
    │   └── refs
    │       └── main
    ├── .ntignore
    └── hello.md
    ```

    Database files have now been created. We have a new object under `objects` representing our commit (`4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6`) and containing the objects:

    ```shell
    $ nt cat-file
    oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    ctime: 2023-01-01T12:00:00
    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          state: added
          mtime: 2023-01-01T12:00:00
          desc: file "hello.md" [d19a2bba42d44d8a82b18b2edcd4320612a3dfbc]
          data: aBc...=
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          state: added
          mtime: 2023-01-01T12:00:00
          desc: 'note "Reference: Hello" [6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4]'
          data: eFg==
    ```

    The commit has been recording into the `commit-graph`:

    ```shell
    $ cat .nt/objects/info/commit-graph
    updated_at: 2023-01-01T12:00:00
    commits:
        - 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    ```

    The reference `main` points to this commit:

    ```shell
    $ cat .nt/refs/main
    4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    ```

    And our index has been updated to clear the staging area:

    ```shell
    $ cat .nt/index
    objects:
        - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
          kind: file
          mtime: 2023-01-01T12:00:00
          commit_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
        - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
          kind: note
          mtime: 2023-01-01T12:00:00
          commit_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    orphan_blobs: []
    staging: []
    ```

3. Edit the note to reference a new media

    ```shell
    $ cp ~/me.png .
    $ echo "\n"'![](me.png)' >> hello.md
    $ nt add
    ```

    Check the staging area is not empty:

    ```shell
    $ cat .nt/index
    objects:
    - oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
      kind: file
      mtime: 2023-01-01T12:00:00
      commit_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    - oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
      kind: note
      mtime: 2023-01-01T12:00:00
      commit_oid: 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    orphan_blobs: []
    staging:
        - commitobject:
            oid: 3837a10fbc3a47c7961896febf64463b4a006c79
            kind: media
            state: added
            mtime: 2023-01-01T12:00:00
            desc: media me.png [3837a10fbc3a47c7961896febf64463b4a006c79]
            data: gHi...=
            previous_commit_oid: ""
        - commitobject:
            oid: d19a2bba42d44d8a82b18b2edcd4320612a3dfbc
            kind: file
            state: modified
            mtime: 2023-01-01T12:00:00
            desc: file "hello.md" [d19a2bba42d44d8a82b18b2edcd4320612a3dfbc]
            data: AbC...=
            previous_commit_oid: "4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6"
        - commitobject:
            oid: 6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4
            kind: note
            state: modified
            mtime: 2023-01-01T12:00:00
            desc: 'note "Reference: Hello" [6ee8a9620d3f4d3f9fbd159744ef85b83400b0d4]'
            data: DeF...=
            previous_commit_oid: "4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6"
        ```

4. Commit changes

    ```shell
    $ nt commit
    [dbeaba7026ce4be8aa84ee85992dc9eca31118f7]
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
    │   ├── objects
    │   │   ├── 4a
    │   │   │   └── 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
    │   │   ├── db
    │   │   │   └── dbeaba7026ce4be8aa84ee85992dc9eca31118f7
    │   │   ├── e7
    │   │   │   └── e7b26e89367a22b5f578532a74742e646f843e1f
    │   │   └── info
    │   │       └── commit-graph
    │   └── refs
    │       └── main
    ├── .ntignore
    └── hello.md
    ```

    The commit has been recorded:

    ```shell
    $ cat .nt/objects/info/commit-graph
    updated_at: 2023-01-01T12:00:00
    commits:
        - 4a03d1ab3dbe4c5d9efacd0e05e187179c5415c6
        - dbeaba7026ce4be8aa84ee85992dc9eca31118f7
    $ cat .nt/refs/main
    dbeaba7026ce4be8aa84ee85992dc9eca31118f7
    ```

That's all. You have seen the various files in action.


