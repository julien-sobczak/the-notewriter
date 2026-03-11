# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    tags: nt.DefaultTags,
    noteTypes: nt.DefaultNoteTypes,
}
