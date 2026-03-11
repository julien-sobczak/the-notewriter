---
title: Configuration
---

Talking about configuration is boring. Starting with configuration is even more boring.

To get started, you don't need to edit a single configuration line. _The NoteWriter_ configuration is generated when running `nt init` and is present in the file `.nt/config.jsonnet`.

For now, your configuration must look like this:

```jsonnet
# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    tags: nt.DefaultTags,
    noteTypes: nt.DefaultNoteTypes,
}
```

The default configuration declares the default note types, attributes, and tags defined in the library file `.nt/nt.libsonnet` also generated during `nt init`. You can safely ignore this file for now.

As you will discover in this guide, _The NoteWriter_ is strongly opionated but there are too many good reasons to write notes that _The NoteWriter_ is also highly configurable. We will update this configuration in the following pages to take your notes to the next level. Editing the configuration is not fun but it will allows us to properly manage repositories with thousands of notes and do more with our notes.

:::tip[Why Jsonnet?]
[Jsonnet](https://jsonnet.org) is not the most popular format, for sure, but is more than helpful to keep the configuration minimal.
:::
