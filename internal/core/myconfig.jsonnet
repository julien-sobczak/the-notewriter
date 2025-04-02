# Demo configuration use when running commands manually
# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax

local nt = import 'nt.libsonnet';

local srsAlgorithmSettings = {
    easeFactor: 2.5,
};

{
    Core: {
        extensions: ["md", "markdown"],
        medias: {
            command: "ffmpeg",
            parallel: 1,
            preset: "ultrafast",
        },
    },

    Attributes: {
        // Identify the author (useful on quotes)
        name: {
            name: "name",
            aliases: ["author"],
            type: "string",
            format: "markdown",
            inherit: true, // Often declared in Front Matter
        },
        nationality: {
            name: "nationality",
            type: "string",
            format: "markdown",
            inherit: true, // Often declared in Front Matter
        },
        occupation: {
            name: "occupation",
            type: "string",
            format: "markdown",
            inherit: true, // Often declared in Front Matter
        },

        // Book
        isbn: {
            name: "isbn",
            type: "string",
            format: "isbn",
            inherit: true, // Often declared in Front Matter
        },

        // Project
        draft: {
            name: "draft",
            type: "boolean",
            inherit: false,
        },

        // Reviews
        rating: {
            name: "rating",
            type: "number",
            min: 0,
            max: 5,
            inherit: true,
        }
    } + nt.Attributes, // Reserved attributes take precedence

    Objects: {
        // Customize default object kinds
        Note: nt.Objects.Note,
        Journal: nt.Objects.Journal,
        Quote: nt.Objects.Quote + {
            requiredAttributes: ["name", "nationality", "occupation"],
        },
        Generator: nt.Objects.Generator,

        // Declare more-refined kinds for specific uses
        Reference: nt.Objects.Note,
        Cheatsheet: nt.Objects.Note,
        Artwork: nt.Objects.Note + {
            optionalAttributes: ["artist", "illustrator", "year"],
        },
        Snippet: nt.Objects.Note,
        BookReview: nt.Objects.Note + {
            requiredAttributes: ["draft", "isbn"],
        }
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
            nt.LintRules.RequireQuoteTags([
                "learning",
                "mind-blowing",
                "believing",
                "meaning",
                "sharing",
                "working",
                "leading",
                "focusing",
                "speaking",
                "reflecting",
                "buying",
                "knowing",
                "living",
                "doing",
                "being",
                "suffering",
                "loving",
                "dying",
                "perseverance",
                "growing",
                "compassion",
                "money",
                "life",
                "passion",
                "purpose",
                "belief",
                "work",
                "parenting",
                "success",
                "history",
                "thinking",
                "ignorance",
                "problem-solving",
                "creativity",
                "courage",
                "productivity",
                "programming",
                "reading",
                "deciding",
                "sleeping",
                "leadership",
                "understanding",
                "management",
                "mindfulness",
                "meditation",
                "intelligence",
                "health",
                "planning",
                "happiness",
                "complexity",
                "widsom",
                "gratitude",
                "art",
                "drawing",
                "running",
                "writing",
                "stress",
                "note-taking",
                "innovation",
                "relationship",
                "humor",
                "imagination",
                "persuasion",
                "excellence",
                "changing",
                "listening",
                "death",
                "philosophy",
                "friendship",
                "time",
                "aging",
                "curiosity",
                "habit",
                "memory",
                "self-help",
                "love",
                "fear",
            ])
        ],
    },

    Searches: {
        favoriteQuotes: {
            title: "Favorite Quotes",
            q: "-#ignore @object:quote",
        },
    },

    Decks: [
        {
            name: "Life",
            query: "path:skills",
            newFlashcardsPerDay: 10,
            algorithmSettings: srsAlgorithmSettings,
        }
    ],

    References: [
        // books
        {
            title: "A book",
            manager: "google-books",
            path: 'references/books/{{index . "title" | slug}}.md',
            template: |||
                title: "{{index . "title" | title}}{{ if index . "subtitle"}}:{{index . "subtitle" | title}}{{end}}"
                short_title: "{{index . "title" | title}}"
                name: {{index . "authors" | join ", "}}
                occupation: Unknown
                nationality: Unknown
                {{- if index . "publishedDate"}}
                date: "{{index . "publishedDate"}}"
                {{- end -}}
                {{- if index . "publisher"}}
                publisher: {{index . "publisher"}}
                {{- end -}}
                {{- if index . "pageCount"}}
                numPages: {{index . "pageCount"}}
                {{- end -}}
                {{- if index . "unknown"}}
                unknown: {{index . "unknown"}}
                {{- end -}}
                {{- if index . "industryIdentifiers"}}
                isbn: "{{index . "industryIdentifiers" | jq ". | first | .identifier"}}"
                {{- end }}
                ---

                # {{index . "title" | title}}
            |||
        }

        // persons
        {
            title: "A person",
            manager: "wikipedia",
            path: 'references/persons/{{index . "name" | slug}}.md',
            template: |||
                ---
                name: {{index . "name"}}
                occupation: {{if index . "occupation"}}{{index . "occupation"}}{{else}}Unknown{{end}}
                nationality: {{if index . "nationality"}}{{index . "nationality"}}{{else}}Unknown{{end}}
                {{- if index . "birth_date"}}
                birth_date: {{index . "birth_date"}}
                {{- end -}}
                {{- if index . "death_date"}}
                death_date: {{index . "death_date"}}
                {{- end -}}
                {{- if index . "known_for"}}
                known_for: "{{index . "known_for"}}"
                {{- end }}
                ---

                # {{index . "name"}}
            |||
        }
    ]

}
