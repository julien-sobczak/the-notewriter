---
title: Note Types
---

## Extending Note Types

Until now, we have only use one of the default note types declared in `nt.libsonnet` and declared in `config.jsonnet`:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes,
}
```

Declaring custom note types is similar to declaring custom attributes:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes + {
        Idea: {
            name: "Idea"
        }
    }
}
```

That's enough to support a new kind of note (but we can do a lot more with custom note type!):

```md[my-project.md]
# My Project

## Ideas

### Idea: Do Something Unique

This idea will be extracted as a note by _The NoteWriter_.
```

## Note Attributes

Attributes can be used on any note but some attributes are particularly relevant on some note types. When declaring a note type, you could declare a list of attributes:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes + {

        Quote: nt.DefaultNoteTypes.Quote + {
            // Override to enforce a few attributes
            attributes: [
                {
                    name: "name",
                    required: true,
                },
                {
                    name: "nationality",
                    required: true,
                },
            ],
        }
    }
}
```

When declaring a new note of type `Quote`, the attribute `name` and `nationality` must be specified (or inherited). For example:

```md[.marcus-aurelius.md]
---
name: Marcus Aurelius
nationality: Roman
---

## Quote: Memento Mori

You could leave life right now.
Let that determine what you do and say and think.
```

This file is valid as the note "Memento Mori" inherits from attributes `name` and `nationality` defined in the Front Matter.

Required attributes are optional when a default value is defined on the attribute. For example:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes + {
        draft: {
            name: "draft",
            type: "bool",
            defaultValue: "true",
            inherit: false,
        },
    }
    noteTypes: nt.DefaultNoteTypes + {
        BookReview: self.Note + {
            name: "BookReview",
            attributes: [
                {
                    name: "draft",
                    required: true,
                },
            ]
        }
    }
}
```

When writing a new book review, the attribute `draft` can be omitted. The value `true` applies automatically. To publish the review (for example using a [hook](./hooks)), the attribute `draft` must be explicitely defined to `false`. No draft will be published before being ready.


## List Notes

Lists are inevitable: reading lists, to-do lists, achievements, etc.

You can use Markdown lists in any of your notes but _The NoteWriter_ supports a predefined type `List`:

```md
## List: Read Books

* _1984_ by George Orwell `#dystopia` `#politics`
* _To Kill a Mockingbird_ by Harper Lee `#justice` `#racism`
* _Pride and Prejudice_ by Jane Austen `#romance` `#society`
* _The Great Gatsby_ by F. Scott Fitzgerald `#american-dream` `#tragedy`
* _Moby-Dick_ by Herman Melville `#adventure` `#obsession`
* _War and Peace_ by Leo Tolstoy `#history` `#war`
* _The Catcher in the Rye_ by J.D. Salinger `#coming-of-age` `#alienation`
* _Brave New World_ by Aldous Huxley `#dystopia` `#science-fiction`
```

When using the type `List`, the note will be displayed differently in the desktop application to let you filter and search inside items.

## Custom List Types

To create custom types that behave similarly, you simply need to extend the type `List` to reuse the same processor:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes + {
       ReadingList: nt.DefaultNoteTypes.List + {
            name: "ReadingList",
            // automatically inherit processor "list-items"
        },
        // Same as
        AchievementList: nt.DefaultNoteTypes.Note + {
            name: "AchievementList",
            processors: ["list-items"],
        }
    }
}
```

Processors are additional logic to execute on a parsed note. Another example is the standard type `Flashcard`:

```jsonnet title=config.jsonnet
{
    noteTypes: {
        Flashcard: self.Note + {
            name: "Flashcard",
            processors: ["flashcard-extractor"],
        },
    }
}
```

The proprocessor `flashcard-extractor` is responsible to extract the front/back cards. You can create your own custom flashcard types by reusing the same processor.
