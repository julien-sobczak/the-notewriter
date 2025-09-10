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
        ReadingList: nt.DefaultTypes.Note + {
            name: "ReadingList",
            attributes: [
                {
                    name: "rating",
                    optional: false,
                    inline: false,
                },
            ],
            preprocessors: ["list-items"],
        },
    },
}
