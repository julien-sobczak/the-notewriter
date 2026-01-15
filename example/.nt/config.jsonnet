# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax
local nt = import 'nt.libsonnet';

{
    core: {
        medias: {
            command: "ffmpeg",
            parallel: 1,
            preset: "ultrafast",
        },
    },

    # TODO add links to the documentation
    attributes: nt.DefaultAttributes + {
        rating: {
            name: "rating",
            type: "string",
            allowedValues: ["★", "★★", "★★★"],
            defaultValue: "★★",
            shorthands: {
                "★": "★",
                "★★": "★★",
                "★★★": "★★★",
            },
        },
    },

    # TODO add links to the documentation
    noteTypes: nt.DefaultNoteTypes + {
        Todo: nt.DefaultNoteTypes.Todo + {
            attributes: [
                {
                    name: "status",
                    optional: false,
                    inline: true,
                },
            ],
        },
        BookReview: nt.DefaultNoteTypes.Note + {
            name: "BookReview",
            attributes: [
                {
                    name: "rating",
                    optional: false,
                    inline: false,
                },
            ],
        },
        ReadingList: nt.DefaultNoteTypes.Note + {
            name: "ReadingList",
            attributes: [
                {
                    name: "rating",
                    optional: false,
                    inline: false,
                },
            ],
            processors: ["list-items"],
        },
    },

    # Predefined queries with tags
    queries: {
        allNotes: {
            title: "All Notes",
            query: "@type:Note",
            tags: ["general"],
        },
        projectNotes: {
            title: "Project Notes",
            query: "path:projects/ @type:Note",
            tags: ["projects", "work"],
        },
    },

    # Desks for organizing notes visually
    desks: [
        {
            name: "The NoteWriter",
            description: "The NoteWriter Project Management",
            root: {
                layout: "vertical",
                elements: [
                    {
                        name: "Notes",
                        size: "70%",
                        query: "path:projects/the-notewriter (@type:Note)",
                    },
                    {
                        layout: "horizontal",
                        elements: [
                            {
                                name: "Backlog",
                                query: "path:projects/the-notewriter @type:Todo",
                                view: "single",
                                size: "30%",
                            },
                            {
                                name: "Quotes",
                                query: "path:projects/the-notewriter (@type:Quote)",
                            },
                        ],
                    },
                ],
            },
        },
    ],

    # Journals for daily note-taking
    journals: [
        {
            name: "My Diary",
            path: "journal/${year}/${year}-${month}-${day}.md",
            defaultContent: "Journal: ${year}-${month}-${day}",
            routines: [
                {
                    name: "Morning Routine",
                    template: |||
                        # 💪 Affirmation

                        <Affirmation wikilink="journaling#List: Affirmations" tags="success,optimism" />

                        # ✍️ Morning Pages

                        <MorningPages throwAway />

                        # 😘 Gratitude Journal

                        3 things I appreciate:

                        * <Input />
                        * <Input />
                        * <Input />

                        # 🤔 Prompt

                        <Prompt wikilink="journaling#List: Prompts" />

                        # 🎯 My BIG thing for today

                        <Input />
                    |||,
                },
                {
                    name: "Shutdown Routine",
                    template: |||
                        # ❓ How was my day? Why?

                        <Input />

                        # 📋 3+1 tasks to complete tomorrow:

                        * [ ] <Input /> (work)
                        * [ ] <Input />
                        * [ ] <Input />
                        * [ ] <Input />
                    |||,
                },
            ],
        },
    ],

    # Stats for data visualization
    stats: [
        {
            name: "World Inspiration",
            query: "@type:Quote",
            groupBy: "nationality",
            visualization: "map",
            mapping: {
                "Roman": "ITA",
                "Greek": "GRC",
                "German": "DEU",
                "French": "FRA",
                "American": "USA",
                "English": "GBR",
            },
        },
    ],

    # Books configuration for generating ePub/PDF books
    books: [
        {
            title: "Sample Book",
            author: ["NoteWriter User"],
            language: "en-US",
            toc: true,
            format: ["epub", "pdf", "markdown"],
            chapters: [
                {
                    title: "Introduction",
                    illustration: "thoughts/medias/pencil.png",
                    text: "This is a sample book generated with The NoteWriter.",
                },
                {
                    title: "Part I",
                    subtitle: "Thoughts",
                    sections: [
                        {
                            title: "On Learning",
                            query: "path:\"thoughts/on-learning.md\"",
                            pageBreaks: false,
                            includeComments: false,
                        },
                        {
                            title: "On Doing",
                            query: "path:\"thoughts/on-doing.md\"",
                            pageBreaks: true,
                            includeComments: true,
                        },
                    ]
                },
                {
                    title: "Afterword",
                    text: importstr 'afterword.md',
                }
            ]
        }
    ],
}
