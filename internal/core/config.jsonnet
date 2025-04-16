# See https://jsonnet.org/learning/tutorial.html to learn the Jsonnet syntax
local nt = import 'nt.libsonnet';

{
    Core: nt.Core,
    Attributes: nt.Attributes,
    Types: nt.Types,
}
