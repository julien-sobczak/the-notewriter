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
template = """---
title: "{{index . "title" | title}}{{ if index . "subtitle"}}:{{index . "subtitle" | title}}{{end}}"
short_title: "{{index . "title" | title}}"
name: {{index . "authors" | join ", "}}
occupation: Unknown
nationality: Unknown
{{- if index . "pageCount"}}
numPages: {{index . "pageCount"}}
{{- end -}}
{{- if index . "industryIdentifiers"}}
isbn: "{{index . "industryIdentifiers" | jq ". | first | .identifier"}}"
{{- end }}
---

# {{index . "title" | title}}
"""
```

## Usage

```shell
$ nt-reference new
```

The CLI is interactive. No option or argument is expected. Simply run it and answer the different questions until your file or note is generated.

<!-- TODO Add section ## Example using asciinema -->

## FAQ

### How to determine available attributes?

You can check the online documentation for the different providers:

| Provider | Implementation | Documentation |
|---|---|---|
| Wikipedia | [Infoboxes](https://en.wikipedia.org/wiki/Help:Infobox) are parsed to extract and parse attributes. It's not easy to find a list of possible attributes. | See [official documentation](https://www.mediawiki.org/wiki/API:Main_page) |
| Google Books | The `volumeInfo` attribute is extracted and exposed. | See [official documentation](https://developers.google.com/books/docs/v1/using) |
| Zotero _(legacy)_ | All attributes returned by the API are exposed. Note that Zotero defines different schemas for the different kinds of work. | See project on [GitHub](https://github.com/zotero/translation-server) or check [Zotero Translation Server schemas](https://github.com/zotero/zotero-schema/blob/master/schema.json) |

Another solution (even simpler to try), is to print all available attributes in your template:

```toml
[reference.book]
template = "{{ . | jsonPretty }}"
```

Once you know which attribute to use, edit the template and relaunch the command.


### What is the supported syntax for templates?

The command uses the Go package `text/template` under the hood. Please read the [official documentation](https://pkg.go.dev/text/template). In addition, the command provides additional custom functions:

<table>

<thead>
<tr>
<th>Function</th>
<th>Description</th>
<th>Example</th>
</tr>
</thead>

<tbody>

<!-- json -->
<tr>
<td><code>json<code></td>
<td>Dump all attributes in compact JSON format</td>
<td>
Attributes:

```json
{
    "title": "Meditations",
    "authors": ["Marcus Aurelius"]
}
```

Usage:

```
{{json .}}
```

Output:

```json
{ "title": "Meditations", "authors": ["Marcus Aurelius"] }
```
</td>
</tr>

<!-- jsonPretty -->
<tr>
<td><code>jsonPretty<code></td>
<td>Dump all attributes in a human-readable format</td>
<td>
Attributes:

```json
{
    "title": "Meditations",
    "authors": ["Marcus Aurelius"]
}
```

Usage:

```
{{jsonPretty .}}
```

Output:

```json
{
    "title": "Meditations",
    "authors": [
        "Marcus Aurelius"
    ]
}
```
</td>
</tr>

<!-- yaml -->
<tr>
<td><code>yaml<code></td>
<td>Dump all attributes in a human-readable format</td>
<td>
Attributes:

```json
{
    "title": "Meditations",
    "authors": ["Marcus Aurelius"]
}
```

Usage:

```
{{yaml .}}
```

Output:

```yaml
title: "Meditations"
authors:
  - "Marcus Aurelius"
```
</td>
</tr>

<!-- jq -->
<tr>
<td><code>jq<code></td>
<td>Support jq expressions to extract values</td>
<td>
Attributes:

```json
{
    "title": "Meditations",
    "authors": ["Marcus Aurelius"]
}
```

Usage:

```
{{jq '. | .title' .}}
```

Output:

```
Meditations
```
</td>
</tr>

<!-- title -->
<tr>
<td><code>title<code></td>
<td>Convert using common book title case</td>
<td>
Attributes:

```json
{
    "title": "How to take smart notes",
    "authors": ["Sönke Ahrens"]
}
```

Usage:

```
{{index . "title" | title}}
```

Output:

```
How to Take Smart Notes
```
</td>
</tr>

<!-- slug -->
<tr>
<td><code>slug<code></td>
<td>Convert to a URL-compliant slug</td>
<td>
Attributes:

```json
{
    "title": "How to Take Smart Notes",
    "authors": ["Sönke Ahrens"]
}
```

Usage:

```
{{index . "title" | slug}}
```

Output:

```
how-to-take-smart-notes
```
</td>
</tr>

<!--
<tr>
<td><code>xxx<code></td>
<td>Do XXX</td>
<td>
Attributes:

```json
{
    "title": "Meditations",
    "authors": ["Marcus Aurelius"]
}
```

Usage:

```
{{xxx .}}
```

Output:

```
XXX
```
</td>
</tr>
-->

</tbody>
</table>


