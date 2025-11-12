---
title: Links
---

Notes can reference each other using wikilinks (ex: `[[file#note]]`).

```md
## Note: A

Check note [[#A]].

## Note: B

Check note [[#B]].
```

Special attributes are also analyzed to determine the links between notes.

* `references` (type: `string[]`)
* `source` (type: `string`)
* `inspirations` (type: `string[]`)

Wikilinks inside these attributes automatically generate links (= navigable links in _The NoteWriter Desktop_).

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
