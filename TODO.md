# TODO

## TODO: Backlog

* [x] Append reserved attributes systematically
* [x] Generate config in `init` using a Go template instead
* [x] Add flag `-i` in `nt init` and a comment `IMPROVEMENT ask questions to customize the config file`
* [ ] Add `postprocessing` attribute in `ConfigType`
* [ ] Add attribute `pattern` in `ConfigType` to use custom heading formats
* [ ] Remove `NoteType` constants completely (= true generic type system 💪)
* [ ] Rework documentation `linter.md` and introduce a new `config.md`

## TODO: Features

* [x] Rework config to use Jsonnet
* [ ] Rework to remove note types completely
* [ ] Implement operations using CRDT datatypes

## TODO: Sprint

* [ ] Implement `GC()` on remotes
* [ ] Implement option `-i` in `nt pull`/`nt push`
* [ ] Add tests on `ObjectDiffs` for `Patch`/etc.
* [ ] Add many many many more tests in `parser_test.go` 💪
* [ ] Drop support for free notes? If yes, remove the linter rule `no-free-note`, and edit the doc.

## TODO: Improvement

* [ ] Add a "Cheatsheet: Fixtures using `testdata`" + "Cheatsheet: Fixtures using raw files" in notes
* [ ] Write custom assertion to compare `ToJSON` and `ToYAML` ignore spaces
