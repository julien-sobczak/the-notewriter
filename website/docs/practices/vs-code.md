---
sidebar_position: 2
---

# Editing Notes with VS Code

_The NoteWriter_ works with any editor. If you are using VS Code, this page contains my personal tips.


## Recommended Plugins

* [Foam](https://foambubble.github.io/foam/): Great list of plugins to work with Markdown files.
* [:emojisense:](https://marketplace.visualstudio.com/items?itemName=bierner.emojisense), by Matt Bierner: Enter emojis faster using autocompletion. Rely on `github/gemoji` (see [complete listing](https://github.com/github/gemoji/blob/master/db/emoji.json)).
* [Grammarly](https://www.grammarly.com/): Great to catch most typos and grammar errors if you accept to have "errors" with arcane termoinology. Here is the plugin configuration (cf `.vscode/settings.json`) to limit Grammarly on Markdown files:
    ```json
    {
        "grammarly.selectors": [
            {
                "language": "markdown",
                "scheme": "file"
            }
        ]
    }
    ```

