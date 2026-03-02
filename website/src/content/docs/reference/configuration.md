---
title: Configuration
---

The NoteWriter is configured using a [Jsonnet](https://jsonnet.org/) file
located at `.nt/config.jsonnet`. Jsonnet is a data templating language that
is a superset of JSON, adding support for comments, variables, and functions.

## Configuration File

The following example presents all available configuration options with
explanations for every field:

```jsonnet
// Import the standard library bundled with The NoteWriter.
// It provides DefaultAttributes, DefaultNoteTypes, and LintRules.
local nt = import 'nt.libsonnet';

// Variables can be defined at the top for reuse across the config.
local srsAlgorithmSettings = {
  // Controls how fast review intervals grow after a correct answer.
  easeFactor: 2.5,
};

{
  // A unique identifier for this notes collection.
  // Important when opening different repositories
  // in the same instance of The NoteWriter Desktop.
  slug: 'my-notes',

  // Attribute definitions used in notes' front matter or inline.
  // Extend the built-in defaults with your own custom attributes.
  attributes: nt.DefaultAttributes + {

    // Example: author name (useful on quotes and books).
    name: {
      // The attribute key used when declaring the attribute (e.g., `name: Alice`).
      name: "name",
      // Alternative keys that resolve to the same attribute.
      aliases: ["author"],
      // Data type: "string", "bool", "integer", "float", "date".
      type: "string",
      // Format hint: "markdown", "isbn", "yyyy-mm-dd", etc.
      format: "markdown",
      // When true, child notes inherit the value from their parent.
      inherit: true,
    },

    // Example: book ISBN stored in the file front matter.
    isbn: {
      name: "isbn",
      type: "string",
      format: "isbn",
      inherit: true,
    },

    // Example: date a book was finished reading.
    read_date: {
      name: "read_date",
      // Use "string" instead of "date" to avoid timestamp coercion.
      type: "string",
      format: "yyyy-mm-dd",
      inherit: true,
      // The desktop application will remind you of this event every year.
      memory: true,
    },

    // Example: attribute with a fixed set of allowed values.
    rating: {
      name: "rating",
      type: "integer",
      // Minimum and maximum accepted values.
      min: 1,
      max: 5,
      // Value assumed when the attribute is absent.
      defaultValue: 3,
      // Map shorthand notations to their canonical integer values.
      shorthands: {
        "★":     1,
        "★★":    2,
        "★★★":   3,
        "★★★★":  4,
        "★★★★★": 5,
      },
    },

    // Example: boolean flag to mark unpublished content.
    draft: {
      name: "draft",
      type: "bool",
      // When false, child notes do NOT inherit this attribute.
      inherit: false,
    },

  },

  // Note type definitions.
  // Extend the built-in types or override their defaults.
  noteTypes: nt.DefaultNoteTypes + {

    // Override a built-in type to enforce specific attributes.
    Quote: nt.DefaultNoteTypes.Quote + {
      attributes: [
        {
          name: "name",
          // Linter reports an error when this attribute is missing.
          required: true,
        },
        {
          name: "rating",
          // Attribute is accepted but not required.
          required: false,
          // When true, the attribute appears on individual list
          // items, not on the note itself.
          inline: false,
        },
      ],
    },

    // Define a simple custom note type (no extra configuration).
    Reference: nt.DefaultNoteTypes.Note + {
      name: "Reference",
    },

    // Custom type with multiple required attributes.
    BookReview: nt.DefaultNoteTypes.Note + {
      name: "BookReview",
      attributes: [
        { name: "isbn",      required: true },
        { name: "rating",    required: true },
        { name: "read_date", required: true },
        { name: "draft",     required: true },
      ],
    },

    // Custom type that uses a built-in content processor.
    ReadingList: nt.DefaultNoteTypes.Note + {
      name: "ReadingList",
      attributes: [
        {
          name: "rating",
          // Attribute is allowed but not required.
          required: false,
          inline: false,
        },
      ],
      // Built-in processors that transform note content:
      //   "list-items"          - extract list items as objects
      //   "quote-rewriter"      - normalize blockquote formatting
      //   "date-extractor"      - parse dates from content
      //   "flashcard-extractor" - extract flashcard Q&A pairs
      //   "generator"           - run an external script
      processors: ["list-items"],
    },

  },

  // Linter configuration.
  linter: {
    // List of active lint rules.
    rules: [
      // Reject notes with an empty title.
      nt.LintRules.NoEmptyTitle(),
      // Reject two notes sharing the same title in the collection.
      nt.LintRules.NoDuplicateNoteTitle(),
      // Reject two notes sharing the same slug.
      nt.LintRules.NoDuplicateSlug(),
      // Reject media files that no note references.
      nt.LintRules.NoDanglingMedia(),
      // Reject wikilinks pointing to non-existent notes.
      nt.LintRules.NoDeadWikilink(),
      // Reject wikilinks that include a file extension.
      nt.LintRules.NoExtensionWikilink(),
      // Reject wikilinks that match more than one note.
      nt.LintRules.NoAmbiguousWikilink(),
      // Reject flashcards not associated with a parent note.
      nt.LintRules.NoOrphanFlashcard(),
      // Reject flashcards that are missing a unique slug.
      nt.LintRules.RequireFlashcardSlug(),
      // Reject notes that have an implicit slug on a flashcard.
      nt.LintRules.NoImplicitSlugOnFlashcard(),
      // Require every note to carry at least one of these tags.
      nt.LintRules.RequireTag(["topic-a", "topic-b"]),
      // Require notes matching a query to carry at least one tag
      // from the provided list.
      // First arg: query used to select target notes.
      // Second arg: list of accepted tags (one required).
      nt.LintRules.RequireTagIf("type:Quote", [
        "insight",
        "motivation",
      ]),
      // Require at least N blank lines between consecutive notes.
      nt.LintRules.MinLinesBetweenNotes(1),
      // Require at most N blank lines between consecutive notes.
      nt.LintRules.MaxLinesBetweenNotes(3),
    ],
  },

  // Named queries saved in the desktop application to avoid
  // re-entering the same searches.
  queries: {

    // Show a random quote on startup.
    dailyQuote: {
      // Human-readable title shown in the UI.
      title: "Daily Quote",
      // Query string (same syntax as `nt search`).
      query: "type:Quote",
      // Tags associate this query with specific features:
      //   "daily-quote" - surfaced by the daily-quote feature
      //   "inspiration" - used by the inspiration feature
      //   "zen"         - used in Zen mode
      //   "project"     - used for project management views
      //   "task"        - used for task management views
      tags: ["daily-quote"],
    },

    // Show all notes tagged as favorites.
    favorites: {
      title: "Favorites",
      query: "#favorite",
      tags: ["inspiration"],
    },

  },

  // Journal configurations. Each entry defines one journal.
  journals: [
    {
      // Display name shown when selecting a journal.
      name: 'My Diary',
      // File path template for new journal entries.
      // Supported custom functions: year, month, day, date.
      path: 'journal/{{ year }}/{{year}}-{{month}}-{{day}}.md',
      // Default heading inserted when a new entry is created.
      defaultContent: 'Journal: {{year}}-{{month}}-{{day}}',
      // Routines are recurring templates appended to a journal entry.
      routines: [
        {
          // Name shown when choosing a routine to insert.
          name: 'Morning Routine',
          // Multiline Go template using Jsonnet text blocks (|||).
          // Supported custom functions:
          //   {{ input }}                     - free-text input prompt
          //   {{ affirmation "<wikilink>" }}  - random affirmation
          //   {{ prompt "<wikilink>" }}       - random writing prompt
          template: |||
            # 🎯 My BIG thing for today

            {{ input }}

            # 😘 Gratitude Journal

            * {{ input }}
            * {{ input }}
            * {{ input }}
          |||,
        },
      ],
    },
  ],

  // Flashcard deck definitions for spaced-repetition study (SRS).
  decks: [
    {
      // Display name of the deck.
      name: "Skills",
      // Optional query restricting which flashcards belong to
      // this deck. Decks are evaluated in order; each flashcard
      // is assigned to the first matching deck.
      query: "path:resources/skills",
      // Maximum number of new cards introduced per study session.
      newFlashcardsPerDay: 10,
      // SRS algorithm tuning parameters (see variable at top).
      algorithmSettings: srsAlgorithmSettings,
    },
    {
      name: "General",
      // No query: this deck catches all remaining flashcards.
      newFlashcardsPerDay: 10,
      algorithmSettings: srsAlgorithmSettings,
    },
  ],

  // Book export definitions (optional).
  // Each entry describes one generated book.
  books: [
    {
      // Title of the book.
      title: "My Collected Quotes",
      // Optional subtitle.
      subtitle: "A Personal Anthology",
      // Output formats: "markdown", "epub", "pdf".
      format: ["markdown"],
      // Author list.
      author: ["Your Name"],
      // Language code (IETF BCP 47).
      language: "en",
      // Each chapter is populated by a query.
      chapters: [
        {
          // Chapter heading.
          title: "On Learning",
          // Notes returned by this query form the chapter content.
          query: "path:thoughts/on-learning type:Quote",
          // Optional chapter icon (path relative to the repo root).
          icon: "medias/on-learning.png",
        },
      ],
    },
  ],

}
```
