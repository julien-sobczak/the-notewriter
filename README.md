# The NoteWriter


## Dependencies

* [SQLite](https://www.sqlite.org/docs.html) with [FTS5 extension](https://www.sqlite.org/fts5.html#external_content_and_contentless_tables) + [`go-sqlite3`](https://github.com/mattn/go-sqlite3). SQLite is used to build a database from notes to speed up UI actions and make possible full-text searches.
* [ffmpeg](https://github.com/FFmpeg/FFmpeg) binary (`brew install ffmpeg`). Used to convert the media files to different formats.


## Debug

### How to debug a unit test using SQLite?

Define a breakpoint. Inspect the repository root directory. Then:

```shell
# On MacOS if the application is installed
$ open -a "DB Browser for SQLite" dirname/.nt/database.db
```

Check the tables, then close the application and resume the debugging session.


## Usage

```shell
$ nt init
# Create a new directory .nt with default configuration files
$ cat .nt/config
# ...

$ nt add . && nt commit
```


## Documentation

Documentation is present in the directory `website/`. Run `make docs` to start locally.
