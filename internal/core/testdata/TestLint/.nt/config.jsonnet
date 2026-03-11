local nt = import 'nt.libsonnet';

local makeHeading = nt.Schema.makeHeading;

{
  core: {
    medias: {
      command: 'random',
    },
  },

  attributes: {
    // Add a name attribute that is required on type Quote
    name: {
      name: 'name',
      aliases: ['author'],
      type: 'string',
      inherit: true,
    },
    // Declare a pattern for the attribute isbn
    isbn: {
      name: 'isbn',
      type: 'string',
      pattern: '^([0-9-]{10}|[0-9]{3}-[0-9]{10})$',
      inherit: true,
    },
    // Declare a date attribute
    birth_date: {
      name: 'birth_date',
      type: 'date',
      format: 'yyyy-mm-dd',
      inherit: true,
    },
    // Declare an integer attribute to promote in journal entries
    steps: {
      name: 'steps',
      description: 'Number of steps per day',
      type: 'integer',
      min: 0,
      max: 100000,
    },
  },

  noteTypes: nt.DefaultNoteTypes {
    Quote: nt.DefaultNoteTypes.Quote {
      attributes: [
        {
          name: 'name',
          required: true,
        },
      ],
    },
    Review: nt.DefaultNoteTypes.Note {
      name: 'Review',
      attributes: [
        {
          name: 'rating',
        },
      ],
    },
    Journal: nt.DefaultNoteTypes.Journal {
      attributes: [
        {
          name: 'date',
          required: true,
        },
        {
          name: 'steps',
          promoteInline: true,
        },
      ],
    },
    // Project-specific note types
    Synopsis: nt.DefaultNoteTypes.Note {
      name: 'Synopsis',
    },
    Idea: nt.DefaultNoteTypes.Note {
      name: 'Idea',
    },
    Task: nt.DefaultNoteTypes.Note {
      name: 'Task',
    },
  },

  fileTypes: nt.DefaultFileTypes {
    Reading: {
      name: 'Reading',
      attributes: [
        {
          name: 'read_date',
          required: true,
        },
        {
          name: 'title',
          required: true,
        },
        {
          name: 'short_title',
        },
        {
          name: 'isbn',
          required: true,
        },
      ],
      schema: {
        body: makeHeading(children=[
          makeHeading(match='^Notes$', allowMultiple=false, children=[
            makeHeading(matchType='^(Note|Quote|Flashcard)$', required=true, allowMultiple=true),
          ]),
          makeHeading(matchType='^Review$', required=false, allowMultiple=false),
        ]),
      },
    },
    Project: {
      name: 'Project',
      attributes: [
        {
          name: 'source',  // Must be a GitHub repo URL
          required: true,
        },
      ],
      schema: {
        body: makeHeading(children=[
          makeHeading(
            matchType='^Synopsis$',
            allowMultiple=false
          ),
          makeHeading(
            matchType='^List$',
            required=false,
            allowMultiple=true
          ),
          makeHeading(
            match='^Tasks$',
            allowMultiple=false,
            children=[
              makeHeading(
                matchType='^(Master|Task)$',
                required=true,
                allowMultiple=true
              ),
            ]
          ),
          makeHeading(
            match='^Ideas$',
            allowMultiple=false,
            children=[
              makeHeading(
                matchType='^(Master|Idea)$',
                required=true,
                allowMultiple=true
              ),
            ]
          ),
        ]),
      },
    },
  },

  linter: {
    rules: [
      nt.LintRules.NoEmptyTitle(),
      nt.LintRules.NoDuplicateNoteTitle(),
      nt.LintRules.NoDuplicateSlug(),
      nt.LintRules.MinLinesBetweenNotes(2),
      nt.LintRules.MaxLinesBetweenNotes(2),
      nt.LintRules.NoDanglingMedia(),
      nt.LintRules.NoDeadWikilink(),
      nt.LintRules.NoExtensionWikilink(),
      nt.LintRules.NoAmbiguousWikilink(),
    ],
  },

  decks: [
    {
      name: 'Deck1',
      query: '#deck1',
      boostFactor: 100,
      newFlashcardsPerDay: 10,
      maxFlashcardsPerDay: 50,
      algorithm: 'nt-0',
    },
    {
      name: 'Deck2',
      query: '#deck2',
      boostFactor: 100,
      newFlashcardsPerDay: 10,
      maxFlashcardsPerDay: 50,
      algorithm: 'nt-0',
    },
  ],

}
