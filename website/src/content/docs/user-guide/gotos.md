---
title: Gotos
---

Markdown links can include in their title was is called a _Goto_. A Goto is a memorable name for a hard-to-remember URL.

The syntax must follow the convention `#go/{name}`.

```md title:go.md
## Note: Useful Links

* [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
* [Go Playground](https://go.dev/play/ "#go/go/playground") is useful to share snippets.
```

Gotos can be browsed directly from the terminal:

```shell
$ nt go go
# Open a new tab in your browser to https://go.dev/doc/

# Or
$ nt go go/playground
```

You can also use Gotos (more conveniently) from _The NoteWriter Desktop_ (no need to have a terminal open inside your notes repository).

:::tip

Use Go links for URLs that you must visit frequently (ex: internal tools at work).

:::

## Placeholders

Goto links support advanced placeholders in their URLs, allowing you to define variables that can be filled in dynamically.

### Placeholder Syntax

A placeholder is defined using `${variable}` inside the URL. There are several forms:

* **No default value**. The value must be entered when expanding the link. Ex: `https://github.com/${user}`
* **Predefined values**. A value must be selected from among options when expanding the link. Ex: `https://github.com/${user}/${repo}/${page:[issues,pulls,actions]}`
* **Suggestions**. A value must be entered but suggestions are offered when expanding the link. Ex: `https://github.com/${user}/${repo}/${page:[issues,pulls,...]}`

| Syntax                                 | Description                                 |
|---------------------------------------- |---------------------------------------------|
| `${var}`                               | Any value required                          |
| `${var:[a,b,c]}`                       | Must be one of `a`, `b`, or `c`                   |
| `${var:[a,b,...]}`                     | Suggestions: `a`, `b`, or any custom value      |

### Example

```md
[Open Issues](https://github.com/${user}/${repo}/${page:[issues,pulls,...]})
```

When rendered, the CLI (or the desktop application) will prompt for `user`, `repo`, and `page`, offering suggestions for `page` but allowing any value.

:::note[Useful Commands]

* Use `nt go <go-name>` to open the URL in your default browser. If the goto contains placeholders, the CLI will ask you to fill a form first.
:::
