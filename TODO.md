# TODO

## TODO: Sprint

* [ ] Cleanup `.bak` files
* [ ] Merge branch?
* [ ] Implement `GC()` on remotes
* [ ] Rework `nt cat-file`
* [ ] Implement option `-i` in `nt pull`/`nt push`
* [ ] Add tests on `ObjectDiffs` for `Patch`/etc.
* [ ] Reread my comment + Add logic in `GenerateBlobs()` to reread from `.nt/objects` first
    ```go
	// BUG?
	// packfile are created before objects (File, Note) to have the pack file OID when creating the object
	// When to generate blobs?
	// - PackFile is created in memory but not on disk
	// - File/Media are created in memory and appended to PackFile
	// - GenerateBlobs() creates blob files on disk using file content hash as OID
	//   => 💥 If the command crashes, some blobs will have been created on disk.
	//         When relaunching the command, the method GenerateBlobs() must find previous blobs but how?
	//         Solution: In NewOrExistingFile/Media, Load by searching for file hash (same Markdown, same media file) = same object
	// - PackFile is saved on disk
	// ...
	// When all packfiles are saved
	// - PackFile is saved in DB
	```
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

* [ ] Use or remove parameter `msg` in `nt commit`
* [ ] Rework `oid.UseSequence()` to invert the digits to support `ShortOID`. Ex: `"0000000000000000000000000000000000000002` => `2000000000000000000000000000000000000000`
* [ ] Exploit `if CurrentConfig().DryRun` in `NewPackFileFromXXX` to avoid bool `saveOnDisk`
* [ ] Try to remove `BlobPath` and `PackFilePath` by `BlobRef.ObjectPath()` and `PackFileRef.ObjectPath()` instead
* [ ] `Pull` `Push` in `Repository` or `DB`?
* [ ] Add a "Cheatsheet: Fixtures using `testdata`" + "Cheatsheet: Fixtures using raw files" in notes
* [ ] Move `NewOrExistingXXX`, `NewPackFileXXX` to `Repository`, etc.
* [ ] Merge `MustWriteFile` (old, better) and `MustWriteFileFromRelativePath` (new) together
* [ ] Search for `os.WriteFile` in tests and replace by `WriteFileFromRelativePath` instead
* [ ] Search for `clock.FreezeAt` and replace by more human-friendly `clock.FreezeOn` instead (but check first `FreezeAt(t, HumanTime(t, "2023-01-01 01:12:30"))`)
* [ ] Use `x.` in `UPSERT` (methods `Save()`) to avoid duplicating fields `@source: https://www.sqlite.org/lang_upsert.html`
* [ ] Writer custom assertion to compare `ToJSON` and `ToYAML` ignore spaces
