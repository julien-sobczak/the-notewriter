# File Types Example

This example demonstrates how to use file types with processors in The NoteWriter.

## Configuration

In your `.nt/config.jsonnet`, add a `fileTypes` section:

```jsonnet
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes,
    fileTypes: {
        ReadingNotes: {
            name: "ReadingNotes",
            pattern: "(?i)^Reading:\\s*(.*)$",  // Matches "Reading: Book Title"
            processors: ["toc"],                 // Generates a table of contents
        },
        ProjectNotes: {
            name: "ProjectNotes", 
            pattern: "(?i)^Project:\\s*(.*)$",  // Matches "Project: Project Name"
            processors: ["toc"],
        },
    },
}
```

## Usage

Create a file with a title that matches a file type pattern:

```markdown
---
title: "Reading: Clean Code by Robert C. Martin"
tags: ["software", "craftsmanship"]
---

# Reading: Clean Code by Robert C. Martin

## Note: Chapter 1 - Clean Code

Clean code is code that has been taken care of. Someone has taken the time to keep it simple and orderly.

## Note: Chapter 2 - Meaningful Names

Use intention-revealing names. The name should tell you why it exists, what it does, and how it is used.

## Note: Chapter 3 - Functions

Functions should do one thing. They should do it well. They should do it only.
```

When this file is parsed, the `toc` processor will automatically run because the file title matches the `ReadingNotes` pattern, generating a table of contents note.

## Benefits

1. **Automatic Processing**: Files matching a type pattern automatically get their processors applied
2. **Consistency**: Enforce a consistent structure across similar files
3. **Extensibility**: Add custom processors and schemas for different file types
4. **Backward Compatible**: Works alongside existing note types

## Future Enhancements

The `schema` field in file types is a placeholder for future Markdown schema validation:

```jsonnet
fileTypes: {
    ReadingNotes: {
        name: "ReadingNotes",
        pattern: "(?i)^Reading:\\s*(.*)$",
        processors: ["toc"],
        schema: {
            frontMatter: {
                attributes: {
                    name: "rating",
                    optional: false,
                }
            },
            body: {
                // Future: Define expected heading structure
            }
        }
    }
}
```
