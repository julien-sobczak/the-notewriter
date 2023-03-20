# Thinking

## Notepad

- process index.md before other files when traversing directories. <= TODO change the walk logic
- Check for a optional local `index.md` to pass in file constructors.
- Add new columns `parent_file_id` and `attributes` in table `file` storing the JSON merged attributes.

```
walk to find files # can contains wikilinks to any file...
for f in files:
  f = ParseFile(f)
  for wikilink in f.Wikilinks:


// Find files to process
// Create an array with their dependencies
*.md => index.md
index.md => InheritableAttributesChanged()? => must add below notes

start with all index.md from root to bottom
if index.md New or AttributesChanged() => rebuild all index.md below (matching the add)

// if we are sure notes to includes or notes/files to inherit are processed first and already saved in DB
// then, we can query for note using their wikilink and file using their relative path if we are inside the same tx



MergeAttributes(file.GetAttributes(), file.GetAttributes(), ...)
```



## Options

### Recursive index.md inheritance

Ex:

```
/index.md
/references/inspirations/
  index.md
  arts/
    paintings.md # Inherit from ./index.md, ../index.md, ../../../index.md
    index.md
```

Pro(s):

* Flexible: Easy to change the default ease-factor globally for all flashcards
* True cascading: closer to CSS

Con(s):

* Updates: As we denormalized attributes in notes to make them searchable easily (`@priority:high`)a change can build rebuilding thousands of notes objects in `.nt/objects` = huge space
* Implementation: Code is clearly not trivial.


### Single directory index.md

Ex:

```
/index.md
/references/inspirations/
  index.md
  arts/
    paintings.md # Inherit only from ./index.md
    index.md
```

Pro(s):

* Still flexible: Can save from most duplication in practice.
* Implementation: Relatively easy.

Con(s):

* Not ideal: Not the best solution from a user's viewpoint.
* Duplication: Some attributes must be duplicated in different directories (an acceptable compromise?)
