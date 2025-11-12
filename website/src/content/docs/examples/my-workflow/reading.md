---
title: My Reading Workflow
---

## While Reading

I read almost all my books using my Boox Note Air4C. I highlight every sentence that resonates (in case of doubt, I highlight and postpone the decision to keep it when importing my notes). I annotate most of my notes: a short note using my own words to rephrase the key idea, "+ flashcard" when I think a flashcard may be relevant, "+ merge" when the note must be merged with the previous highlight.

My reading notes are automatically synchronized to Google Drive using the Boox support.

## After Every Reading

### Add a new read book

I append the new read book into my reading list:

```md title=readings.md
## Reference: Read Books

`@hook: blog_library`

* 2025-08-20: _Revenge of the librarians_ ★★☆☆☆ `@author: Tom Gauld` `@isbn: 978-1770466166`
* 2025-09-03: _A Responsibility to Awe_ ★★★☆☆ `@author: Rebecca Elson` `@isbn: 978-1784106553`
* 2025-09-10: _The Tao of Pooh_ ★★★★★ 👍 `@author: Benjamin Hoff` `@isbn: 978-1405293785`
* 2025-09-22: _We, Programmers_ ★★☆☆☆ `@author: Robert C. Martin` `@isbn: 978-0135344262`
* 2025-09-26: _Beyond Vibe Coding_ ★★★★☆ `@author: Addy Osmani` `@isbn: 979-8341634756`
* 2025-10-06: _Winnie the Pooh_ ★★★★★ 👍 `@author: A. A. Milne` `@isbn: 978-0525444435`
* 2025-10-08: _The Te of Piglet_ ★★★★★ 👍 `@author: Benjamin Hoff` `@isbn: 978-0525934967`
* 2026-01-01: _The Let Them Theory_
```

I write a custom hook (present in `.nt/hooks/blog_library.py`) that commit a JSON document containing the note items to my blog repository (see [here](TODO)):

```json
TODO snippet
```

The file is then statically imported in MDX (my blog is written using [Astro](https://astro.build/)) and a custom React component render the list of all my readings. No need to edit my blog when adding a new read book. Using hooks make possible to automate some tasks so that recurring tasks only takes a few minutes.

### Initialize the Markdown file

I use the undocumented `nt-reference` binary to init a new book file. This command reads `config.jsonnet` to find the kinds of available reference notes:

```jsonnet title=config.jsonnet
{
    references: [
        // books
        {
            title: "A book",
            manager: "google-books",
            path: 'references/books/{{index . "title" | slug}}.md',
            template: |||
                ---
                title: "{{index . "title" | title}}{{ if index . "subtitle"}}:{{index . "subtitle" | title}}{{end}}"
                short_title: "{{index . "title" | title}}"
                name: {{index . "authors" | join ", "}}
                occupation: Unknown
                nationality: Unknown
                {{- if index . "publishedDate"}}
                date: "{{index . "publishedDate"}}"
                {{- end -}}
                {{- if index . "publisher"}}
                publisher: {{index . "publisher"}}
                {{- end -}}
                {{- if index . "pageCount"}}
                numPages: {{index . "pageCount"}}
                {{- end -}}
                {{- if index . "unknown"}}
                unknown: {{index . "unknown"}}
                {{- end -}}
                {{- if index . "industryIdentifiers"}}
                isbn: "{{index . "industryIdentifiers" | jq ". | first | .identifier"}}"
                {{- end }}
                ---

                # {{index . "title" | title}}
            |||
        },

        // persons
        {
            title: "A person",
            manager: "wikipedia",
            path: 'references/persons/{{index . "name" | slug}}.md',
            template: |||
                ---
                name: {{index . "name"}}
                occupation: {{if index . "occupation"}}{{index . "occupation"}}{{else}}Unknown{{end}}
                nationality: {{if index . "nationality"}}{{index . "nationality"}}{{else}}Unknown{{end}}
                {{- if index . "birth_date"}}
                birth_date: {{index . "birth_date"}}
                {{- end -}}
                {{- if index . "death_date"}}
                death_date: {{index . "death_date"}}
                {{- end -}}
                {{- if index . "known_for"}}
                known_for: "{{index . "known_for"}}"
                {{- end }}
                ---

                # {{index . "name"}}
            |||
        },
    ],
}
```

I will not go into details. The CLI asks the kind of reference, uses the attribute `manager` to determine the API to use to retrieve the metadata (ex: Google Books or Wikipedia), applies the Go `template` providing the retrieve metadata, and writes the result to the `path`, which is also a Go template:

```shell
$ nt-reference new
┃ Select reference category
┃ > A book
┃   A person

┃ Search query (title, ISBN, etc.)
┃ > Let them theory

┃ Select a result
┃ > The Let Them Theory: A Life-Changing Tool That Millions of People Can't Stop Talking About
┃   The Let Them Theory: A Synopsis of Mel Robbins Life-Changing Tool

┃ Review the generated reference (press Enter to continue)
┃ ---
┃ title: "The Let Them Theory:A Life-Changing Tool That Millions Of People Can't Stop Talking About"
┃ short_title: "The Let Them Theory"
┃ name: Mel Robbins
┃ occupation: Unknown
┃ nationality: Unknown
┃ date: "2024-12-24"
┃ publisher: Hay House Inc
┃ numPages: 337
┃ isbn: "9781401971366"
┃ ---
┃
┃ # The Let Them Theory
┃
┃ >

┃ Filename to save reference
┃ > references/books/the-let-them-theory.md
```

A new file has been created. I simply edited the attribute to fill the missing values and add some tags:

```md
---
title: "The Let Them Theory: A Life-Changing Tool That Millions Of People Can't Stop Talking About"
short_title: "The Let Them Theory"
name: Mel Robbins
occupation: author, lawyer, speaker, podcast host
nationality: American
date: "2024-12-24"
publisher: Hay House Inc
numPages: 337
isbn: "9781401971366"
tags: being
---

# The Let Them Theory
```

### Importing Boox notes

The next step is to retrieve the notes pushed by my Boox device. I installed the Google Drive desktop application. I also have a Python script `boox2notes.py` in my `notes` repository.

When run, the script lists reading notes file present in the Google Drive directory where Boox pushed its files. When a file is selected, the script converts the content from Boox-specific HTML to Markdown compatible with _The NoteWriter_ and asks for the target file (`references/book/the-let-them-theory.md`) on your example:

```md
# The Let Them Theory

## 10 How to Make Comparison Your Teacher

### ???

Put in the reps. It’s a phrase my buddy bestselling author Jeff Walker always
says, “Success is about putting in the reps.” What’s that mean? Simple: To be
successful, to lose weight, to write a book, or to become a YouTuber, you have
to show up every day and do the boring, irritating, and uncomfortable work.
You’ve got to put in the reps.

### ???

The famous quarterback Tom Brady recently said about success, “The truth is you
don’t have to be special. You just have to be what most people aren’t:
consistent, determined, and willing to work for it.”

> Merge + Quote

### ???

because you know you could do it too. And you are just mad that you didn’t
start doing it a long time ago. The fact is, inspiration is not enough to get
you motivated to do something.

This is why anger is important. This is why comparison can be one of your
greatest teachers, and I am willing to bet when it happens, it will likely be
someone you know who makes you mad.

> flashcard "Why comparison is useful according to Mel"?
...
```

The result is far from being perfect. The notes must be edited. The title `???` must be replaced, the note type sometimes be changed, tags and attributes added, and interesting links be added between notes. But it's far more convenient than working with the Boox generated file. Here the same section in HTML:

```html
<div style="padding-top: 1em; padding-bottom: 1em; border-top: 1px dotted lightgray;">
    <div
      style="font-size: 10pt; margin-bottom: 1em; padding-left: 8px; border-left: 5px solid #ed6c00; color: #888888;">
      2026-01-15 05:36:09
    </div>
    <div style="font-size: 12pt;">
        <span>Put in the reps. It’s a phrase my buddy bestselling author Jeff Walker always
        says, “Success is about putting in the reps.” What’s that mean? Simple: To be successful, to lose weight, to
        write a book, or to become a YouTuber, you have to show up every day and do the boring, irritating, and
        uncomfortable work. You’ve got to put in the reps.</span>
    </div>
  </div>
  <div style="padding-top: 1em; padding-bottom: 1em; border-top: 1px dotted lightgray;">
    <div
      style="font-size: 10pt; margin-bottom: 1em; padding-left: 8px; border-left: 5px solid #ed6c00; color: #888888;">
      2026-01-15 05:36:38
    </div>
    <div style="font-size: 12pt;">
        <span>The famous quarterback Tom Brady recently said about success, “The truth is
        you don’t have to be special. You just have to be what most people aren’t: consistent, determined, and willing
        to work for it.”</span>
    </div>
  </div>
  <div>
    <span style="color: #a9a9a9; padding-right: 0.5em; vertical-align: top; width: 2px;">注 | </span>
    <span style="padding-left: 0.5em; width: 10000px; vertical-align: top; color: #555555;">Merge + Quote</span>
  </div>
</div>
```

After editing, the result looks like:

```md
# The Let Them Theory

## Chapter 10: How to Make Comparison Your Teacher

### Note: Put in the Reps

Put in the reps. It’s a phrase my buddy bestselling author Jeff Walker always says,
“Success is about putting in the reps.” What’s that mean? Simple: To be successful,
to lose weight, to write a book, or to become a YouTuber, you have to show up
every day and do the boring, irritating, and uncomfortable work. You’ve got to
put in the reps. [...] The famous quarterback Tom Brady recently said about success,
“The truth is you don’t have to be special. You just have to be what most people
aren’t: consistent, determined, and willing to work for it.”

### Quote: Tom Brady About Success

`#success`

The truth is you don’t have to be special. You just have to be what
most people aren’t: consistent, determined, and willing to work for it.

### Note: How Anger Motivates You

`#productivity`

[B]ecause you know you could do it too. And you are just mad that you didn’t
start doing it a long time ago. The fact is, inspiration is not enough to get
you motivated to do something.

This is why anger is important. This is why comparison can be one of your
greatest teachers, and I am willing to bet when it happens, it will likely be
someone you know who makes you mad.

#### Flashcard: How Anger Motivates You

(Productivity) Why **comparison is useful** according to
Mel Robbins in _The Let Them Theory_?

---

Inspiration is not enough to get motivated. Anger is important.

**Comparison will help you get mad and start working**.
...
```

For every gathered note, I reevaluate the note:

* **Is it still relevant**? Sometimes I don't remember why I highlight a passage. I don't know why it resonates. I remove the note.
  * If I added a comment "+ flashcard", I reconsider if I really want the information in my long-term memory. Sometimes, I don't. Sometimes, I haven't anticipate the flashcard but when rereading the note, it becomes obvious and I create it.
* **Where does the note fit in my second brain**? The most insightful ideas deserves to be rewritten in my own words and lands in a more actionable notebook than just some reference notes. Two options:
  * The directory `skills/` where I have a Markdown file for every skill: `learning.md`, `psychology.md`, `parenting.md`, etc. These files contains everything I consider essential to master the skill. For example, after reading a productivity book, if I found an interesting principle, I create an additional note inside this file. Notes under `references/` are kept for references and are optional if I need to study a particular topic.
  * The directory `thoughts/` where I have a Markdown file for every general topic: `on-aging.md`, `on-doing.md`, etc. These files are the closest to what I consider my commonplace book.

## After Great Books

I do not write a review for every book I read. I prefer to write about book that I really want to recommend.

Therefore, for my great books, I add a new note of type `BookReview` at the end of the file in `references/book` and write my review along my reading notes.

I use a snippet in VS Code to quickly insert a new book review:

```json title=.vscode/nt.code-snippets
{
	"New BookReview": {
		"prefix": "bookreview",
		"scope": "markdown",
		"description": "Insert a book review template",
		"body": [
		"## BookReview: My Review",
		"",
		"`@read_date: ${CURRENT_YEAR}-${CURRENT_MONTH}-${CURRENT_DATE}`",
		"`@review_rating: ${1}` `@review_stars: ${2}` `@review_recommendation: ${3}`",
		"`@draft: true`",
		"",
		"**Summary**",
		"",
		"TODO"
		]
	}
}
```

I then simply enter `bookreview`, press Tab, complete the placeholders and I'm ready to write my review.

```md

```

A hook `blog_review` is defined on the definition of my custom type `BookReview`:

```jsonnet title=config.jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes + {
        // Reviews
        review_rating: {
            name: "review_rating",
            type: "integer",
            min: 0,
            max: 20,
        },
        review_stars: {
            name: "review_stars",
            type: "integer",
            min: 0,
            max: 5,
        },
        review_recommendation: {
            name: "review_recommendation",
            type: "integer",
            min: 0,
            max: 10,
        },
        read_date: {
            name: "read_date",
            type: "string",
            format: "yyyy-mm-dd",
        },
    },

    noteTypes: {
        BookReview: self.Note + {
            name: "BookReview",
            attributes: [
                {
                    // ISBN is required to identify the book
                    name: "isbn",
                    required: true,
                },
                {
                    name: "draft",
                    required: true,
                },
                {
                    // Rating of the book (0-20)
                    name: "review_rating",
                    required: true,
                },
                {
                    // Date when the book was read
                    name: "read_date",
                    required: true,
                },
                {
                    // Subject of the book (e.g., "Books", "Programming", "History")
                    name: "subject",
                    required: true,
                },
            ],
            hooks: ["blog_review"],
        },
    },
}
```

The custom hook `blog_review` (present in `.nt/hooks/blog_review.py`) uses the GitHub GraphQL API to commit a new file in my blog repository (similarly to the previous hook `blog_library`). The result:

```md
TODO
```

TODO screenshot juliensobczak.com


