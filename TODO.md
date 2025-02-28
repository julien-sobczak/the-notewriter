# TODO

## TODO: Sprint

* [ ] Merge branch?
* [ ] Implement `GC()` on remotes
* [ ] Rework `nt cat-file`
* [ ] Implement option `-i` in `nt pull`/`nt push`
* [ ] Add tests on `ObjectDiffs` for `Patch`/etc.
* [ ] Rework `nt hook`
* [ ] Add many many many more tests in `parser_test.go` 💪
* [ ] Check how the slug is evaluated in `ParseNote()`. Here is the old code in `note.go`:

```go
func (n *Note) updateSlug() {
	// Slug is determined based on the following values
	var fileSlug string
	var attributeSlug string
	var kind NoteKind
	var shortTitle markdown.Document

	// Check if a specific slug is specified
	if newSlug, ok := n.Attributes.Slug(); ok {
		attributeSlug = newSlug
	}

	// Check the slug on the file
	if n.GetFile() != nil { // FIXME jso now!!!!!
		fileSlug = n.GetFile().Slug
	}

	kind = n.NoteKind
	shortTitle = n.ShortTitle

	newSlug := DetermineNoteSlug(
		fileSlug,
		attributeSlug,
		kind,
		string(shortTitle),
	)
	if n.Slug != newSlug {
		n.Slug = newSlug
		n.stale = true
	}
}


// DetermineNoteSlug determines the note slug from the attributes.
func DetermineNoteSlug(fileSlug string, attributeSlug string, kind NoteKind, shortTitle string) string {
	if attributeSlug != "" {
		// @slug takes priority
		return attributeSlug
	}

	// Slug must be generated
	return markdown.Slug(fileSlug, string(kind), shortTitle)
}
```


* [ ] Check how is determined the long title in `ParsedNote`. Here is the old code in `note.go`:

```go
// func (n *Note) updateLongTitle() {
// 	var titles []markdown.Document
// 	if n.GetFile() != nil && n.GetFile().ShortTitle != "" {
// 		titles = append(titles, n.GetFile().ShortTitle)
// 	}
// 	titles = append(titles, n.ShortTitle)
// 	newLongTitle := FormatLongTitle(titles...)
// 	if n.LongTitle != newLongTitle {
// 		n.LongTitle = newLongTitle
// 		n.stale = true
// 	}
// }
```

* [ ] Rewrite doc

## TODO: Improvement

* [ ] `Pull` `Push` in `Repository` or `DB`?
* [ ] Add a "Cheatsheet: Fixtures using `testdata`" + "Cheatsheet: Fixtures using raw files" in notes
* [ ] Move `NewOrExistingXXX`, `NewPackFileXXX` to `Repository`, etc.
* [ ] Write custom assertion to compare `ToJSON` and `ToYAML` ignore spaces
