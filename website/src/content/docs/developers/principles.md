---
title: Principles
---

_The NoteWriter_ is a side project. Decisions are important when time is limited. Here are a few self-imposed constraints:

## Minimal Dependencies

All dependencies have a cost. The dependency needs to be updated, and eventually replaced when obsolete.

> The dependencies between components in a design should be in the direction of stability of the components. A package should only depend upon packages that are more stable than it is.
>
> — _Stable Dependencies Principle_ (DSP)

_The NoteWriter_ is expected to be stable, adding new features sparingly. Dependencies must be chosen wisely, finding the right balance between the gain (= the number of lines of code we don't need to write) versus the cost (= the time required to update/contribute/deprecate/replace a dependency).

## Minimal Integrations

As a codebase grows over time, the number of lines of code between the core logic and the various integrations (ex: support different source formats, export to different applications, support different storage solutions, etc) evolves differently. The core logic remains stable when the number of integrations continue to grow (= more line to write and maintain).

_The NoteWriter_ is not a general tool. It focus on developers working with Git and hosting their repositories on a platform like GitHub (= most of developers). The goal is to have a codebase where the core logic represents the majority of the lines of code.

## Minimal Features

_The NoteWriter_ was not created to replace other tools but to experiment with a new approach to note-taking. The goal is not to have a long list of features but to implement the core ones in a new way.

## Minimal Codebase

My goal is to keep the codebase small for a single person to maintain it over a very long time. Large codebase are slow to change and not all lines of code have the same value. A plugin system would be convenient but when storing notes for decades, any dependency will become a problem someday.
