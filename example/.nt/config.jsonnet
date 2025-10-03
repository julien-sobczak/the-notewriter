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
