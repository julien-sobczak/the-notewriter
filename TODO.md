# TODO

## TODO: Backlog

* [ ] Rework documentation `linter.md` and introduce a new `config.md`
* [ ] Add tests for `GeneratorPreprocessor`

## TODO: Bug Fixes

* [ ] Add linter `no-empty-title` (ex: `## Flashcard: ` is accepted...)
* [ ] Understand why 900 flashcards in SQLite but 1350 `## Flashcard:` in VS Code 🤔
* [ ] `nt add .` `nt gc` `nt add .` `nt gc` ... Files are always garbage-collected when nothing must be adding and garbage-collected on second run => Add a unit test
* [ ] Rework `CastDateFn` to keep the string content (better when exploiting fields later) => Use `type: string` and `format: date` instead and have a list of predefined formats (like `isbn` to check in `CheckAttributes`)

## TODO: Sprint

* [ ] Fix unit test `TestEvaluateTimeExpressionAfter`
* [ ] Implement `GC()` on remotes
* [ ] Implement option `-i` in `nt pull`/`nt push`
* [ ] Add tests on `ObjectDiffs` for `Patch`/etc.
* [ ] Add many many many more tests in `parser_test.go` 💪

## TODO: Improvement

* [ ] Move Datastore method on `DB`
* [ ] Add a "Cheatsheet: Fixtures using `testdata`" + "Cheatsheet: Fixtures using raw files" in notes
* [ ] Write custom assertion to compare `ToJSON` and `ToYAML` ignore spaces
