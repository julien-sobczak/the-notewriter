---
sidebar_position: 3
---

# `nt status`

## Name

`the-notewriter status` — Show the working tree status.

## Synopsis

```
Usage:
  nt status [flags]

Flags:
  -h, --help   help for status
```

## Description

Displays paths that have differences between the index file and the working tree.

## Examples

A basic example adding a new file:

```shell
$ edit hello.md
$ nt status
Changes to be committed:
  (use "nt restore..." to unstage)

Changes not staged for commit:
  (use "nt add <file>..." to update what will be committed)
	added:	hello.md

$ nt add hello.md
$ nt nt status
Changes to be committed:
  (use "nt restore..." to unstage)
	added:	file "hello.md" [01181370906c423f917bfa063e1fb15867357b22]
```

## See Also

* [`nt-add`](./nt-add.md) to add new changes reported by `nt status`
* [`nt-commit`](./nt-commit.md) to commit changes reported by `nt status`

