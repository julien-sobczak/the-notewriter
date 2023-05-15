---
sidebar_position: 9
---

# `nt gc`

## Name

```the-notewriter gc` - Cleanup unnecessary files and optimize the local repository.

## Synopsis

```
Usage:
  nt gc [flags]

Flags:
  -h, --help   help for gc
```

## Description

Runs a number of housekeeping tasks within the current repository, such as removing unreachable objects which may have been created from prior invocations of `nt add` or stale working trees. May also update ancillary indexes such as the `commit-graph`.

Running this command is safe when Git is used as files that will be deleted can still be reread using Git history.
