---
sidebar_position: 4
---

# Linter

The linter enforces rules on your notes to ensure their syntax is consistent and makes easy to find them later. It's particularly interesting as your collection of notes grows over time.

## Configuration

The linter reads its configuration from the YAML file `.nt/lint`. No file exists by default. Ex:

```yaml
rules:
- name: min-lines-between-notes
  args: [2]
- name: no-free-note
  includes:
  - references/
- name: no-dangling-media
- name: no-dead-wikilink
```

Rules are declared under the attribute `rules`. Some rules accept arguments using the attribute `args` (array of primitive values) and all rules can be restricted to apply on a subset of your notes using the attribute `includes` (array of glob path expressions).


## Rules

| Rule  | Description  | Arguments  |
|---|---|---|
| `no-duplicate-note-title` | Enforce no duplicate between note titles inside the same file | - |
| `min-lines-between-notes` | Enforce a minimum number of lines between notes | <ul><li><code>int</code> The number of lines</li></ul> |
|	`max-lines-between-notes` | Enforce a maximum number of lines between notes | <ul><li><code>int</code> The number of lines</li></ul> |
|	`note-title-match` | Enforce a consistent naming for notes | <ul><li><code>string</code> A Golang regex</li></ul> |
|	`no-free-note` | Forbid untyped notes | - |
|	`no-dangling-media` | Path to media files must exist | - |
|	`no-dead-wikilink` | Links between notes must exist | - |
|	`no-extension-wikilink` | No extension in wikilinks | - |
|	`no-ambiguous-wikilink` | No ambiguity in wikilinks | - |
|	`require-quote-tag` | At least one tag on quotes (must match the optional pattern) | <ul><li><code>string</code> A regex that must match all accepted tags on quotes</li></ul> |
|	`check-attribute` | Attributes must satisfy their schema if defined (see below) | - |


### `no-duplicate-note-title`

Configuration:

```yaml title=.nt/lint
rules:
- name: no-duplicate-note-title
```

Example (with violations highlighted):

```md {15}
# Example

## Note: The same title is allowed on different kinds

This is a note.

### Flashcard: The same title is allowed on different kinds

This is a flashcard.

## Note: Long title must be unique inside a file

This is a note.

## Note: Long title must be unique inside a file

This is a note with the same title.
```

:::tip

Use the rule `no-duplicate-note-title` to ensure internal links are not ambiguous.

:::

### `min-lines-between-notes`

Configuration:

```yaml title=.nt/lint
rules:
- name: min-lines-between-notes
  args:
  - 2
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

```yaml title=.nt/lint
rules:
- name: max-lines-between-notes
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

### `note-title-match`


Configuration:

```yaml title=.nt/lint
rules:
- name: note-title-match
  args:
  - "^(Note|Reference):\s\S.*$"
```

Example (with violations highlighted):

```md {7}
# Example

## Reference: Example

A title matching the regular expression `(Reference|...):\s\S.*`.

## reference: Example

The kind is in lowercase (allowed but enforced by the linter).

```

:::tip

Use the rule `note-title-match` to apply naming conventions on your notes.

:::

### `no-free-note`


Configuration:

```yaml title=.nt/lint
rules:
- name: no-free-note
```

Example (with violations highlighted):

```md {3}
# Example

## A free note

This is a free note.

## Note: A typed note

This is a typed note.

## Cheatsheet: Another typed note

This is another typed note.
```

:::tip

Use the rule `no-free-note` to catch notes that are wrongly or not typed.

:::

### `no-dangling-media`


Configuration:

```yaml title=.nt/lint
rules:
- name: no-dangling-media
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

```yaml title=.nt/lint
rules:
- name: no-dead-wikilink
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

```yaml title=.nt/lint
rules:
- name: no-extension-wikilink
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

```yaml title=.nt/lint
rules:
- name: no-ambiguous-wikilink
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

### `require-quote-tag`


Configuration:

```yaml title=.nt/lint
rules:
- name: require-quote-tag
  args:
  - "^(life|favorite)$"
```

Example (with violations highlighted):

```md {7,14}
# Example

## Note: A Note

Only quotes are concerned by this rule.

## Quote: No Tag

`@name: Anonymous`

This is the first quote.


## Quote: Tag

`@name: Anonymous`
`#useless`

This is the second quote.
```

:::tip

Use the rule `require-quote-tag` to enforce all quotes have tags and use the argument to limit the list of required tags.

:::

### `check-attribute`


Configuration:

```yaml title=.nt/lint
rules:
- name: check-attribute

schemas:

- name: Quotes
  kind: quote
  path: references
  attributes:
    - name: name
      aliases: [author, illustrator]
      type: string
      required: true
```

Example (with violations highlighted):

```md {7}
# Example

## Note: Marcus Aurelius On Others

> What’s bad for the hive is bad for the bee.

## Quote: Summum Bonum

Just that you do the right thing.

## Quote: Memento Mori

`@author: Marcus Aurelius`

You could leave life right now. Let that determine what you do and say and think.
```

:::tip

Use the rule `check-attribute` to ensure all values are valid and consistent between notes.

:::

## Schemas

Schemas are used to defined attributes and must follow this structure:

```yaml title=.nt/lint
schemas:

- name: Quotes          # A name used when reporting violations
  kind: quote           # Restriction on the note kinds
  path: references      # Restriction on the note path (glob pattern)
  attributes:           # Define a list of attributes
    - name: name        # The attribute name
      aliases: [author] # Optional aliases for the attribute name
      type: string      # One of: array, string (default), boolean, number, object
      required: true    # Mandatory? (default: false)
      inherit: true     # Attribute is inheritable by sub-notes? (default: true)
```

Default schemas (important for the inner working of the application) are predefined:

```yaml
schemas:

  - name: Hooks
    attributes:
    - name: hook
      type: array
      inherit: false

  - name: Tags
    attributes:
      - name: tags
        type: array

  - name: Relations
    attributes:
      - name: source
        inherit: false
      - name: references
        type: array
      - name: inspirations
        type: array
```

Declaring attributes as `array` is convenient as value will automatically be appended to existing values:

```md
---
tags: life # Same as tags: [life]
---

# A Note

`@tag: life-changing`

This note will have the tags `#life` and `#life-changing`.
```

:::caution

Schemas are only enforced when enabling the rule `check-attribute`.

:::
