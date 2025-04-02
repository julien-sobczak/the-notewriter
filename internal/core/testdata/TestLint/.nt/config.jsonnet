local nt = import 'nt.libsonnet';

{
    Core: nt.Core + {
        medias: {
            command: "random",
        },
    },

    Attributes: nt.Attributes,

    Objects: nt.Objects,

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
