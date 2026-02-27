---
title: "nt-book"
---

## Name

`nt-book` — Generate ePub and PDF books from notes.

## Synopsis

```
Usage:
  nt-book [command]

Available Commands:
  generate    Generate book from notes using pandoc

Flags:
  -h, --help   help for nt-book
      --v      enable verbose info output
      --vv     enable verbose debug output
      --vvv    enable verbose trace output
```

## Description

`nt-book` generates ePub and PDF books from notes using [pandoc](https://pandoc.org/). Books are defined in the `.nt/config.jsonnet` configuration file. The command reads the note contents matching a configured query and assembles them into a book.

PDF generation requires [weasyprint](https://weasyprint.org/) (`pip install weasyprint`).

## Subcommands

### `nt-book generate [book-title]`

Generate ePub and PDF books from notes based on the configuration. If no book title is specified, all configured books are generated.

## Options

* `--v`
  * Enable verbose info output.
* `--vv`
  * Enable verbose debug output.
* `--vvv`
  * Enable verbose trace output.

## Configuration

Books are defined in `.nt/config.jsonnet`. Each book supports the following fields:

* `title` (required) — Book title.
* `author` (required) — Array of author names.
* `language` — Language code (e.g., `"en-US"`).
* `format` (required) — Array of output formats: `"epub"` and/or `"pdf"`.
* `toc` — Generate a table of contents.
* `cover` — Optional path to a cover image.
* `chapters` — Array of chapters (see below).

Each chapter or section supports:

* `title` (required) — Section title.
* `subtitle` — Optional subtitle.
* `illustration` — Optional path to an image (relative to repository root).
* `text` — Direct text content.
* `query` — A search query to select notes.
* `notes` — A list of specific notes to include (by `wikilink` or `slug`).
* `sections` — Nested subsections.
* `pageBreaks` — Force page breaks between notes.
* `includeComments` — Include the comment field of notes.

Example configuration:

```jsonnet
books: [{
  title: "My Book",
  author: ["Author Name"],
  language: "en-US",
  format: ["epub", "pdf"],
  toc: true,
  chapters: [{
    title: "Chapter 1",
    sections: [{
      title: "Section 1",
      query: "type:Note #important"
    }]
  }]
}]
```

## Examples

* Generate all configured books:

        $ nt-book generate

* Generate a specific book by title:

        $ nt-book generate "My Book"

## See Also

* [`nt-add`](./nt-add.md) to index notes before generating a book
