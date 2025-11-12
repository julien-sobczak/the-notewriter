---
title: Linter
---

The linter enforces rules on your notes to ensure their syntax is consistent and makes easy to find them later. It's particularly interesting as your collection of notes grows over time.


## Configuration

By default, the linter ensures that attributes satisfy their definition. In addition, the linter also support a list of optional rules declared as functions in `nt.libsonnet` to use in `config.jsonnet`. For example:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes,
    linter: {
        rules: [
            // Declare rules below
            nt.LintRules.NoEmptyTitle(),
            nt.LintRules.NoDuplicateNoteTitle(),
            nt.LintRules.NoDuplicateSlug(),
            nt.LintRules.NoDanglingMedia(),
            nt.LintRules.NoDeadWikilink(),
        ],
    },
}
```

Rules have no severity to make sure that violations are fixed instead of accumulating them.


## Rules

| Rule  | Description  | Arguments  |
|---|---|---|
| `no-empty-title` | Enforce all notes have a non-empty title | - |
| `no-duplicate-note-title` | Enforce no duplicate between note titles inside the same file | - |
| `no-duplicate-slug` | Enforce no duplicate slugs between notes across files | - |
| `no-implicit-slug-on-flashcard` | Enforce explicit slugs on flashcards to preverse study history on rewriting | - |
| `min-lines-between-notes` | Enforce a minimum number of lines between notes | <ul><li><code>int</code> The number of lines</li></ul> |
|	`max-lines-between-notes` | Enforce a maximum number of lines between notes | <ul><li><code>int</code> The number of lines</li></ul> |
|	`no-dangling-media` | Path to media files must exist | - |
|	`no-dead-wikilink` | Links between notes must exist | - |
|	`no-extension-wikilink` | No extension in wikilinks | - |
|	`no-ambiguous-wikilink` | No ambiguity in wikilinks | - |
|	`require-tag-if` | At least one tag on notes matching the query must match | <ul><li><code>string</code> A query to match notes</li><li><code>string[]</code> A list of tags</li></ul> |


### `no-empty-title`

Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoEmptyTitle(),
        ],
    }
}
```

Example (with violations highlighted):

```md {3}
# Example

## Note:

This is a note.
```

:::tip

Use the rule `no-empty-title` to ensure you haven't forgotten to write the title.

:::

### `no-duplicate-note-title`

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoDuplicateNoteTitle(),
        ],
    }
}
```

Example (with violations highlighted):

```md {15}
# Example

## Note: The same title is allowed on different types

This is a note.

### Flashcard: The same title is allowed on different types

This is a flashcard.

## Note: Long title must be unique inside a file

This is a note.

## Note: Long title must be unique inside a file

This is a note with the same title.
```

:::tip

Use the rule `no-duplicate-note-title` to ensure internal links are not ambiguous.

:::

### `no-duplicate-slug`

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoDuplicateSlug(),
        ],
    }
}
```

Example (with violations highlighted):

```md {11,17}
# Example

## Note: The slug attribute is supported

`@slug: note1`

This is a note.

### Note: The same slug cannot be reused

`@slug: note1`

This is a note.

## Note: Slugs must be compatible with URLs

`@slug: not a valid slug`

This is a note.
```

:::tip

Use the rule `no-duplicate-slug` to ensure slugs can be used in URLs and match only a single note.

:::

### `min-lines-between-notes`

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.MinLinesBetweenNotes(2),
        ],
    }
}
```

Example (with violations highlighted):

```md {7,15}
# Example

## Note: One

This is the first note.

## Note: Two

This is the second note.


## Note: Three

This is the third note.
## Note: Four
This is the fourth note.
```

:::tip

Use the rule `min-lines-between-notes` to force spaces between your notes to make editing easier than often result from rough editing session.

:::

### `max-lines-between-notes`


Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.MaxLinesBetweenNotes(1),
        ],
    }
}
```

Example (with violations highlighted):

```md {6,16}
# Example




## Note: One

This is the first note.

## Note: Two

This is the second note.



## Note: Three

This is the third note.


## Note: Four
This is the fourth note.

```

:::tip

Use the rule `max-lines-between-notes` to avoid too many blank spaces between notes.

:::

### `no-dangling-media`

Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoDanglingMedia(),
        ],
    }
}
```

Example (with violations highlighted):

```md {3,5}
# Example

![Missing directory](pic.jpeg)
![OK](no-dangling-media/pic.jpeg)
![Wrong extension](no-dangling-media/pic.jpg)
![OK](no-dangling-media/../no-dangling-media/pic.jpeg)
![OK](./no-dangling-media/pic.jpeg)
```

:::tip

Use the rule `no-dangling-media` to ensure links to medias are correctly resolved.

:::

### `no-dead-wikilink`


Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoDeadWikilink(),
        ],
    }
}
```

Example (with violations highlighted):

```md title=no-dead-wikilink.md {5,7}
# Example

## Note: A

[[#B]]
[[#Note: B]]
[[unknown.md]]

## Note: B

[[no-dead-wikilink.md#Note: A]]
[[no-dead-wikilink#Note: A]]
```

:::tip

Use the rule `no-dead-wikilink` to ensure links are not dead (useful after renaming for example).

:::

### `no-extension-wikilink`


Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoExtensionWikilink(),
        ],
    }
}
```

Example (with violations highlighted):

```md title=no-extension-wikilink.md {9,17}
# Example

## Note: Link 1

[[no-extension-wikilink#Note: Link 2]]

## Note: Link 2

[[no-extension-wikilink.md#Note: Link 1]]

## Note: Link 3

[[no-extension-wikilink]]

## Note: Link 4

[[no-extension-wikilink.md]]
```

:::tip

Use the rule `no-extension-wikilink` to keep your internal links as short as possible.

:::

### `no-ambiguous-wikilink`

Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.NoAmbiguousWikilink(),
        ],
    }
}
```

Example (with violations highlighted):

```md {3}
# Example

[[books.md]]
[[notes/books.md]]
[[reviews/books.md]]
```

:::tip

Use the rule `no-ambiguous-wikilink` to ensure links are explicit and can be followed properly.

:::

### `require-tag-if`


Configuration:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    linter: {
        rules: [
          nt.LintRules.RequireTagIf("type:Quote", [
                "learning",
                "mastering",
          ])
        ],
    }
}
```

Example (with violations highlighted):

```md {3,10}
# Example

## Quote: No Tag

`@name: Anonymous`

This is the first quote.


## Quote: Tag

`@name: Anonymous`
`#useless`

This is the second quote.
```

:::tip

Use the rule `require-tag-if` to enforce some notes have tags and use the argument to limit the list of required tags.

:::
