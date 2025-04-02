# TODO

## TODO: Backlog

* [ ] Call `CheckAttributes` in `nt add` after having fix the unit test in `lint_test.go`
* [ ] Rework `NoteKind` to use `ObjectType` and declare constants for predefined types
* [ ] Push `config.json` to `.nt` and push it to remotes
* [ ] Complete attributes in `config.jsonnet`

## TODO: Features

* [ ] Rework config to use Jsonnet
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
