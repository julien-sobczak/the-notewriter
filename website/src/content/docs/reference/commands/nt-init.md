---
title: "nt init"
---

## Name

`nt init` — Create an empty _The NoteWriter_ repository.

## Synopsis

```
Usage:
  nt init [flags]

Flags:
  -h, --help   help for init
```

## Description

This command creates an empty _The NoteWriter_ repository — basically a `.nt` directory with a subdirectory for `objects`, and default configuration files. A default index will be created.

Running `nt init` in an existing repository is safe. It will not overwrite things that are already there.

## Example

* Init a new repository:

       $ nt init
       $ tree -a
       .
       ├── .nt
       │   ├── .gitignore
       │   └── config
       └── .ntignore

       1 directory, 3 files

## See Also

* [`nt-add`](nt-add.md) to add your first objects
