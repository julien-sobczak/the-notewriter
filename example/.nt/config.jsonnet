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
    types: nt.DefaultTypes,
}
