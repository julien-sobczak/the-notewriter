# The NoteWriter

_The NoteWriter_ is a minimally-designed, file-based note management system for creative developers.

_The NoteWriter_ is a CLI to parse Markdown files and extract objects (notes, flashcards, reminders, links). Objects are saved in packfiles (YAML files) that are pushed/pulled between devices and into a SQLite database for indexing, search, and performance reasons.

[_The NoteWriter Desktop_](https://github.com/julien-sobczak/the-notewriter-desktop/) is the desktop companion to use your notes in a creative way: journaling, studying flashcards, organizing using custom desks, finding inspiration by pondering on past notes, completing reminders, managing personal projects with tasks and (not so) much more.

[_The NoteWriter Nomad_](https://github.com/julien-sobczak/the-notewriter-nomad) (upcoming) will be the mobile companion to carry your notes with you and study your flashcards on the go.

Check the [landing page](https://julien-sobczak.github.io/the-notewriter/) to understand the philosophy and the unique features or browse the documentation to understand my [motivations](https://julien-sobczak.github.io/the-notewriter/why/), the core [principles](https://julien-sobczak.github.io/the-notewriter/developers/presentation/) that influence the implementation.

There are many modern note-taking applications trying to do more than competitors. **_The NoteWriter_ does less, but differently.**


## Installing

### Using the install script (recommended)

The easiest way to install The NoteWriter is using the installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/julien-sobczak/the-notewriter/main/install.sh | sh
```

This will download the latest release for your platform and install it to `$HOME/.nt/bin`. Follow the instructions printed by the script to add the directory to your `$PATH`.

You can also install a specific version by passing the version tag as an argument:

```bash
curl -fsSL https://raw.githubusercontent.com/julien-sobczak/the-notewriter/main/install.sh | sh -s v0.0.1
```

### Manual installation

You can download the latest prebuilt binaries for Linux, macOS, and Windows from the [Releases page](https://github.com/julien-sobczak/the-notewriter/releases).

#### Latest release

You can always fetch the latest version directly using GitHub’s `/releases/latest` endpoint:

| Platform | Download link |
|-----------|----------------|
| **Linux (amd64)**   | [Download](https://github.com/julien-sobczak/the-notewriter/releases/latest/download/nt-linux-amd64.tar.gz) |
| **macOS (amd64)**   | [Download](https://github.com/julien-sobczak/the-notewriter/releases/latest/download/nt-darwin-amd64.tar.gz) |
| **Windows (amd64)** | [Download](https://github.com/julien-sobczak/the-notewriter/releases/latest/download/nt-windows-amd64.tar.gz) |


#### Example (Linux)

```bash
curl -L https://github.com/julien-sobczak/the-notewriter/releases/latest/download/nt-linux-amd64.tar.gz | tar xz
sudo mv nt /usr/local/bin/
```


## Quick Start

```shell
$ nt init
# Create a new directory .nt with default configuration files
$ cat .nt/config.jsonnet
# ...

# Add some Markdown files
$ nt add . && nt commit
```


## Documentation

Documentation is present in the directory `website/`. Run `make docs` to start locally.


## Project Structure

```
.
├── cmd/           # CLI tools (nt, nt-anki, nt-book, etc.)
├── internal/      # Core logic (core, markdown, medias, etc.)
├── pkg/           # Public packages
├── build/         # Compiled binaries
├── example/       # Example repository and config
├── e2e/           # End-to-end tests
├── website/       # Documentation site (Astro/Starlight)
└── ...
```

## Dependencies

* [SQLite](https://www.sqlite.org/docs.html) with [FTS5 extension](https://www.sqlite.org/fts5.html#external_content_and_contentless_tables) + [`go-sqlite3`](https://github.com/mattn/go-sqlite3). SQLite is used to build a database from notes to speed up UI actions and make possible full-text searches.
* [ffmpeg](https://github.com/FFmpeg/FFmpeg) binary (`brew install ffmpeg`). Used to convert the media files to different formats.
* _(Optional)_ [Pandoc](https://pandoc.org/) and [Weasyprint](https://weasyprint.org/) for book generation.


## Troubleshooting

### How to debug a unit test using SQLite?

Define a breakpoint. Inspect the repository root directory. Then:

```shell
# On MacOS if the application is installed
$ open -a "DB Browser for SQLite" dirname/.nt/database.db
```

Check the tables, then close the application and resume the debugging session.


## License

[GNU General Public License v3.0](./LICENSE)
