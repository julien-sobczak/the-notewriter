---
title: My Daily Workflow
---

## Every Day (and Night)

I use notes to limit interruptions. When I think about something I need to do, read, check, buy or when I have a new idea about a project, I write a short Google Keep note (usually a few words or just a link). The motivation is to get everything out of my head as soon as possible.

Using an application such as Google Keep is great because the application is easily accessible (I have the Android application on the home screen and have a pinned tab in my browser). The notes are automatically synchronized between devices. These kinds of notes are best managed outside _The NoteWriter_ (and is one of the core principles to keep the codebase minimal).

## Every Week

Every week, on Friday, I devote one hour processing all these notes in bulk. For every note, and inspired by the Getting Things Done (GTD) methodology, I asked myself a few questions:

1. **Is it relevant?** Several notes are just deleted without any action. What seems interesting is no longer so valuable just a few days later.
2. **Is it actionable?** When?
  * Now? If the note takes less than 2 minutes, I do it immmediately. For example, when finding a new book to read, I create a note with the title. During my review, I check the book online (the reviews, the table of contents, etc.) and add the new book to my reading list (a note using a custom type `ReadingList` in `todo/maybe/read.md`). If the note is a link to a YouTube video, I quickly browse the content and add it my "Watch Later" list.
  * Later? If the note is useful for a in-progress, planned, or imagined project, I put the note inside the project file (I keep a directory per project with an `index.md` inside). If the note is a quote, I move it to my commonplace book (in the directory `references/persons` in a file named about the author).

Example of common daily notes:

* A quote => move to `references/persons/<name>.md` + add tags, attributes like `@source`, etc.
* An idea => move to `projects/<project>/index.md` or create a new directory into `projects/`. It can be the idea for a new blog post or just an idea to integrate inside a future blog posts.
* A book => add to `todo/maybe/read.md` + add a rating to evaluate my interest for it. Same for video (`todo/maybe/watch.md`).
* A task => Do it now or add to the note `Todo: Backlog` inside the corresponding `projects/`.
* A link => Browse the page and determine the next action (ex: add into `todo/maybe/buy.md` for items to buy someday).
