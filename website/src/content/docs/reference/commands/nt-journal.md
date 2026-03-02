---
title: "nt-journal"
---

## Name

`nt-journal` — Create and manage daily journal entries.

## Synopsis

```
Usage:
  nt-journal [flags]

Flags:
  -h, --help   help for nt-journal
      --v      enable verbose info output
      --vv     enable verbose debug output
      --vvv    enable verbose trace output
```

## Description

`nt-journal` is an interactive tool to create and manage daily journal entries. It uses a terminal user interface (TUI) to guide you through selecting a journal and running a configured routine (e.g., a morning or evening check-in).

Journal entries are stored as markdown files in the path defined by the journal configuration (e.g., `journal/{{ year }}/{{ year }}-{{ month }}-{{day}}.md`). Routines provide structured templates with prompts and dynamic content (affirmations, gratitude prompts, etc.).

Journals and routines are defined in the `.nt/config.jsonnet` configuration file.

## Options

* `--v`
  * Enable verbose info output.
* `--vv`
  * Enable verbose debug output.
* `--vvv`
  * Enable verbose trace output.

## Configuration

Journals are defined in `.nt/config.jsonnet`. Each journal supports the following fields:

* `name` (required) — Journal name displayed in the selection prompt.
* `path` (required) — Path template for journal files (must be a valid Go template). Supports `year`, `month`, `day` and `date` custom functions.
* `defaultContent` — Default content for new entries (must be a valid Go template).
* `routines` — Array of routines to run (see below).

Each routine supports:

* `name` (required) — Routine name displayed in the selection prompt.
* `template` (required) — Markdown template for the routine content (must be a valid Go template). Supports custom functions such as `input`, `prompt`, and `affirmation`.

Example configuration:

```jsonnet
journals: [{
  name: 'My Diary',
  path: 'journal/{{ year }}/{{ year }}-{{ month }}-{{ day }}.md',
  defaultContent: 'Journal: {{ year }}-{{ month }}-{{ day }}',
  routines: [
    {
      name: 'Morning Routine',
      template: |||
        # 💪 Affirmation

        {{ affirmation "journaling#List: Affirmations" }}

        # 😘 Gratitude Journal

        * {{ input }}
        * {{ input }}
        * {{ input }}

        # 🎯 My BIG thing for today

        {{ input }}
      |||,
    },
    {
      name: 'Shutdown Routine',
      template: |||
        # ❓ How was my day? Why?

        {{ input }}

        # 📋 3 tasks to complete tomorrow:

        * [ ] {{ input }}
        * [ ] {{ input }}
        * [ ] {{ input }}
      |||,
    },
  ],
}]
```

## Examples

* Open the interactive journal prompt:

        $ nt-journal

## See Also

* [`nt-add`](./nt-add.md) to index the generated journal files
