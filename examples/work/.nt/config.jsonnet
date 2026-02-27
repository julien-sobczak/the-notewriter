local nt = import 'nt.libsonnet';

{
  slug: 'work',
  attributes: nt.DefaultAttributes,
  noteTypes: nt.DefaultNoteTypes {
    // Reference for reference notes
    Reference: nt.DefaultNoteTypes.Note {
      name: 'Reference',
    },
    // Project overview
    Overview: self.Note {
      name: 'Overview',
    },
    // Checklist for list of items to review
    Checklist: nt.DefaultNoteTypes.List {
      name: 'Checklist',
    },
    // Cheatsheet presents "How to..." solutions
    Cheatsheet: self.Note {
      name: 'Cheatsheet',
    },
  },

  linter: {
    rules: [
      nt.LintRules.NoEmptyTitle(),
      nt.LintRules.NoDuplicateNoteTitle(),
      nt.LintRules.NoDuplicateSlug(),
      nt.LintRules.NoDanglingMedia(),
      nt.LintRules.NoDeadWikilink(),
      nt.LintRules.NoExtensionWikilink(),
      nt.LintRules.NoAmbiguousWikilink(),
    ],
  },

  queries: {
    // Project Management
    workProjects: {
      title: 'Work Projects',
      query: 'path:projects/ @title:Overview',
      tags: ['project'],
    },
    workBacklog: {
      title: 'Work Backlog',
      query: 'path:projects/ type:Todo',
      tags: ['task'],
    },
  },

  journals: [
    {
      name: 'Work Diary',
      path: 'journal/{{ year }}/{{ year }}-{{ month }}-{{ day }}.md',
      defaultContent: '# Journal: {{ year }}-{{ month }}-{{ day }}',
    },
  ],

  decks: [
    {
      name: 'Work',
      // Matches all flashcards with default settings
    },
  ],
}
