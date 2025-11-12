---
title: Memories
---

Memories are past events you want to review at their anniversary date. Memories are defined using an `string` attribute with the format `date` and for which the special attribute `memory` is set to `true`:

```md title=config.jsonnet
local nt = import 'nt.libsonnet';
{
    attributes: nt.DefaultAttributes + {
        read_date: {
            name: "read_date",
            type: "string",
            format: "date",
            memory: true, // Used to mark this note as memory
        },
    },
}
```

We use a list of read books to illustrate memories:

```md title=readings.md

## List: Read Books

- **How to Take Smart Notes** by Sönke Ahrens [[how-to-take-smart-notes]] `@read_date: 2022-03-15`
- **The Bullet Journal Method** by Ryder Carroll [[the-bullet-journal-method]] `@read_date: 2021-11-10`
- **Getting Things Done** by David Allen [[getting-things-done]] `@read_date: 2020-07-05`
- **The Organized Mind** by Daniel J. Levitin [[the-organized-mind]] `@read_date: 2019-09-22`
```

Memories will be rendered along [reminders](./reminders.md) in the notifications panel in _The NoteWriter Desktop_.

Another example is to have an accomplishement list:

```md
## List: Accomplishements

- Ran a half marathon `@completion_date: 2021-12-21`
- Redesigned my resume `@completion_date: 2025-12-11`
- Released first version of _The NoteWriter_ `@completion_date: 2026-01-01`
```

Using memories reminds you of all the work you've already completed and helps you keep going with your current project.

:::tip

Use memories on notes you want to revisit over time.

:::
