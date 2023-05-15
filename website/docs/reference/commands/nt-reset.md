---
sidebar_position: 5
---

# `nt reset`

## Name

`the-notewriter reset` — Reset staging area.

## Synopsis

```
Usage:
  nt reset [flags]

Flags:
  -h, --help   help for reset
```

## Description

This command resets the current staging area to cancel all previous `nt add` commands from the last commit.

## Examples

A basic example if to revert a previously added file still not committed:

```shell
$ edit hello.md
$ nt add hello.md
$ nt status
Changes to be committed:
  (use "nt restore..." to unstage)
	added:	file "hello.md" [60409b7bd01d49509bbffe6adba1e9916eb31c06]
$ nt reset
Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
	added:	hello.md
```

## See Also

* [`nt-add`](./nt-add.md) to add new changes in staging area
