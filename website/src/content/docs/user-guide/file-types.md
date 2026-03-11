---
title: File Types
---

## Defining File Types

Until now, we have only use files as container of notes but files may also be typed. Unlike notes, there are no default file types declared in `nt.libsonnet` but custom ones may be declared in `config.jsonnet`:

```jsonnet
local nt = import 'nt.libsonnet';

{
	fileTypes: nt.DefaultFileTypes + { // inherit future file types
		Reading: {
			name: "Reading",
		},
	},
}
```

### Attributes

Attributes may be defined like for note types:

```jsonnet
local nt = import 'nt.libsonnet';
{
    attributes: {
        isbn: {
            name: "isbn",
            type: "string",
        },
	},
    fileTypes: {
        Reading: {
            name: "Reading",
            attributes: [
                {
                    name: 'isbn',
                    required: true,
                },
            ],
        }
    }
}
```

The attribute `isbn` is now required in the Front Matter.

### Schema

A schema could also defined to enforce some structure on the file. Since there is no standard to define a Markdown schema, we use a minimal version. Here is an example:

```jsonnet
local nt = import 'nt.libsonnet';
local makeHeading = nt.Schema.makeHeading;
{
    fileTypes: nt.DefaultFileTypes + {
        "Reading": {
            name: "Reading",
            attributes: [
                {
                    name: "isbn",
                    required: true,
                },
            ],
            schema: {
                body: makeHeading(children=[
                    makeHeading(
                        match="^Notes$",
                        allowMultiple=false,
                        children=[
                            makeHeading(
                                matchType="^(Note|Quote|Flashcard)$",
                                required=true,
                                allowMultiple=true),
                    ]),
                    makeHeading(
                        matchType="^BookReview$",
                        required=false,
                        allowMultiple=false),
                ]),
            },
        }
    },
}
```

The schema expects a Markdown heading `Notes` containing notes of type `Note`, `Quote`, or `Flashcard`, following by a note of type `BookReview`. For example, the following file matches this schema:

```md
# Reading: Some Book

## Notes

### Note: Some interesting sentence

Some sentence that was highlighted from the books.

### Quote: An inspiring quote

A memorable quote.

## BookReview: My Review

A great book!
```

:::tip

**Use schemas sparingly**. Imposing too much structure on notes can make your notes difficult to evolve. Schemas are interesting when reusing the same desk template (not covered for now) between a large number of files.

:::

### Processors

Processors transform file extracted objects during parsing. There are only two available processors:

- `master`: Processes [Master notes](./note-types#master-notes) defined in a file. This processor is automatically run when a note of type `Master` is detected in a file.
- `toc`: Generates a new note "Table of Contents" with Markdown links to the different notes present in the file.

Example (`"toc"`):

```jsonnet title=config.jsonnet
{
	fileTypes: {
		Reading: {
			name: "Reading",
			processors: ["toc"],
		},
	},
}
```

The above configuration will automatically generate a Table of Contents for all Markdown files with the type `Reading`.

The preprocessor `toc` may also triggered automatically by using the special tag `#toc` in a file:

```md
---
tags: "toc"
---
# TOC

## Introduction

This is a demonstration of the TOC feature.
This section should not appear in the TOC because it has no child notes.

## Note: Main Concept

This is a main concept note. It should appear in the TOC with a wikilink.

### Note: Sub-concept

This is a sub-concept note. It should appear indented under the main concept.

### Flashcard: Definition

What is the main concept?

---

The main concept is the primary idea being discussed.

## Resources

This untyped section contains resources and should appear
in TOC because it has child notes.

### Quote: Famous Quote

> Knowledge is power.

-- Francis Bacon

### Note: Resource Note

This is a resource note under the resources section.

## Empty Section

This section has no child notes, so it should not appear in the TOC.
```

The generated note is similar to having written this note:

```md
## Note: Table of Content

* [[#Note: Main Concept]]
  * [[#Note: Sub-concept]]
  * [[#Flashcard: Definition]]
* Resources
  * [[#Quote: Famous Quote]]
  * [[#Note: Resource Note]]
```
