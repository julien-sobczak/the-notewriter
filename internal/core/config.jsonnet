# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax
local nt = import 'nt.libsonnet';

{
    attributes: nt.DefaultAttributes,
    noteTypes: nt.DefaultNoteTypes,
    
    # Backward compatibility: also support "types" field
    types: nt.DefaultNoteTypes,
}
