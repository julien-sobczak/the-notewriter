{
    // Declare default attributes with special meaning for The NoteWriter
    DefaultAttributes: {
        due: {
            name: "due",
            description: "Due date",
            type: "date",
        },
        status: {
            name: "status",
            aliases: ["state"],
            description: "Status of a task",
            type: "string",
            options: ["todo", "planned", "in-progress", "done", "cancelled", "on-hold", "blocked", "archived"],
            defaultValue: "todo",
            shorthands: {
                "📝": "todo",
                "📅": "planned",
                "❓": "to-refine",
                "⏱️": "in-progress",
                "✅": "done",
                "❌": "cancelled",
                "⏸️": "on-hold",
                "🚧": "blocked",
                "🗄️": "archived",
            },
            preserveShorthand: false,
        },
        priority: {
            name: "priority",
            description: "Priority of a task",
            type: "string",
            options: ["low", "medium", "high", "urgent"],
            defaultValue: "medium",
            shorthands: {
                "🔽": "low",
                "🔼": "medium",
                "❗": "high",
                "🚨": "urgent",
            },
            preserveShorthand: false,
        },
        rating: {
            name: "rating",
            description: "Rating",
            type: "integer",
            min: 0,
            max: 10,
            shorthands: {
                "☆": 0,
                "\u2BE8": 1,
                "★": 2,
                "★\u2BE8": 3,
                "★★": 4,
                "★★\u2BE8": 5,
                "★★★": 6,
                "★★★\u2BE8": 7,
                "★★★★": 8,
                "★★★★\u2BE8": 9,
                "★★★★★": 10,
            },
            preserveShorthand: true,
        },
        read_date: {
            name: "read_date",
            type: "string", # Avoid type "date" to not dump a full date as timestamp
            format: "yyyy-mm-dd",
            inherit: true, // Often declared in Front Matter
            memory: true, // Used to mark this note as memory
        },
        attended_date: {
            name: "attended_date",
            type: "string", # Avoid type "date" to not dump a full date as timestamp
            format: "yyyy-mm-dd",
            inherit: true, // Often declared in Front Matter
            memory: true, // Used to mark this note as memory
        },
    },

    // Declare default note types
    DefaultTypes: {
        // Prefedefined objects = types with custom logic in The NoteWriter when processing them
        Note: {
            name: "Note",
        },
        Task: self.Note + {
            name: "Task",
            attributes: [
                {
                    name: "status",
                    required: true,
                },
                {
                    name: "due",
                },
                {
                    name: "priority",
                },
            ],
        },
        Journal: self.Note + {
            name: "Journal",
            attributes: [
                {
                    name: "date",
                    required: true,
                },
            ],
            preprocessors: ["date-extractor", "list-items"],
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
            attributes: [
                {
                    name: "srs.algorithm",
                },
                {
                    name: "srs.boostFactor",
                },
            ],
            preprocessors: ["flashcard-extractor"],
        },
        Todo: self.Note + {
            name: "Todo",
            attributes: [
                {
                    name: "status",
                    inline: true,
                    required: true,
                },
                {
                    name: "due",
                    inline: true,
                },
                {
                    name: "priority",
                    inline: true,
                    required: true,
                },
            ],
            preprocessors: ["list-items"],
        },
        Generator: self.Note + {
            name: "Generator",
            attributes: [
                {
                    name: "file",
                },
                {
                    name: "interpreter",
                },
            ],
            preprocessors: ["generator"],
        },
        ReadingList: self.Note + {
            name: "ReadingList",
            attributes: [
                {
                    name: "author",
                },
                {
                    name: "rating",
                },
            ],
            preprocessors: ["list-items"],
        },
        Master: self.Note + {
            name: "Master",
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
        NoImplicitSlugOnFlashcard()::
            {
                name: "no-implicit-slug-on-flashcard",
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
