local nt = import 'nt.libsonnet';

local srsAlgorithmSettings = {
  easeFactor: 2.5,
};

local makeHeading = nt.Schema.makeHeading;

{
  slug: 'life',
  attributes: nt.DefaultAttributes {
    date: {
      name: 'date',
      type: 'string',
      format: 'yyyy-mm-dd',
      inherit: true,
    },
    // Identify the author (useful on quotes)
    name: {
      name: 'name',
      aliases: ['author'],
      type: 'string',
      format: 'markdown',
      inherit: true,  // Often declared in Front Matter
    },
    occupation: {
      name: 'occupation',
      type: 'string',
      format: 'markdown',
      inherit: true,  // Often declared in Front Matter
    },

    // Book
    isbn: {
      name: 'isbn',
      type: 'string',
      format: 'isbn',
      inherit: true,  // Often declared in Front Matter
    },

    // Project
    draft: {
      name: 'draft',
      type: 'bool',
      inherit: false,
    },

    // Readings
    read_date: {
      name: 'read_date',
      type: 'string',  // Avoid type "date" to not dump a full date as timestamp
      format: 'yyyy-mm-dd',
      inherit: true,  // Often declared in Front Matter
      memory: true,  // Used to mark this note as memory
    },
    rating: {
      name: 'rating',
      type: 'string',
      allowedValues: ['★', '★★', '★★★', '★★★★', '★★★★★'],
      defaultValue: '★★★',
      shorthands: {
        '★': '★',
        '★★': '★★',
        '★★★': '★★★',
        '★★★★': '★★★★',
        '★★★★★': '★★★★★',
      },
    },
  },

  tags: nt.DefaultTags,

  noteTypes: nt.DefaultNoteTypes {
    // Customize default note types
    Quote: nt.DefaultNoteTypes.Quote {
      // Override to enforce a few attributes
      attributes: [
        {
          name: 'name',
          required: true,
        },
        {
          name: 'occupation',
          required: true,
        },
      ],
    },
    Artwork: nt.DefaultNoteTypes.Artwork {
      attributes: [
        {
          name: 'artist',
        },
        {
          name: 'year',
        },
      ],
    },

    // Declare custom types for specific uses
    Reference: nt.DefaultNoteTypes.Note {
      name: 'Reference',
    },
    // Synopsis presents the project overview (= the initial idea)
    Synopsis: self.Note {
      name: 'Synopsis',
    },
    Idea: self.Note {
      name: 'Idea',
    },
    // Cheatsheet presents "How to..." solutions
    Cheatsheet: self.Note {
      name: 'Cheatsheet',
    },
    // BookReview is a note reviewing a book
    BookReview: self.Note {
      name: 'BookReview',
      attributes: [
        {
          // ISBN is required to identify the book (often inherited from the book reference)
          name: 'isbn',
          required: true,
        },
        {
          // Omitted the attribute is not allowed to prevent a review from being published before it's ready
          name: 'draft',
          required: true,
        },
        {
          // Rating of the book
          name: 'rating',
          required: true,
        },
        {
          // Date when the book was read
          name: 'read_date',
          required: true,
        },
      ],
    },
    // ReadingList is a list of books (read, to read, dnf, etc.)
    ReadingList: nt.DefaultNoteTypes.Note {
      name: 'ReadingList',
      attributes: [
        {
          name: 'rating',
          optional: true,
          inline: false,
        },
      ],
      processors: ['list-items'],
    },
  },

  fileTypes: nt.DefaultFileTypes {
    Project: {
      name: 'Project',
      attributes: [
        {
          name: 'source',  // Must be a GitHub repo URL
          required: true,
        },
      ],
      deskTemplates: ['Project Template'],
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

  desks: [
    {
      name: 'Project Template',
      description: 'Visualize your project',
      template: true,
      root: {
        layout: 'vertical',
        elements: [
          {
            layout: 'horizontal',
            elements: [
              {
                name: 'Synopsis',
                query: 'type:Synopsis',
                view: 'single',
              },
              {
                name: 'Lists',
                query: 'type:List',
                view: 'list',
              },
            ],
          },
          {
            name: 'Tasks',
            query: 'type:Tasks',
            view: 'grid',
          },
          {
            name: 'Ideas',
            query: 'type:Master @title:Ideas',
            view: 'single',
          },
        ],
      },
    },
  ],

  linter: {
    rules: [
      nt.LintRules.NoEmptyTitle(),
      nt.LintRules.NoDuplicateNoteTitle(),
      nt.LintRules.NoDuplicateSlug(),
      nt.LintRules.NoDanglingMedia(),
      nt.LintRules.NoDeadWikilink(),
      nt.LintRules.NoExtensionWikilink(),
      nt.LintRules.NoAmbiguousWikilink(),
      nt.LintRules.NoOrphanFlashcard(),
      nt.LintRules.RequireTagIf('type:Quote', [
        'focusing',
        'reading',
        'understanding',
        'knowing',
        'learning',
        'mastering',
        'doing',
        'being',
        'reflecting',
        'living',
        'suffering',
        'loving',
        'problem-solving',
        'thinking',
        'programming',
        'planning',
        'writing',
        'note-taking',
        'aging',
        'dying',
      ]),
    ],
  },

  queries: {
    // Show random quote at startup
    dailyQuote: {
      title: 'Daily Quote',
      query: 'path:resources/books type:Quote',
      tags: ['daily-quote'],
    },
    // A few custom queries to review some notes regularly
    favoriteQuotes: {
      title: 'Favorite Quotes',
      query: '#favorite type:Quote',
      tags: ['daily-quote'],
    },
    unpublishedReviews: {
      title: 'Unpublished Book Reviews',
      query: '@draft:true type:BookReview',
    },
    myNextBook: {
      title: 'Reading List',
      query: 'type:ReadingList',
    },

    // Configure sources of inspiration
    inspirationArt: {
      title: 'Art',
      query: 'path:resources/art type:Artwork',
      tags: ['inspiration'],
    },
    inspirationFavorite: {
      title: 'Favorite',
      query: '#favorite',
      tags: ['inspiration'],
    },

    // Project Management
    sideProjects: {
      title: 'Side Projects',
      query: 'path:projects/ @title:Synopsis',
      tags: ['project'],
    },
    personalBacklog: {
      title: 'Personal Backlog',
      query: 'path:projects/ type:Todo',
      tags: ['task'],
    },

    // Zen Mode
    zenQuotes: {
      title: 'Bits of Books',
      query: 'path:references/books type:Quote',
      tags: ['zen'],
    },
    zenThoughts: {
      title: 'Bits of Learning',
      query: 'path:area/learning type:Quote',
      tags: ['zen'],
    },

  },

  journals: [
    {
      name: 'My Diary',
      path: 'journal/{{ year }}/{{ year }}-{{ month }}-{{ day }}.md',
      defaultContent: '# Journal: {{ year }}-{{ month }}-{{ day }}',
      routines: [
        {
          name: 'Morning Routine',
          template: |||
            # 💪 Affirmation

            {{ randomListItem "journaling#List: Affirmations" }}

            # 😘 🗑️ Gratitude Journal

            3 things I appreciate:

            * _Thing 1_
            * _Thing 2_
            * _Thing 3_

            # 🤔 Prompt

            {{ randomListItem "journaling#List: Prompts" }}
            _Your Answer_

            # 🎯 My BIG thing for today

            _Your Answer_
          |||,
        },
        {
          name: 'Shutdown Routine',
          template: |||
            # ❓ How was my day? Why?

            👍/👎

            # 📋 3 tasks to complete tomorrow:

            * [ ] _Task 1_
            * [ ] _Task 2_
            * [ ] _Task 3_
          |||,
        },
      ],
    },
  ],

  decks: [
    {
      name: 'Skills',
      query: 'path:resources/skills',
      newFlashcardsPerDay: 10,
      algorithmSettings: srsAlgorithmSettings,
    },
    {
      name: 'General',
      // Every other flashcards that didn't match above decks
      newFlashcardsPerDay: 10,
      algorithmSettings: srsAlgorithmSettings,
    },
  ],

  books: [
    {
      title: 'Reflections on Life',
      subtitle: 'Modern Wisdom from Timeless Quotes',
      format: ['markdown'],
      author: ['Various'],
      language: 'en',
      chapters: [
        {
          title: 'On Being',
          query: 'path:thoughts/on-being',
          icon: 'thoughts/medias/on-being.png',
        },
        {
          title: 'On Doing',
          query: 'path:thoughts/on-doing',
          icon: 'thoughts/medias/on-doing.png',
        },
        {
          title: 'On Learning',
          query: 'path:thoughts/on-learning',
          icon: 'thoughts/medias/on-learning.png',
        },
        {
          title: 'On Thinking',
          query: 'path:thoughts/on-thinking',
          icon: 'thoughts/medias/on-thinking.png',
        },
      ],
    },
  ],
}
