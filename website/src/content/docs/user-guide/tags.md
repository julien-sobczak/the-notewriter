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
| `#secure` | Ensure the file was encrypted using `nt-vault` when adding it using `nt add` |
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
| `#bookmark` | ⭐ |

With this configuration, writing `⭐` in a heading or list item is equivalent to adding `#bookmark`:

```md
## Note: My Favorite Book ⭐

Content here...
```

The `preserveShorthand` property controls whether the shorthand symbol must be kept in the rendered Markdown (shorthands are preserved by default, use `false` to strip them).
