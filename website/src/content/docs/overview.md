---
title: Overview
---

Let's discover **_The NoteWriter_ in less than 5 minutes**.

_The NoteWriter_ turns **Markdown files into objects**, providing an all-in-one note-taking solution where your reading notes, flashcards, reminders, tasks, bookmarks, journals and anything you decide to write down will live happily for a very long time.

## Just Markdown Files

To start using _The NoteWriter_, you simply need to write your notes in Markdown files using your favorite editor. You don't need _The NoteWriter_ to write your notes. You must only follow a few conventions. Here is an example:

```` .go.md
---
tags: go
---

# Go

## Common Mistakes

### Note: Using goroutines on loop iterator variables

`@source: https://go.dev/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables`

```go
// ❌ DON'T
for _, val := range values {
    go func() {
        fmt.Println(val)
    }()
}

// ✅ DO
for _, val := range values {
    go func(val interface{}) {
        fmt.Println(val)
    }(val)
}
```

By adding `val` as a parameter to the closure, `val` is evaluated at each iteration and placed on the stack for the goroutine.

## Note: Golang Conferences `#watching`

* GopherCon, USA, Late August `#remind-every-${year}-08`
* GopherCon Europe, Mid June

### Flashcard: Largest Conference

(Go) **Largest conference**?

---

**GopherCon** is the United States.
````

This small file illustrates (almost) all special conventions imposed by _The NoteWriter_.

A _file_ is basically a notebook where you group all notes relating to a similar topic. When _The NoteWriter_ will read the file, it will traverse the headings looking for headings matching known note types. For example, by default, _The NoteWriter_ recognizes headings starting with `Note:` or `Flashcard:`. _The NoteWriter_ will therefore extracts 3 notes: `Note: Using goroutines on loop iterator variables`, `Note: Golang Conferences` and `Flashcard: Largest Conference`.

A _note_ support metadata using _tags_ (defined as `` `#tag` ``) and _attributes_ (defined as `` `@name: value` ``). Tags are attributes with syntactic sugar. Attributes can also be defined using the Front Matter. And more interestingly, most attributes are inherited by default.

In addition to notes, _The NoteWriter_ will also extract additional objects. On the above example, the note `Flashcard: Largest Conference` will generate a flashcard to study using a spaced repetition system (SRS). The tag `` `#remind-every-${year}-08` `` will generate a recurring reminder "GopherCon, USA, Late August" that will pop up every August 1st.


## Summary

Writing your notes with _The NoteWriter_ means writing regular Markdown files locally. A few conventions exist but you would probably adopt similar syntaxes even without using _The NoteWriter_.

Let's try to add a few notes into _The NoteWriter_ but first, we need to discuss about the motivations behind one _more_ note-taking application.
