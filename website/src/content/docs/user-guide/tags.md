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
