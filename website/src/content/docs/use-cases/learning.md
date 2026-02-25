---
title: Learning
---

## Decks

Flashcards must be first grouped into decks declared in `config.jsonnet`:

```jsonnet
local srsAlgorithmSettings = {
    easeFactor: 2.5,
}

{
    decks: [
        {
            name: "Skills",
            query: "path:areas/skills",
            newFlashcardsPerDay: 10,
            algorithmSettings: srsAlgorithmSettings,
        }
        {
            name: "English",
            query: "path:areas/english",
            newFlashcardsPerDay: 50,
            algorithmSettings: srsAlgorithmSettings,
        }
    ],
}
```

The `algorithmSettings` is used to tune the [SRS algorithm](../user-guide/flashcards.md) by providing one of these values:

* `easeFactor` (type `number`): The default ease factor for new cards. Used to calculate intervals for reviewing. (default: `2.5`)
* `steps` (type `string[]`) An array of interval steps for learning (default: `['1m', '10m', '1d']`). These steps define the progression for cards in the learning queue.

In addition, flashcards can declare [attributes](../user-guide/attributes.md) to finely tune the interval scheduling logic:

* The optional attribute `ease_factor` overrides the value in the deck configuration.
* The optional attribute `interest_factor` can provided to boost the scheduling interval.

These two attributes are useful to boost the scheduling interval between reviews depending your interest on the topic. It's usually easier to remember on topics where we are naturally more enthusiastic about it. Thes attributes will often be used in the Front Matter:

```md
---
ease_factor: 3
interest_factor: 1.5
---

# My Favorite Topic

...
```


## Studies

Studying decks is available in the desktop application.

<img class="illustration" src="/the-notewriter/screen-deck-selection@2x.png" alt="Deck selection" width="400" />

:::tip[Study on the go]

[The support for remotes](../user-guide/remotes) is minimal. The long-term goals is to have various remotes to carry your notes with you on the go, with the possibility to study the flashcards on your phone.

:::

After selecting a deck, the study session for today begins:

<img class="illustration" src="/the-notewriter/screen-flashcard@2x.png" alt="Flashcard study" width="400" />

Flashcards are stored along other objects defined in a Markdown file. When studying, the application need to persist reviews. These study sessions are saved differently. They don't belong to the flashcard object but are a different kind of objects that work on top of the flashcard object itself.

For example, when reviewing a flashcard in the desktop application, an operation will be appended to a journal log. The journal log can be flushed using a button. A new pack file will be created containing all study operations. These pack files will be pushed or pulled like other pack files.

:::tip[Under the hood]

Operations are an example of [Conflict-free replicated data type](https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type) (CRDT) supporting concurrent updates (ex: studying the same flashcard on different devices). The algorithm automatically resolves any inconsistencies (the last write wins, which means if a flashcards is studied twice, the last review wins and the other is ignored).

TODO nt cat-file <operation-id>
:::
