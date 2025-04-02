{
    Core: {
        extensions: ["md", "markdown"],
        medias: {
            command: "ffmpeg",
            parallel: 1,
            preset: "ultrafast",
        },
    },

    // Declare reserved attributes
    Attributes: { // IMPROVEMENT hardcoded in go?
        date: {
            name: "date",
            type: "string",
            format: "date", // TODO map[format]{pattern} ???
            inherit: true,
        },

        hook: {
            name: "hook",
            type: "string[]",
            inherit: false, // Subnotes must not trigger the hook
        },

        tags: {
            name: "tags",
            type: "string[]",
            inherit: true,
        },

        source: {
            name: "source",
            type: "string",
            format: "markdown",
            inherit: true,
        },

        references: {
            name: "references",
            type: "string[]",
            format: "markdown",
            inherit: false,
        },

        inspirations: {
            name: "inspirations",
            type: "string[]",
            format: "markdown",
            inherit: false,
        },
    },

    // Declare default object kinds
    Objects: {
        // Prefedefined objects = types with custom logic in The NoteWriter when processing them
        Note: {},
        Journal: $.Objects.Note + {
            requiredAttributes: ["date"],
        },
        Quote: $.Objects.Note,
        Flashcard: $.Objects.Note + {
            optionalAttributes: ["srs.algorithm", "srs.boostFactor"]
        },
        TODO: $.Objects.Note,
        Generator: $.Objects.Note + {
            optionalAttributes: ["file", "interpreter"]
        },
    },

    // Declare available Linter rules
    LintRules: {
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
        RequireQuoteTags(tags)::
            {
                name: "require-quote-tag",
                args: ["ˆ(" + std.join("|", tags) + ")$"],
            },
    },
}
