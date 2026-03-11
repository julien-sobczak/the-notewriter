---
title: Tags
---

## Using Tags

Tags are defined using the attribute `tags`:

```md title=meditations.md
---
tags: [philosophy]
---

# Notes

## Quote: Memento Mori

`@source: _Meditations_` `@author: Marcus Aurelius`

You could leave life right now.
Let that determine what you do and say and think.
```

A short-hand syntax exists when declaring tags in Markdown. Both declarations are identical:

```md title=meditations.md
# Notes

## Quote: Memento Mori

`@tags: philosophy` `#philosophy`

You could leave life right now.
Let that determine what you do and say and think.
```

## Special Tags

_The NoteWriter_ reserves a few tags with built-in meanings:

| Tag | Description |
|-----|-------------|
| `#ignore` | Exclude the file or note from all processing |
| `#suspended` | Suspend a flashcard from SRS review |
| `#secure` | Mark a file or note as sensitive (excluded from remote synchronization) |
| `#bookmark` | Mark a note as a bookmark |

## Shorthand Tags

Similar to [shorthand attributes](./attributes.md#shorthand-attributes), tags can be configured with a shorthand symbol (typically an emoji) that, when present in a heading or list item, automatically applies the tag.

Tags are configured in `.nt/config.jsonnet`:

```jsonnet
local nt = import 'nt.libsonnet';

{
  tags: nt.DefaultTags + {
    favorite: {
      name: 'favorite',
      shorthand: '⭐',
      preserveShorthand: false,  // Default: true
    },
  },
}
```

The `DefaultTags` already contains shorthands for the special tags:

| Tag | Shorthand |
|-----|-----------|
| `#ignore` | 🚫 |
| `#secure` | 🔒 |
| `#bookmark` | 🏷️ |

With this configuration, writing `⭐` in a heading or list item is equivalent to adding `#favorite`:

```md
## Note: My Favorite Book ⭐

Content here...
```

The `preserveShorthand` property controls whether the shorthand symbol is kept in the rendered output:
- `preserveShorthand: true` (default): The emoji remains visible in the title
- `preserveShorthand: false`: The emoji is stripped from the title after the tag is extracted
