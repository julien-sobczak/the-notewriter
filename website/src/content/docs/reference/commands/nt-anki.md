---
title: "nt-anki"
---

## Name

`nt-anki` — Convert Anki flashcards to notes.

## Synopsis

```
Usage:
  nt-anki [command]

Available Commands:
  import      Import Anki flashcards from an .apkg file

Flags:
  -h, --help          help for nt-anki
  -t, --parallel int  Number of workers to use when generating blobs
      --v             enable verbose info output
      --vv            enable verbose debug output
      --vvv           enable verbose trace output
```

## Description

`nt-anki` is an extra tool to convert Anki flashcards to notes. It reads an Anki package file (`.apkg`) and generates a markdown file containing the flashcards as notes.

An `.apkg` file is a ZIP archive containing:

* `collection.anki2` — SQLite database with notes and cards.
* `media` — JSON file mapping media IDs to filenames.
* Numbered files (`0`, `1`, etc.) — The media files.

After importing, review the generated markdown file and run `nt add` to index it.

## Subcommands

### `nt-anki import <anki.apkg> <markdown.md>`

Import flashcards from an Anki `.apkg` file and write them to a markdown file.

**Flags:**

* `--media-dir <DIR>`
  * Subdirectory for media files relative to the output markdown file.
* `--ignore-scheduling`
  * Ignore scheduling information from the revlog table. Outputs only the markdown file and media without creating a packfile.

## Options

* `-t`, `--parallel <N>`
  * Number of workers to use when generating blobs.
* `--v`
  * Enable verbose info output.
* `--vv`
  * Enable verbose debug output.
* `--vvv`
  * Enable verbose trace output.

## Examples

* Import an Anki deck into a markdown file:

        $ nt-anki import english.apkg skills/english.md

* Import with media files in a subdirectory:

        $ nt-anki import english.apkg skills/english.md --media-dir medias

* Import without scheduling information:

        $ nt-anki import english.apkg skills/english.md --ignore-scheduling

## See Also

* [`nt-add`](./nt-add.md) to index the generated markdown file
