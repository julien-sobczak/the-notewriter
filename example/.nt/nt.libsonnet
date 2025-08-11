{
    // Declare default note types
    DefaultTypes: {
        // Prefedefined objects = types with custom logic in The NoteWriter when processing them
        Note: {
            name: "Note",
        },
        Journal: self.Note + {
            name: "Journal",
            requiredAttributes: ["date"],
            preprocessors: ["date-extractor"],
        },
        Quote: self.Note + {
            name: "Quote",
            preprocessors: ["quote-rewriter"],
        },
        Artwork: self.Note + {
            name: "Artwork",
        },
        Flashcard: self.Note + {
            name: "Flashcard",
            optionalAttributes: ["srs.algorithm", "srs.boostFactor"],
            preprocessors: ["flashcard-extractor"],
        },
        Todo: self.Note + {
            name: "Todo",
        },
        Generator: self.Note + {
            name: "Generator",
            optionalAttributes: ["file", "interpreter"],
            preprocessors: ["generator"],
        },
    },

    // Declare available Linter rules
    LintRules: {
        NoEmptyTitle()::
            {
                name: "no-empty-title",
            },
        NoDuplicateNoteTitle()::
            {
                name: "no-duplicate-note-title",
            },
        NoDuplicateSlug()::
            {
                name: "no-duplicate-slug",
            },
        MinLinesBetweenNotes(count)::
            {
                name: "min-lines-between-notes",
                args: [count],
            },
        MaxLinesBetweenNotes(count)::
            {
                name: "max-lines-between-notes",
                args: [count],
            },
        NoDanglingMedia()::
            {
                name: "no-dangling-media",
            },
        NoDeadWikilink()::
            {
                name: "no-dead-wikilink",
            },
        NoExtensionWikilink()::
            {
                name: "no-extension-wikilink",
            },
        NoAmbiguousWikilink()::
            {
                name: "no-ambiguous-wikilink",
            },
        RequireTag(tags)::
            {
                name: "require-tag",
                args: ["^(" + std.join("|", tags) + ")$"],
            },
        RequireTagIf(query, tags)::
            {
                name: "require-tag",
                args: ["^(" + std.join("|", tags) + ")$"],
                query: query,
            },
    },
}
