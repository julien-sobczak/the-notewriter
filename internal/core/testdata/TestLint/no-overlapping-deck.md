---
slug: no-overlapping-deck
---

# `no-overlapping-deck`

## Flashcard: Matching only Deck1

`@slug: matching-deck1`
`#deck1`

This flashcard matches only deck1 [{c1::valid}].

## Flashcard: Matching only Deck2

`@slug: matching-deck2`
`#deck2`

This flashcard matches only deck2 [{c1::valid}].

## Flashcard: Matching both decks

`@slug: overlapping-flashcard`
`#deck1`
`#deck2`

This flashcard matches both decks [{c1::violation}].

## Flashcard: Suspended flashcard

`@slug: suspended-flashcard`
`#suspended`
`#deck1`
`#deck2`

This flashcard is suspended and should be ignored [{c1::ignored}].
