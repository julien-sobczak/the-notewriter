---
title: "nt pull"
---

## Name

`nt pull` - Fetch recent missing packfiles and blobs from a remote locally.

## Synopsis

```
Usage:
  nt pull [flags]

Flags:
```

## Description

Incorporates changes from a remote into the current repository. If objects or blobs are missing locally, there will be retrieved.

No conflicts can occur when pulling changes. The `.nt/index` file will be merged to incorporate changes.

## Configuration

See [`nt-push`](./nt-push) for "Configuration".

## Examples

* Pull missing commits from the remote ref:

        $ nt pull

## See Also

* [`nt-commit`](./nt-commit.md) to create a new commit from changes in staging area
* [`nt-push`](./nt-push.md) to push recent commits to a remote ref

