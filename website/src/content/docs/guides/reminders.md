---
title: Reminders
---

Reminders are special tags that determine a timestamp when a note must be reviewed.

Reminders are displayed when planning your day using the commands `nt bye` and `nt hi`.

## Syntax

The syntax must follow `#reminder-{expr}`. Recurring reminders must use the additional keyword `every-` like this `#reminder-every-{expr}`.

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

:::tip

Use reminders for notes only actionable in the future: places to visit with your kids, conference to attend, travel ticket registration, ...

:::
