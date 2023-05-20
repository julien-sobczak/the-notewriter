---
sidebar_position: 2
---

# Editing Notes with VS Code

_The NoteWriter_ works with any editor. If you are using VS Code, this page contains my personal tips.


## Recommended Snippets

**TODO** complete


## Recommended Plugins

* [Foam](https://foambubble.github.io/foam/): Great list of [plugins](https://foambubble.github.io/foam/user/getting-started/recommended-extensions) to work with Markdown files.
* [:emojisense:](https://marketplace.visualstudio.com/items?itemName=bierner.emojisense), by Matt Bierner: Enter emojis faster using autocompletion. Rely on `github/gemoji` (see [complete listing](https://github.com/github/gemoji/blob/master/db/emoji.json)).
* [Grammarly](https://www.grammarly.com/): Great to catch most typos and grammar errors if you accept to have "errors" with arcane terminology. Here is the plugin configuration (cf `.vscode/settings.json`) to limit Grammarly on Markdown files:
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

:::tip

_How to enable the extension on specific workspaces only?_

You may not want to run Grammarly on every workspace on your laptop. [VS Code supports disabling it globally and enabling it specifically on a few workspaces](https://github.com/microsoft/vscode/issues/15611).

:::

