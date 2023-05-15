---
sidebar_position: 4
---

# `nt diff`

## Name

`the-notewriter diff` — Show changes between the last commit, the index, and the working tree.

## Synopsis

```
Usage:
  nt diff [flags]

Flags:
      --cached   Show staged changes
  -h, --help     help for diff
      --staged   Show staged changes
```

## Description

Show changes between the working tree and the index or changes between the index and the last commit.

* `nt diff`
  * This form is to view the changes you made relative to the index (staging area for the next commit). In other words, the differences are what you **could** tell _The NoteWriter_ to further add to the index but you still haven't. You can stage these changes by using [`nt-add`](./nt-add.md).

* `nt diff --staged`, `nt diff --cached`
  * This form is to view the changes you staged for the next commit relative to the last commit. `--staged` is a synonym of `--cached`. In other words, the differences you have already added using [`nt-add`](./nt-add.md).

## Examples

* Show changes in the working tree not yet staged for the next commit.

        $ nt diff

* Show changes between the index and your last commit; what you would be committing if you run `nt commit`.

        $ nt diff --staged


## See Also

* [`nt-add`](./nt-add.md) to add new files in staging area
* [`nt-commit`](./nt-commit.md) to create a new commit from changes in staging area

