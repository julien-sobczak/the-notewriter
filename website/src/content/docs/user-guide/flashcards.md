---
title: Flashcards
---


:::caution

Flashcards can (for now) only be studied using _The NoteWriter Desktop_. Support in _The NoteWriter Nomad_ is planned. A [remote](remotes.md) will be required in this case.

:::

## Syntax

Flashcards are special notes of type `Flashcard` using an additional syntax. The front and the back cards must be separated by `---`.

Ex:

```md
## Flashcard: GTD's creator

**Who** created GTD?

---

David Allen
```

### Cloze Deletion

Cloze deletions are a powerful tool for creating fill-in-the-blank flashcards. To create a cloze deletion, wrap the text you want to hide in double curly braces with a unique number:

```md
## Flashcard: Cloze Deletion Syntax

A cloze deletion is rendering with an [{c1::hidden text}].
```

When studying this flashcard, `hidden text` will be hidden and revealed after answering the flashcard.

:::info
_The NoteWriter_ uses the [Anki syntax](https://docs.ankiweb.net/editing.html#cloze-deletion) for cloze deletions except curly braces are inside brackets.

This mixed syntax avoids conflicts with wikilinks (`[[...]]`) and popular templating systems (`{{...}}`).
:::

You can have multiple cloze deletions in a single note:

```md
[{c1::Water}] boils at [{c2::100°C}] at sea level.
```

Two flashcards will be generated from this note (`[{c1::Water}] boils at 100°C at sea level.` and `Water boils at [{c2::100°C}] at sea level`).

You can also provide a hint:

```md
Canberra was founded in [{c1::1913::year}].
```

The hint will be rendered to provide more context so that the expected answer is clear.


## Algorithm

The algorithm is heavily inspired by [Anki SM-2 variant](https://www.juliensobczak.com/inspect/2022/05/30/anki-srs.html). Under the hood, the feedback is stored as a number representing the confidence level, providing more flexibility to quickly bury a newly created flashcard you are already familiar with.


## Safe Updates

Flashcards are extracted like other notes when parsing a Markdown file. Unlike notes where the OID is randomly generated, the OID for flashcards is generated from the slug.

```md title=memory.md
## Flashcard: Method of Loci

What is the **origin** of the *Method of Loci*?

---

Originated in **ancient Greece** when poet **Simonides of Ceos** reportedly
identified banquet guests after a building collapse by recalling
where they had been seated.
```

By default, _The NoteWriter_ generates a slug for every note (ex: `memory-flashcard-method-of-loci`). When a Markdown file is edited, _The NoteWriter_ will parses the new file and tries to find existing notes in database matching the parsed note by searching for a note with the same slug.

When the content of a note is heavily edited, the slug will be different, a new note will be created and the old one will be garbage-collected.

For flashcards, it's interesting to be more explicit and declare the attribute `slug` explicitely:

## Flashcard: Method of Loci

`@slug: flashcard_method-of-loci-history`

What is the **origin** of the *Method of Loci*?

---

Originated in **ancient Greece** when poet **Simonides of Ceos** reportedly
identified banquet guests after a building collapse by recalling
where they had been seated.
```

As long as the slug is not updated, a flashcard will be retrieved successfully when editing its content and the history of study sessions will be preserved.

:::tip

Create flashcards for everything you need to have in your long-term memory.

:::
