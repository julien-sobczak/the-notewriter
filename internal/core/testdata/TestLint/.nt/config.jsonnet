local nt = import 'nt.libsonnet';

{
    Core: nt.Core + {
        medias: {
            command: "random",
        },
    },

    Attributes: nt.Attributes + {
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
    },

    Types: nt.Types + {
        Quote: nt.Types.Quote + {
            requiredAttributes: ["name"],
        },
    },

    Linter: {
        Rules: [
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
