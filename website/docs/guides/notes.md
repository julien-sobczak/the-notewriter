---
sidebar_position: 1
---

# Notes

We use notes for so many things. _The NoteWriter_ supports different kinds.

When writing a new note, you must prefix your note's title by one the following kinds (ex: `## Flashcard: My Flashcard Title`).

## Kinds

### `Note` to clear your mind

Use `Note` for anything you don't want to forget.

    ## Note: Dark Mode

    The dark mode will allow users to switch the application in dark mode.

### `Flashcard` to remember

Use `Flashcard` for knowledge you want to remember.

    ## Flashcard:  The Zeigarnik Effect

    (Learning) What is the **Zeigarnik effect**?

    ---

    **The way unfinished tasks remain active in our mind**, intruding into our thoughts and our sleep until they are dealt with.

Check the guide ["Flashcards"](./flashcards.md) to learn more about the syntax and the algorithm.

### `Cheatsheet` to repeat faster

Use `Cheatsheet` for actions you don't want to relearn from scratch every time.

    ## Cheatsheet: Table-Driven Tests in Go

    ```go
    func TestFib(t *testing.T) {
            var tests = []struct {
                    name     string
                    n        int    // input
                    expected int    // output
            }{
                    {"1", 1, 1},
                    {"2", 2, 1},
            }
            for _, tt := range tests {
                    t.Run(tt.name, func(t *testing.T) {
                            actual := Fib(tt.n)
                            if actual != tt.expected {
                                    t.Errorf("Fib(%d): expected %d, actual %d", tt.n, tt.expected, actual)
                            }
                    })
            }
    }
    ```

### `Quote` to get inspired

Use `Quote` for inspiring quotes that resonate with you.

    ## Quote: Albert Einstein about Cluttered Desks

    If a cluttered desk is a sign of a cluttered mind, of what, then is an empty desk a sign?

### `Reference` to use later

Use `Reference` for information you may need in the future.

    ## Reference: Best Note-Taking Books

    * _How to Take Smart Notes_, by Sönke Ahrens
    * _Sönke Ahrens_, by Tiago Forte

### `TODO` to plan tasks

Use `TODO` for tasks you need to perform.

    ## TODO: Reading List

    * [x] _Tools Of Titans_, by Tim Ferris
    * [ ] _Stumbling on Happiness_, by Daniel Gilbert
    * [ ] _Bittersweet_, by Susan Cain

### `Journal` to track your day

Use `Journal` for tasks you need to perform.

    ## Journal: 2023-05-01

    * 📁 Complete documentation
      * [x] Add doc about the different kinds of note
      * [ ] Add doc about attributes and tags

### `Artwork` to get inspired

Use `Artwork` for artworks that resonate with you.

**TODO**



