---
sidebar_position: 1
---

# Notes

## Format

### General

Notes are written in Markdown files (GitHub Flavored Markdown is supported).

```md title=notes.md
# A Basic Note

Notes can use **Markdown syntax**.
```

The note title is defined by the heading and the note content continues until the next heading (or until the end of file).

Since notes are used for so many things, _The NoteWriter_ supports different kinds (see below) using a prefix:

```md title=notes.md
# Quote: Epictetus on Learning

You can't learn if you think you already know.
```

A file will often contain several notes:

```md title=notes.md
# My Notes

## Note: A Basic Note

A first note.

## Quote: A Basic Quote

A first quote.
```

A note can contain headings:

```md title=notes.md
# My Notes

## Note: A Structured Note

A first sentence.

### Subsection 1

A first subsection.
```

The subsection "Subsecton 1" is included in the note `A Structured Note`.


## Kinds

### `Note` to clear your mind

Use `Note` for anything you don't want to forget.

    ## Note: Dark Mode

    The dark mode will allow users to switch the application in dark mode.

### `Flashcard` to remember

Use `Flashcard` for knowledge you want to remember.

    ## Flashcard:  The Zeigarnik Effect

    (Learning) What is the **Zeigarnik effect**?

    ---

    **The way unfinished tasks remain active in our mind**, intruding into our thoughts and our sleep until they are dealt with.

Check the guide ["Flashcards"](./flashcards.md) to learn more about the syntax and the algorithm.

### `Cheatsheet` to repeat faster

Use `Cheatsheet` for actions you don't want to relearn from scratch every time.

    ## Cheatsheet: Table-Driven Tests in Go

    ```go
    func TestFib(t *testing.T) {
            var tests = []struct {
                    name     string
                    n        int    // input
                    expected int    // output
            }{
                    {"1", 1, 1},
                    {"2", 2, 1},
            }
            for _, tt := range tests {
                    t.Run(tt.name, func(t *testing.T) {
                            actual := Fib(tt.n)
                            if actual != tt.expected {
                                    t.Errorf("Fib(%d): expected %d, actual %d", tt.n, tt.expected, actual)
                            }
                    })
            }
    }
    ```

### `Quote` to get inspired

Use `Quote` for inspiring quotes that resonate with you.

    ## Quote: Albert Einstein about Cluttered Desks

    If a cluttered desk is a sign of a cluttered mind, of what, then is an empty desk a sign?

### `Reference` to use later

Use `Reference` for information you may need in the future.

    ## Reference: Best Note-Taking Books

    * _How to Take Smart Notes_, by Sönke Ahrens
    * _Sönke Ahrens_, by Tiago Forte

### `TODO` to plan tasks

Use `TODO` for tasks you need to perform.

    ## TODO: Reading List

    * [x] _Tools Of Titans_, by Tim Ferris
    * [ ] _Stumbling on Happiness_, by Daniel Gilbert
    * [ ] _Bittersweet_, by Susan Cain

### `Journal` to track your day

Use `Journal` for tasks you need to perform.

    ## Journal: 2023-01-01

    * 📁 Complete documentation
      * [x] Add doc about the different kinds of note
      * [ ] Add doc about attributes and tags

### `Artwork` to get inspired

Use `Artwork` for artworks that resonate with you.

**TODO**



### Free notes

Notes can omit the kind prefix. Their are called "free" notes and are processed like any other notes (they are searchable).

In practice, defining the note kind adds metadata that can be useful when searching in your notes. In addition, some notes like flashcards requires the kind to be defined.

:::tip

* **Use kinds to classify your notes**, to make easy to retrieve them or to restrict when searching for specific notes.
* **Use the [linter](linter.md) rule `no-free-note`** if you want to enforce a kind on all notes.

:::



## Extended Syntax


### Ignore Files

Files or notes with a tag `ignore` are ignored and not present in the index, that is not searchable.

```md
---
tags: ignore
---

# My Special Document

This document is ignored.
```

Or

```md
## Note: A Parsed Note

This note will be indexed.

## Note: An Ignored Note

`#ignore`

This note will be ignored.
```

:::tip

Use ignorable files for free-editing Markdown files when working on projects.

:::


### Quote Shorthand Syntax

Markdown quotes can be defined using the common Mardown syntax:

```md
## Quote: Me

> Something not very interesting.
>
> — Me
```

_The NoteWriter_ automatically convert your notes of kind `Quote` to this syntax. The previous note can be rewritten:

```md
## Quote: Me

`@author: me`

Something not very interesting.
```

If a `source` attribute is defined, the content will be appended to the author when rendered in HTML.


### Embed Files

Notes can be embedded inside another notes using the not-official Markdown syntax `![[wikilink-to-file#note-section]]`


:::important

The wikilink must include a reference to a Markdown heading inside the referenced file.

:::

Example:

```md title=notes.md
## Quote: Tim Ferris on Productivity

`@author: Tim Ferris`

Focus on being productive instead of busy.

## Note: On Busyness

Productivity is doing the right thing. Doing less useless things, and doing more important things.

![[#Quote: Tim Ferris on Productivity]]
```

The second note is equivalent to:

```md
## Note: On Busyness

Productivity is doing the right thing. Doing less useless things, and doing more important things.

> Focus on being productive instead of busy.
>
> — Tim Ferris
```


### Comments

All notes can end with a comment (using the Mardown common syntax for quotations):

```md
## Quote: Wayne Gretzky on Trying

`@author: Wayne Gretzky`

You miss 100% of the shots you don’t take.

> Trying will always more effective than doing nothing.
```

These comments are useful to explain why a note like a quote resonates in you, or to summarize the key idea. These comments are highlighted differently (or ommitted) when rendered in _The NoteWriter Desktop_.


## Asciidoc Text Replacements

_The NoteWriter_ parses Markdown files but support the same [character replacement substitutions](https://docs.asciidoctor.org/asciidoc/latest/subs/replacements/) as Asciidoc.


Ex (em-dash):

```md
And yet, when she was ready--she decided to stay here.
```

Is the same as:

```md
And yet, when she was ready—she decided to stay here.
```
