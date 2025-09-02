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

    attributes: {
        status: {
            name: "status",
            aliases: ["state"],
            type: "string",
            allowedValues: ["todo", "in-progress", "blocked", "done"],
            shorthands: {
                "📋": "todo",
                "🕒": "in-progress",
                "⛔": "blocked",
                "✅": "done",
            },
            preserveShorthand: false,
        },
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
}
