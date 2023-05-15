---
sidebar_position: 1
---

# Internals

_The NoteWriter_ is fundamentally a CLI to extract objects from files.

## _The NoteWriter_ Objects

When running commands like `nt add`, _The NoteWriter_ parses Mardown files to extract different objects.

### `note`

Each [note](../guides/notes.md), independent of its kind, generates an object `note` representing the note as a YAML document.

Ex:

```md

```


## _The NoteWriter_ Database

Above objects are stored in the index. Ex:

```shell
**TODO** use real-world
```

This section documents the format of the different files that composed the internal database.

### `.nt/objects`

This directory contains two kinds of objects:

* Commits (a kind of "packfile" regrouping all updated objects in a single commit).
* Blobs (the raw bytes for a media file, the metadata are stored in the commit file introducing the blob).

All objects (leafs such as `note` or composite such as `commit`  commits) and all blobs are uniquely identified by their OID, a 40-character string similar to the SHA-1 used by Git. This OID is used to spread the files into subdirectories and avoid having thousands of files directly under `.nt/objects`.



### `.nt/index`












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
    [0d080602a03e4f6d864dde703110bfc6e9bc54f8]
     2 objects changes, 2 insertion(s)
     create file "hello.md" [8b8d6d820e6f40a88c9df429ec25f4f1c284aaf5]
     create note "Reference: Hello" [7def57a97fa0459a881988191e3c1a59532d4b7d]
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
    │   │   ├── 0d
    │   │   │   └── 0d080602a03e4f6d864dde703110bfc6e9bc54f8
    │   │   └── info
    │   │       └── commit-graph
    │   └── refs
    │       └── main
    ├── .ntignore
    └── hello.md
    ```

    Database files have now been created. We have a new object under `objects` representing our commit (`0d080602a03e4f6d864dde703110bfc6e9bc54f8`) and containing the objects:

    ```shell
    ```




