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
    types: nt.DefaultTypes + {
        Todo: nt.DefaultTypes.Todo + {
            attributes: [
                {
                    name: "status",
                    optional: false,
                    inline: true,
                },
            ],
        },
        BookReview: nt.DefaultTypes.Note + {
            name: "BookReview",
            attributes: [
                {
                    name: "rating",
                    optional: false,
                    inline: false,
                },
            ],
        },
    },

    # Books configuration for generating ePub/PDF books
    books: [
        {
            title: "Sample NoteWriter Book",
            author: ["NoteWriter User"],
            language: "en-US",
            toc: true,
            format: ["epub", "pdf", "markdown"],
            chapters: [
                {
                    title: "Introduction",
                    text: "This is a sample book generated from NoteWriter notes.\n\nIt demonstrates how to collect and organize notes into a cohesive publication."
                },
                {
                    title: "Thoughts and Reflections", 
                    sections: [
                        {
                            title: "Notes",
                            query: "type:Note",
                            pageBreaks: true,
                            includeComments: false,
                        }
                    ]
                }
            ]
        }
    ],
}
