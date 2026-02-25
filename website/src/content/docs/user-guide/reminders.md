---
title: Reminders
---

Reminders are special tags that determine a timestamp when a note must be reviewed.

## Syntax

The syntax must follow `#reminder-{expr}`. Recurring reminders must use the additional keyword `every-` like `#reminder-every-{expr}`.

## Examples

:::note

Timestamps are always relative. For this documentation, we consider today is 2023, January 1.

:::

| Tag | Description | Next Occurrence(s) |
|---|---|---|
| `#reminder-2023-02-01` | Static date | `2023-02-01` |
| `#reminder-every-${year}-02-01` | Same date every year | `2023-02-01`, `2024-02-01`, ... |
| `#reminder-${even-year}-02-01` | Same date every even year | `2023-02-01` |
| `#reminder-${odd-year}-02-01` | Same date every odd year | `2024-02-01` |
| `#reminder-every-2025-${month}-02` | Every beginning of month in 2025 | `2025-01-02`, `2025-02-02`, ..., `2025-12-02` |
| `#reminder-every-2025-${odd-month}` | Odd month with unspecified day | `2025-02-02`, `2025-04-02`, ..., `2025-12-02` |
| `#reminder-every-${day}` | Every day | `2023-01-01`, `2023-01-02`, ... |
| `#reminder-every-${tuesday}` | Every Tuesday | `2023-01-03`, `2023-01-10`, `2023-01-17`, ... |


## Usage

Reminders can be declared as a block attribute defined on a note:

```md title=go.md
## Todo: Attend GopherCon Europe

`#reminder-every-${year}-06`

Annual conference in Europe in June in different cities (Paris, Berlin).
```

A reminder `Attend GopherCon Europe` will be created and will become visible the desktop application a few days before the 1st June every year.

Reminders can also be declared as a inline attribute defined on a list note:

```md title=go.md
## List: Conferences

- GopherCon USA
- GopherCon Europe `#reminder-every-${year}-06`
- GopherCon UK
- dotGo Paris `#reminder-every-${year}-11`
- GoLab Italy `#reminder-every-${year}-10`
```

Different reminders will be created from this note.

:::tip

Use reminders for notes only actionable in the future: places to visit with your kids, conferences to attend, booking registrations, ...

:::
