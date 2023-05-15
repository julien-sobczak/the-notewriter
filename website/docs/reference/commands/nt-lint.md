---
sidebar_position: 10
---

# `nt lint`

## Name

`the-notewriter lint` — Check all rules for possible violations.

## Synopsys

```
Usage:
  nt lint [flags] [--] [<pathspec>]

Flags:
  -h, --help           help for lint
  -r, --rules string   comma-separated list of rule names used to filter (default "all")
```

## Description


## Options

* `<pathspec>`...
  * Files to validate using the same syntax as supported by [`nt add`](./nt-add.md).

## Configuration

Rules are declared in file `.nt/lint`. See the [guide "Lint"](../../guides/linter.md) for additional information.

## Examples

* Run all rules on all files:

        $ nt rules

* Run all rules on a given file:

        $ nt rules -- references/books/a-mind-for-numbers.md

* Run only the rule `check-attributes`:

        $ nt rules --rules=check-attributes

## See Also

* [`nt-add`](./nt-add.md) to add new contents satisfying the linter rules

