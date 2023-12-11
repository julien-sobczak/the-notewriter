# Command `nt-reference`

`nt-reference` is an interactive CLI to generate reference files or notes already filled with metadata. Several sources are supported like Google Books, Zotero, and Wikipedia.

## Configuration

The command reads the same configuration file `.nt/config` as the command `nt`. To use this command, declare new sections like `[reference.ABC]` where each section represents one kind of generation.

For example:

```toml
[reference.books]
title = "A book"
manager = "google-books"
path = """references/books/{{index . "title" | slug}}.md"""
template = """
---
name: {{.Name}}
occupation: founding executive editor of Wired magazine
nationality: American
title: "Excellent Advice for Living: Wisdom I Wish I'd Known Earlier"
short_title: Excellent Advice for Living
date: 2023
num_pages: 224
isbn: "978-0593654521"
---

# {{index . "title"}}
"""
```

TODO complete with the de


## FAQ

### How to determine available attributes?

You can check the online documentation for the different providers:

| Provider | Implementation | Documentation |
|---|---|---|
| Zotero _(legacy)_ | All attributes returned by the API are exposed. Zotero defines different schemas for the different kinds of work | See project on [GitHub](https://github.com/zotero/translation-server) or check [Zotero Translation Server schemas](https://github.com/zotero/zotero-schema/blob/master/schema.json) |
| Google Books | The `volumeInfo` attribute is extracted and exposed. | See [official documentation](https://developers.google.com/books/docs/v1/using) |
| Wikipedia | [Infoboxes](https://en.wikipedia.org/wiki/Help:Infobox) are parsed to extract and parse attributes. It's not easy to find a list of possible attributes. | See [official documentation](https://www.mediawiki.org/wiki/API:Main_page) |

Another solution (even simpler to try), is to print all available attributes in your template:

```toml
[reference.book]
template = "{{ . | jsonPretty }}"
```

Once you know which attribute to use, edit the template and relaunch the command.


### What is the supported syntax for templates

The command uses the Go package `text/template` under the hood. Please read the [official documentation](https://pkg.go.dev/text/template). In addition, the command provides additional custom functions:

| Function | Description | Example |
| --- | --- | --- |
| `json` | Dump all attributes in compact JSON format | TODO add code + output |
| `jsonPretty` | Dump all attributes in a human-readable format | TODO add code + output |
| `yaml` | Dump all attributes in a human-readable format | TODO add code + output |
| `jq` | Support JQ expressions to extract values | TODO add code + output |
| `title` | Support JQ expressions to extract values | TODO add code + output |
| `slug` | Support JQ expressions to extract values | TODO add code + output |
