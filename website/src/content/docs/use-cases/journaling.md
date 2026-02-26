---
title: Journaling
---

## Journal

Keeping a journal is easy to do with Markdown files. You can simply write a new note every day.

```md title=journal.md
## Note: 2026-01-01

* 🏃 Ran 10k
* 🤔 3h (deep work)
```

_The NoteWriter_ defines a default type for this use case:

```jsonnet nt.libsonnet
{
    DefaultAttributes: {
        date: {
            name: "date",
            type: "string",
            format: "yyyy-mm-dd",
            inherit: true,
        },
    },
    DefaultNoteTypes: {
        Journal: self.Note + {
            name: "Journal",
            attributes: [
                {
                    name: "date",
                    required: true,
                },
            ],
            processors: ["date-extractor", "list-items"],
        },
    }
}
```

The type `Journal` does a few things:

* It requires an attribute `@date`.
* It automatically extract the date from the title (ex: `## Journal: 2026-01-01` will create an attribute `@date: 2026-01-01`).
* It expects a list (to display a list when rendering the journal).

Notes can be declared below a journal entry:

```md title=journal/2026/2026-01-01.md
# Journal: 2026-01-01

* ✅ Fixed bug at startup
* 📖 Finished reading _The Enchiridion_ by Epictectus
* 🥦 Eat healthy

## Note: Bug at startup

Some notes to explain how the bug was resolved.

## List: Food

* 🥗 Quinoa Salad with chickpeas, cherry tomatoes, cucumber, and a lemon-tahini dressing
* 🍛 Grilled salmon served with steamed broccoli and brown rice
* 🌯 Whole wheat wrap filled with hummus, spinach, shredded carrots, and avocado
```

_The NoteWriter Desktop_ provides a screen to visualize a diary using a timeline to filter on date range or attributes. Sub-notes are also listed (but not expanded by default).


## Routines

Routines plays an important role in journaling. Routines reduce friction. When you have a morning routine and a shutdown routine set up, it becomes an habit. Journaling without routines captures moments. Journaling with routines captures patterns.

Routines can be defined in `config.jsonnet`:

```jsonnet title=config.jsonnet
{
  journal: [
    {
      name: 'My Diary',
      path: 'journal/${year}/${year}-${month}-${day}.md',
      defaultContent: '# Journal: ${year}-${month}-${day}',
      routines: [
        {
          name: 'Morning Routine',
          template: |||
            # 💪 Affirmation

            <Affirmation wikilink="journaling#List: Affirmations" tags="success,optimism" />

            # 😘 Gratitude Journal

            3 things I appreciate:

            * <Input />
            * <Input />
            * <Input />

            # 🤔 Prompt

            <Prompt wikilink="journaling#List: Prompts" />

            # 🎯 My BIG thing for today

            <Input />
          |||,
        },
        {
          name: 'Shutdown Routine',
          template: |||
            # ❓ How was my day? Why?

            <Input />

            # 📋 3 tasks to complete tomorrow:

            * [ ] <Input />
            * [ ] <Input />
            * [ ] <Input />
          |||,
        },
      ],
    },
  ],
}
```

Routines will be appended to your journal entry specified using these attributes:

* `path`: A template to the path of note containing the daily journal entry. Ex: `journal/${year}/${year}-${month}-${day}.md`
* `defaultContent`: The default content for this note if the file doesn't exist. Ex: `# Journal: ${year}-${month}-${day}` will define the file title.

Several routines could be defined. For example:

```jsonnet
{
  name: 'Morning Routine',
  template: |||
    # 💪 Affirmation

    <Affirmation wikilink="journaling#List: Affirmations" tags="success,optimism" />

    # 😘 Gratitude Journal

    3 things I appreciate:

    * <Input />
    * <Input />
    * <Input />

    # 🤔 Prompt

    <Prompt wikilink="journaling#List: Prompts" />

    # 🎯 My BIG thing for today

    <Input />
    |||,
},
```

A routine is defined by its `name` and a `template`. A template is the text that will be rendered with placeholders for the user to enter:

* `<Input />`: A text input to answer a question, write a line, etc.
* `<Affirmation wikilink="journaling#List: Affirmations" />`: Render an affirmation selected in a list note identified by the attribute `wikilink`. The optional attribute `tags` can be used to filter affirmations.
* `<Prompt wikilink="journaling#List: Prompts" tags="doing" />`: Render a prompt and a text input to write your reflection on it. The optional attribute `tags` can be used to filter prompts.
* `<MorningPages throwAway />`: Render a textarea to fill your morning pages. The optional attribute `throwAways` can be used to not append the result in the final journal entry.

Example of notes for affirmations and prompts:

```md title=journaling.md
## List: Affirmations

* All I need is within me right now.
* I am not pushed by my problems; I am led by my dreams.
* Today will be a productive day.
* I stand up for what I believe.
* I can do anything I put my mind to.
* I can do better next time.
* Every day is a fresh start.

## List: Prompts

* What's ONE thing you can do today that makes tomorrow easier? `#doing`
* What's the ONE Thing I can do such that by doing it everything else will be easier or unnecessary? `#doing`
* What did you do as a child that made the hours pass like minutes? `#doing`
* If I can only achieve three things over the next three months, what should they be? `#planning`
* In what ways could it actually be an opportunity in disguise? `#reflecting`
* What is the 2-minute version of the task I'm avoiding? `#doing`
* What can I automate? `#doing`
```

Here is the rendered routine in the desktop application:

TODO screenshot morning routine

After saving, the daily file will be appended like this:

```md
TODO copy based on the screenshot
```

<!-- IMPROVEMENT Document Bullet Journal using attributes shorthands -->

## Stats

_The NoteWriter Desktop_ supports statistics determined on attributes.


### Daily Metrics
