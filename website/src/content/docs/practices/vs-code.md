---
title: Editing Notes with VS Code
---


_The NoteWriter_ works with any editor. If you are using VS Code, this page contains my personal tips.


## Recommended Snippets

Use VS Code snippets to quickly write a new note without forgetting some required attributes.

For example, I have a snippet to generate a new book review:

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


## Recommended Plugins

* A plugin to **improve Markdown editing**. For example, the [project Foam](https://foambubble.github.io/foam/) provides a great list of [plugins](https://foambubble.github.io/foam/user/getting-started/recommended-extensions) to work with Markdown files.
* A plugin to **insert Emojis easily**. For example, I use [:emojisense:](https://marketplace.visualstudio.com/items?itemName=bierner.emojisense), by Matt Bierner to enter emojis faster using autocompletion. The plugin relies on `github/gemoji` (see [complete listing](https://github.com/github/gemoji/blob/master/db/emoji.json)). Using emojis is useful during journaling, to make flashcards more visual, to categorize note annotations, etc.
* A plugin to **work with AI agents**. For example, I currently use GitHub Copilot. I always edit my notes manually but sometimes use AI tools to refactor my notes (ex: add an explicit attribute `@slug` on my flashcards, add the publication year on my reading notes, etc.).

:::tip

_How to enable these extensions on specific workspaces only?_

You may not want to run these extensions on every workspace on your laptop. [VS Code supports disabling it globally and enabling it specifically on a few workspaces](https://github.com/microsoft/vscode/issues/15611).

:::

