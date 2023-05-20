---
sidebar_position: 6
---

# Links

Markdown links can include in their title was is called a _Go link_, that is a memorable name for a hard-to-remember URL.

The syntax must follow the convention `#go/{name}`.

```md title:go.md
## Note: Useful Links

* [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.
* [Go Playground](https://go.dev/play/ "#go/go/playground") is useful to share snippets.
```

Go links can be browse directly from the terminal:

```shell
$ nt go go
# Open a new tab in your browser to https://go.dev/doc/

# Or
$ nt go go/playground
```

You can also use Go links (more conveniently) since _The NoteWriter Desktop_ (no need to have a terminal open inside your notes repository).


:::tip

Use Go links for URL that you must visit frequently (ex: internal tools at work).

:::

