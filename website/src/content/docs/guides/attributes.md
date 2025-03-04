---
title: Attributes
---


Notes can be enriched with metadata using attributes.

## Syntax

Attributes are defined using a YAML Front Matter at the top of the file (similar to [Jekyll](https://jekyllrb.com/docs/front-matter/)):

```md title=meditations.md
---
source: _Meditations_
author: Marcus Aurelius
---

# Notes

## Quote: Memento Mori

You could leave life right now. Let that determine what you do and say and think.
```

All notes inherit attributes defined in the YAML front matter (restrictions can be defined using [schemas](./linter.md)).

Attributes can also be defined in Markdown using the syntax `@name: value`. The previous example can be rewritten like this:

```md title=meditations.md
# Notes

## Quote: Memento Mori

`@source: _Meditations_` `@author: Marcus Aurelius`

You could leave life right now. Let that determine what you do and say and think.
```

Both syntaxes can be mixed.


## Tags

Tags are defined using the attribute `tags`:

```md title=meditations.md
---
tags: [philosophy]
---

# Notes

## Quote: Memento Mori

`@source: _Meditations_` `@author: Marcus Aurelius`

You could leave life right now. Let that determine what you do and say and think.
```

A short-hand syntax exists when declaring tags in Markdown. Both declarations are identical:

```md title=meditations.md
# Notes

## Quote: Memento Mori

`@tags: philosophy` `#philosophy`

You could leave life right now. Let that determine what you do and say and think.
```

## Types

Attributes can be typed using [schemas](./linter.md).
