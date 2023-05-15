---
sidebar_position: 6
---

# `nt commit`

## Name

`the-notewriter commit` — Record changes to the repository.

## Synopsis

```
Usage:
  nt commit [flags]

Flags:
  -h, --help             help for commit
  -m, --message string   commit message
```

## Description

Create a new commit containing the current contents of the index and the given log message describing the changes. The new commit is a direct child of the last known commit.

The content to be committed can be specified in several ways:

* by using `nt add` to incrementally "add" changes to the index before using the `commit` command (Note: even modified files must be "added").

No other was are currently supported.

## Options

* `-m <msg>`, ` --message=<msg>`
  * Use the given `<msg>` as the commit message. No multiple `-m` are supported.

## Examples

When recording your own work, the contents of modified files in your working tree are temporarily stored to a staging area called the "index" with `nt add`. After building the state to be committed incrementally, `nt commit` is used to record what has been staged so far. This is the most basic form of the command. An example:

```shell
$ edit hello.md
$ nt add hello.md
$ nt commit -m "Add hello.md"
```

A commit doesn't have an author. Use Git to version your files and preserve the history.

## See Also

* [`nt-add`](./nt-add.md) to add new changes in staging area
