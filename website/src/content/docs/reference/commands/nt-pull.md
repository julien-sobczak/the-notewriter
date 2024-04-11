---
title: "nt pull"
---

## Name

`the-notewriter pull` - Fetch recent commits and objects from a remote ref to a local ref.

## Synopsis

```
Usage:
  nt pull [flags]

Flags:
```

## Description

Incorporates changes from a remote into the current repository. If commits are missing locally, there will be applied.

No conflicts can occurs when pulling changes. The `.nt/index` file will be merged to incorporate misssing and new commits and all missing objects will be downloaded.

## Configuration

See [`nt-push`](./nt-push) for "Configuration.

## Examples

* Pull missing commits from the remote ref:

        $ nt pull

## See Also

* [`nt-commit`](./nt-commit.md) to create a new commit from changes in staging area
* [`nt-push`](./nt-push.md) to push recent commits to a remote ref

