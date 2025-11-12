---
title: Attributes
---

Notes can be enriched with metadata using attributes.

## Using Attributes

Attributes can be defined using a YAML Front Matter at the top of the file (similar to [Jekyll](https://jekyllrb.com/docs/front-matter/)):

```md title=meditations.md
---
source: _Meditations_
author: Marcus Aurelius
---

# Notes

## Quote: Memento Mori

You could leave life right now.
Let that determine what you do and say and think.
```

All notes inherit attributes defined in the YAML front matter.

Attributes can also be defined in Markdown using the syntax `@name: value`. The previous example can be rewritten like this:

```md title=meditations.md
# Notes

## Quote: Memento Mori

`@source: _Meditations_` `@author: Marcus Aurelius`

You could leave life right now.
Let that determine what you do and say and think.
```

In practice, both syntaxes can be mixed.


## Attribute Types

Attributes have a type that is dynamically determined during parsing.

```md
---
pages: 305
---

# Meditations, by Marcus Aurelius

Some reading notes...
```

The attribute `pages` will be of type `integer`.

Several primitive types exists:

| Type      | Examples                                  |
|-----------|-------------------------------------------|
| `string`  | `"Meditations"`, `"Marcus Aurelius"`      |
| `integer` | `305`, `2024`                             |
| `float`   | `3.14`, `0.5`                             |
| `bool`    | `true`, `false`                           |
| `date`    | `2024-06-01`, `2023-12-31`                |

Arrays containing primitive values are also supported. Use a YAML array in Front Matter or repeat the attribute in Markdown. Both syntax are valid:

```md
---
references:
- _Ego Is the Enemy_, by Ryan Holiday
- _How to Think Like a Roman Emperor_, by Donald Robertson
---

# Meditations, by Marcus Aurelius

## Quote: Memento Mori

`@references: _Ego Is the Enemy_, by Ryan Holiday`
`@references: _How to Think Like a Roman Emperor_, by Donald Robertson`

You could leave life right now.
Let that determine what you do and say and think.
```

## Reserved Attributes

_The NoteWriter_ automatically defines a list of reserved attributes required for the inner working of the application:

| Name          | Type       | Inherited |
|---------------|------------|-----------|
| `title`       | `string`   | No        |
| `slug`        | `string`   | No        |
| `tags`        | `string[]` | Yes       |
| `date`        | `date`     | Yes       |
| `hook`        | `string[]` | No        |
| `source`      | `string`   | Yes       |
| `references`  | `string[]` | No        |
| `inspirations`| `string[]` | No        |

These attributes cannot be overriden. Some attributes are inherited, which means notes declared as children will inherit them.

```md
# Note: Parent

`@hook: gist` `#some-tag`

## Note: Child

The child note have the tag `#some-tag` automatically
defined but the `hook` attribute is not inherited.
```

* The attribute `@title` is automatically added to every note and contains the Markdown heading (ex: `Note: Parent`), which is sometimes useful in searches.
* The attribute `@slug` is generated from the title but can be explicitely defined. When editing a Markdown document, the code will try to find existing notes matching the slug. This is especially useful for flashcards to avoid recreating a new card when a note is edited but we want to preserve the study history.
* The attribute `@date` represents a single day and is useful mostly for [journaling](../examples/my-workflow/journaling).
* The attribute `@hook` is used to trigger user-defined scripts automatically for automation and is covered in its [own page](./hooks).
* The attributes `@source`, `@references`, `@inspirations` are used to link the notes together and are covered in their [own page](./links) too.


## Predefined Attributes

Some attributes are also predefined but can be overriden in `config.jsonnet`.

```
// Default configuration file
local nt = import 'nt.libsonnet';
{
    attributes: nt.DefaultAttributes
    // ...
}
```

Check the generated file `.nt/libsonnet` for the exhaustive list. Here is an example:

```jsonnet
{
    DefaultAttributes: {
        status: {
            name: "status",
            aliases: ["state"],
            description: "Status of a task",
            type: "string",
            options: [
                "todo", "planned",
                "in-progress", "done",
                "cancelled", "on-hold",
                "blocked", "archived"
            ],
            defaultValue: "todo",
        }
        // ... more attributes
    }
}
```

Predefined attributes can be redeclared, which could be useful to overriden some properties or use shorthands.

## Custom Attributes

Custom Attributes, in addition to [custom note types](./note-types) and the [linter](./linter) are one the the key features to manage a large collection of notes over time.

Attributes explicitely defined (like the predefined ones) are enforced when running `nt add`. You cannot add a file containing invalid attributes.

Here is the default configuration file:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes,
}
```

You can extend the default attributes like this:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes + {
        nationality: {
            name: "nationality",
            type: "string",
            format: "markdown",
            inherit: true, // Often declared in Front Matter
        },
    }
    noteTypes: nt.DefaultNoteTypes,
}
```

Only properties `name` and `type` are required (see the above [list of types](./attributes.md#attribute-types)).

Additional properties are supported:

* `format`: Only for attributes with type `string` or `string[]`. The value must match the expected format (among `markdown`, `isbn`, `date`).
* `pattern`: Also only for attributes with type `string` or `string[]`. The value must match the given regex.
* `inherit`: `false` to present an attribute to be inherited by sub-notes. Attributes are inherited by default.
* `min`/`max`: Only for attributes with type `integer`. The value must be comprised between `min` and `max` (inclusive).

Here is another example of a custom attribute:

```jsonnet
    recommendation: {
        name: "recommendation",
        type: "integer",
        min: 0,
        max: 10,
    },
```

## Shorthand Attributes

Imagine an attribute `@rating: 1-10` useful when reviewing items like a book:

```md
## Todo: Reading List

* _Creativity, Inc._ by Ed Catmull `@isbn: 978-0593594643` `@rating: 6`
* _Man's Search for Meaning_ by Viktor Frankl `@isbn: 080701429X` `@rating: 8`
* _Quiet_ by Susan Cain `@isbn: 978-0307352149` `@rating: 10`
```

Using shorthands, the attribute `@rating` can be declared using emojis:

```md
## Todo: Reading List

* _Creativity, Inc._ by Ed Catmull `@isbn: 978-0593594643` ★★★
* _Man's Search for Meaning_ by Viktor Frankl `@isbn: 080701429X` ★★★★
* _Quiet_ by Susan Cain `@isbn: 978-0307352149` ★★★★★
```

Here is the definition of the predefined attribute `rating`:

```jsonnet
rating: {
    name: "rating",
    description: "Rating",
    type: "integer",
    min: 0,
    max: 10,
    shorthands: {
        "☆": 0,
        "\u2BE8": 1,
        "★": 2,
        "★\u2BE8": 3,
        "★★": 4,
        "★★\u2BE8": 5,
        "★★★": 6,
        "★★★\u2BE8": 7,
        "★★★★": 8,
        "★★★★\u2BE8": 9,
        "★★★★★": 10,
    }
}
```

The magic is revealed by the property `shorthands`. If a key is present in a Markdown heading or in a Markdown list (⚠️ shorthands are ignored inside the body of a standard note), the attribute will automatically be added with the value matching the key as outlined by our previous examples with books.

Shorthands are particularly useful during [journaling](../examples/my-workflow/journaling.md), like when using the Bullet Journal Method.
