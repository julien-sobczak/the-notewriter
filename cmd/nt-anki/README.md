# nt-anki

`nt-anki` is a CLI tool for importing Anki flashcards into _The NoteWriter_ repository.

## Features

- Converts only Anki `.apkg` collections to Markdown notes (`.colpkg` files will not be supported).
- Copies media files referenced in Anki cards.
- Creates packfiles containing review operations for imported flashcards.

## Usage

```bash
nt-anki import <anki.apkg> [--staged] [--media-dir DIR] [--ignore-scheduling] [--tag-mapping TAG=FILE ...]
```

### Options

- `--staged`
  If set, newly created packfiles are staged in the index for commit.

- `--media-dir DIR`
  Specify a subdirectory for media files (images, audio) referenced in Anki cards.

- `--ignore-scheduling`
  If set, skips creation of packfiles for reviews and only outputs Markdown and media.

- `--tag-mapping TAG=FILE`
  Map Anki tags to specific Markdown output files. Use `default=FILE` for notes without a matching tag.

### Example

```bash
nt-anki import mydeck.apkg --staged --media-dir medias \
  --tag-mapping english=skills/english.md \
  --tag-mapping software=skills/software.md \
  --tag-mapping default=skills/misc.md
```

## Workflow

1. Export a Anki deck (from main screen) or a subset of your notes (from browser screen).
2. Run `nt-anki --ignore-scheduling` to check the generated Markdown first.
3. Reset the changes using `git restore`
4. Rerun `nt-anki --staged` to saved reviews
5. Edit the Markdown files to define flashcard titles.
6. Commit the changes using `git commit`.

