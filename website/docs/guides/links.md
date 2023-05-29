---
sidebar_position: 6
---

# Links


## Relation Links

Notes can reference each other using wikilinks (ex: `[[file#note]]`).

```md
## Note: A

Check note [[#A]].

## Note: B

Check note [[#B]].
```

Special attributes are also analyzed to determine the relations between notes.

* `references` (type: `array`)
* `source` (type: `string`)
* `inspirations` (type: `string`)

Wikilinks inside these attributes automatically generate relations (= links in _The NoteWriter Desktop_).

### `references`

:::tip

Use the `references` attribute to mention that another note **is referenced by** a website, a book, or another note.

:::

```md
## Note: A

`@references: https://random.website`
`@references: _A Random Book_`
`@references: [[#B]]`

A first note.

## Note B

A second note.
```

The last reference is similar to:

```md
## Note: A

A first note.

## Note: B

A second note referencing [[#A]]
```

### `source`

:::tip

Use the `source` attribute to remember if a note was collected from a book, a website, etc.
:::

```md
# Note: A

`@source: https://some.random.blog`
```

### `inspirations`

:::tip

Use the `inspirations` attribute to specify which work has inspired this note (a website, a book, another note, ...)

:::

```md
## Note: A

`@inspiration: [[books/book-A#Quote: On Note-Taking]]`

A note.
```

## Go Links

Markdown links can include in their title was is called a _Go link_, that is a memorable name for a hard-to-remember URL.

The syntax must follow the convention `#go/{name}`.

```md title:go.md
## Note: Useful Links

* [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
* [Go Playground](https://go.dev/play/ "#go/go/playground") is useful to share snippets.
```

Go links can be browse directly from the terminal:

```shell
$ nt go go
# Open a new tab in your browser to https://go.dev/doc/

# Or
$ nt go go/playground
```

You can also use Go links (more conveniently) since _The NoteWriter Desktop_ (no need to have a terminal open inside your notes repository).


:::tip

Use Go links for URL that you must visit frequently (ex: internal tools at work).

:::

