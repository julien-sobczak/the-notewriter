local nt = import 'nt.libsonnet';

{
    core: {
        medias: {
            command: "random",
        },
    },

    attributes: {
        // Add a name attribute that is required on type Quote
        name: {
            name: "name",
            aliases: ["author"],
            type: "string",
            inherit: true,
        },
        // Declare a pattern for the attribute isbn
        isbn: {
            name: "isbn",
            type: "string",
            pattern: "^([0-9-]{10}|[0-9]{3}-[0-9]{10})$",
            inherit: true,
        },
        // Declare a date attribute
        birth_date: {
            name: "birth_date",
            type: "date",
            format: "yyyy-mm-dd",
            inherit: true,
        },
    },

    noteTypes: nt.DefaultNoteTypes + {
        Quote: nt.DefaultNoteTypes.Quote + {
            attributes: [
                {
                    name: "name",
                    required: true,
                },
            ],
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

}
