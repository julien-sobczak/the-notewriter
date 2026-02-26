---
title: Syntax
---

This page documents all the special Markdown syntaxes supported by _The NoteWriter_.

## Notes

### Note Types

Notes are declared with a Markdown heading using the syntax `## Type: Title`:

```md
## Note: My First Note

Content of the note.

## Quote: A Famous Quote

> The journey of a thousand miles begins with a single step.
> -- Lao Tzu

## Flashcard: Capitals of Europe

What is the capital of France?

---

Paris
```

The note type is everything before the colon (e.g., `Note`, `Quote`, `Flashcard`). The title is everything after the colon (e.g., `My First Note`). Note types are defined in `.nt/config.jsonnet`.

Default built-in note types include:

| Type | Description |
|------|-------------|
| `Note` | A standard note |
| `Quote` | A quotation |
| `Flashcard` | A flashcard for spaced repetition |
| `List` | A list with structured items |
| `Todo` | A list of actionable tasks |
| `Task` | A single task |
| `Journal` | A journal entry (date extracted from title) |
| `Routine` | A routine (used in journal templates) |
| `Artwork` | A visual artwork |
| `Generator` | A script that generates notes dynamically |
| `ReadingList` | A list of books to read |
| `Master` | A master note to consolidate related notes |

You can declare custom note types in `.nt/config.jsonnet`:

```jsonnet title=.nt/config.jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes + {
        Idea: {
            name: "Idea",
        },
    },
}
```

### Note Comments

Notes can end with a personal comment, which appears separately in the desktop application. Use a Markdown blockquote at the very end of a note:

```md
## Quote: Wayne Gretzky on Trying

You miss 100% of the shots you don't take.

> Trying will always be more effective than doing nothing.
```

The blockquote at the bottom (preceded by a blank line) is treated as a personal annotation. It is displayed differently and can be hidden when rendering notes.

### Note Embedding

Notes can be embedded inside other notes using the unofficial Markdown syntax `![[wikilink#section]]`:

```md
## Note: On Productivity

Productivity is doing the right thing, not more things.

![[thoughts#Quote: Tim Ferris on Productivity]]
```

The referenced note is inlined at that position when rendered.

:::note

The wikilink must point to a specific note heading (e.g., `#Quote: Tim Ferris on Productivity`). Embedding an entire file with `![[filename]]` is not supported.

:::


## Attributes

### YAML Front Matter

Attributes can be declared in a YAML front matter block at the top of a file. All notes in the file inherit these attributes:

```md
---
author: Marcus Aurelius
source: Meditations
tags: [philosophy, stoicism]
---

# Meditations

## Quote: Memento Mori

You could leave life right now.
Let that determine what you do and say and think.
```

### Inline Attributes

Attributes can also be declared inline using the syntax `` `@name: value` ``:

```md
## Quote: Memento Mori

`@author: Marcus Aurelius` `@source: _Meditations_`

You could leave life right now.
Let that determine what you do and say and think.
```

Multiple attributes on their own line (no other content) are treated as a block of metadata and stripped from the rendered note body.

### Attribute Shorthands

Some attributes support Unicode character shorthands. For example, the predefined `rating` attribute can use star characters:

```md
## ReadingList: To Read

* _Meditations_ by Marcus Aurelius ★★★★★
* _Atomic Habits_ by James Clear ★★★★
```

Shorthands are defined per attribute in `.nt/config.jsonnet` using a `shorthands` map.

Status and priority attributes also support emoji shorthands:

```md
## Todo: My Tasks

* Write documentation ✅ ❗️
* Review PR 📅 🔽
```


## Tags

### YAML Front Matter Tags

Tags can be declared in the YAML front matter:

```md
---
tags: [philosophy, stoicism]
---
```

### Inline Tags

Tags can also be declared inline using the syntax `` `#tag-name` ``:

```md
## Quote: Memento Mori

`#philosophy` `#stoicism`

You could leave life right now.
```

Both syntaxes can be freely mixed. Tags declared in the front matter apply to all notes in the file. Inline tags apply only to the note where they appear.

### Tag for Ignoring

The special tag `#ignore` causes a note (or an entire file when in the front matter) to be skipped during indexing:

```md
## Note: Work In Progress

`#ignore`

This note will not be indexed.
```


## Reminders

Reminders are declared using a special tag syntax: `` `#reminder-{expr}` ``.

### Static Date

```md
## Todo: Renew Passport

`#reminder-2025-06-01`

Remember to renew your passport before it expires.
```

### Recurring Reminder

Use the `every-` keyword for recurring reminders:

```md
## Note: Annual Review

`#reminder-every-${year}-01-01`

Review your goals every January 1st.
```

### Reminder Expressions

| Expression | Description |
|------------|-------------|
| `#reminder-2025-06-01` | Static date |
| `#reminder-every-${year}-06-01` | Same date every year |
| `#reminder-every-${year}-${month}-01` | First day of every month |
| `#reminder-every-${year}-${odd-month}-01` | First day of every odd month |
| `#reminder-every-${year}-${even-month}-01` | First day of every even month |
| `#reminder-every-${odd-year}-06-01` | June 1st of every odd year |
| `#reminder-every-${even-year}-06-01` | June 1st of every even year |
| `#reminder-every-${day}` | Every day |
| `#reminder-every-${monday}` | Every Monday |
| `#reminder-every-${tuesday}` | Every Tuesday |
| `#reminder-every-${wednesday}` | Every Wednesday |
| `#reminder-every-${thursday}` | Every Thursday |
| `#reminder-every-${friday}` | Every Friday |
| `#reminder-every-${saturday}` | Every Saturday |
| `#reminder-every-${sunday}` | Every Sunday |

Reminders can also be attached to individual list items:

```md
## List: Conferences

- GopherCon USA
- GopherCon Europe `#reminder-every-${year}-06`
- dotGo Paris `#reminder-every-${year}-11`
```


## Gotos

A Goto is a memorable name for a URL, declared inside a Markdown link title using the syntax `#go/{name}`:

```md
## Note: Useful Links

[Golang](https://go.dev/doc/ "#go/go") is the official documentation.
[Go Playground](https://go.dev/play/ "#go/go/playground") for quick experiments.
```

Gotos can then be opened from the terminal:

```shell
$ nt go go
$ nt go go/playground
```

### Goto Placeholders

Goto URLs can include placeholders filled in at runtime:

```md
[GitHub Issue](https://github.com/${user}/${repo}/issues "${title} #go/github/issue")
```

Placeholder syntax:

| Syntax | Description |
|--------|-------------|
| `${var}` | Any value required |
| `${var:[a,b,c]}` | Must be one of `a`, `b`, or `c` |
| `${var:[a,b,...]}` | Suggestions: `a`, `b`, or any custom value |


## Wikilinks

Wikilinks are internal links to other notes, using double-bracket syntax:

```md
See also [[notes#Note: Example]] for more details.
```

### Forms

| Syntax | Description |
|--------|-------------|
| `[[filename]]` | Link to a file |
| `[[filename#Note: Title]]` | Link to a specific note in a file |
| `[[#Note: Title]]` | Link to a note in the current file |
| `[[filename\|display text]]` | Link with custom display text |

### Embedded Notes

Prefix a wikilink with `!` to embed the referenced note inline:

```md
![[filename#Note: Title]]
```


## Media Embedding

Medias (images, videos, audio) are embedded using the standard Markdown image syntax:

```md
![Profile photo](medias/me.png)

![Introduction video](medias/intro.mp4)

![Background music](medias/theme.mp3)
```

_The NoteWriter_ automatically converts media files:

| Input formats | Output |
|---------------|--------|
| `jpeg`, `png`, `gif`, `tiff`, ... | `avif` (thumbnail + full size) |
| `wav`, `aac`, `flac`, ... | `mp3` |
| `mp4`, `avi`, ... | `webm` + `avif` thumbnail |

External URLs are passed through unchanged and not stored internally.


## Flashcards

Flashcards are notes of type `Flashcard` with the front and back sides separated by a horizontal rule (`---`):

```md
## Flashcard: GTD's Creator

Who created **GTD** (Getting Things Done)?

---

**David Allen**
```

### Reversed Flashcards

Add the tag `#reversed` to automatically generate a second card with front and back swapped:

```md
## Flashcard: Go vs Golang

`#reversed`

Why is Go also known as Golang?

---

Because `go.org` was unavailable, so the team used `golang.org`.
```

### Cloze Deletion

Cloze deletions create fill-in-the-blank flashcards using the syntax `[{cN::hidden text}]` or `[{cN::hidden text::hint}]`:

```md
## Flashcard: Boiling Point

Water boils at [{c1::100°C}] at sea level.
```

When studying, `100°C` is hidden and replaced with `[...]`. Multiple groups can be defined in one note:

```md
## Flashcard: Boiling Point

[{c1::Water}] boils at [{c2::100°C}] at sea level.
```

This generates two flashcards — one hiding `Water` and one hiding `100°C`.

An optional hint can be added after a second `::`:

```md
## Flashcard: Foundation of Canberra

Canberra was founded in [{c1::1913::year}].
```

The hint (`year`) is displayed instead of `[...]` to provide context.

:::info

_The NoteWriter_ uses the [Anki cloze deletion syntax](https://docs.ankiweb.net/editing.html#cloze-deletion) but wraps curly braces inside square brackets (`[{c1::text}]`) to avoid conflicts with wikilinks (`[[...]]`) and templating systems (`{{...}}`).

:::


## Files

### Asciidoc Text Replacements

_The NoteWriter_ parses Markdown files but supports the same [character replacement substitutions](https://docs.asciidoctor.org/asciidoc/latest/subs/replacements/) as Asciidoc.


Ex (em-dash):

```md
And yet, when she was ready--she decided to stay here.
```

Is the same as:

```md
And yet, when she was ready—she decided to stay here.
```

Full list of supported substitutions:

| Input | Output | Name |
|-------|--------|------|
| `(C)` | © | Copyright |
| `(R)` | ® | Registered trademark |
| `(TM)` | ™ | Trademark |
| `--` | — | Em dash |
| `...` | … | Ellipsis |
| `->` | → | Right arrow |
| `=>` | ⇒ | Double right arrow |
| `<-` | ← | Left arrow |
| `<=` | ⇐ | Double left arrow |

Substitutions are not applied inside code spans (`` `code` ``) or fenced code blocks.

### Comments

_The NoteWriter_ strips two kinds of inline comments from note bodies before indexing:

**HTML comments** (stripped entirely):

```md
<!-- This will not appear in the indexed content -->
```

**Unofficial Markdown comments** (also stripped):

```md
<!--- This is also stripped --->
```

