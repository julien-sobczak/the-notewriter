---
title: "nt push"
---

## Name

`nt push` — Update the remote ref along with associated objects.

## Synopsis

```
Usage:
  nt push [flags]

Flags:
  -h, --help   help for push
```

## Description

Updates the remote using the local index, sending missing objects in remote index.

If the remote contains objects not present in the local ref, the command is aborted. Run `nt pull` first.

## Configuration

Remotes are declared inside the `.nt/config` file. Several remote implementations are supported:

* `file`
* `s3`

**TODO** complete

## Examples

* Push all commits not present in the remote ref:

        $ nt push

## See Also

* [`nt-commit`](./nt-commit.md) to create a new commit from changes in staging area
* [`nt-pull`](./nt-pull.md) to download recent changes in the local ref

