---
title: "nt add"
---

## Name

`nt add` — Add files to the index.

## Synopsis

```
Usage:
  nt add [flags] [--] [<pathspec>…​]

Flags:
  -h, --help   help for add
```

## Description

This command updates the index (`.nt/index`) and the database (`.nt/objects`) to create new pack files and append them in the staging area, to prepare the content staged for the next commit.

The "index" holds a snapshot of the content of the working tree, and it is this snapshot that is taken as the content of the next commit. Thus after making any changes to the working tree, and before running the `commit` command, you must use the `add` command to add any new or modified files to the index.

This command can be performed multiple times before a commit. It only adds the content of the specified file(s) at the time the add command is run; if you want subsequent changes included in the next commit, then you must run `nt add` again to add the new content to the index.

The `nt status` command can be used to obtain a summary of which objects have changes that are staged for the next commit.

The `nt add` command will not add ignored files by default (based on `.ntignore` file).

The `nt add` command will refuse to add files that violate lint rules. Violations are printed when this occurs.

## Options

* `<pathspec>`...
  * Files to add content from. Fileglobs (e.g. `*.c`) can be given to add all matching files. Also a leading directory name (e.g. `dir` to add `dir/file1` and `dir/file2`) can be given to update the index to match the current state of the directory as a whole (e.g. specifying `dir` will record not just a file `dir/file1` modified in the working tree, a file `dir/file2` added to the working tree, but also a file `dir/file3` removed from the working tree).


## Examples

* Add all changes:

        $ nt add .


* Add contents under `projects/secret` directory and its subdirectories:

        $ nt add projects/secret

## See Also

* [`nt-lint`](./nt-lint.md) to list all violations based on linter rules
* [`nt-status`](./nt-status.md) to list pending changes in staging area
* [`nt-commit`](./nt-commit.md) to create a new commit from changes in staging area
* [`nt-restore`](nt-reset.md) to revert some changes in staging area
* [`nt-diff`](./nt-diff.md) to show changes in staging area
