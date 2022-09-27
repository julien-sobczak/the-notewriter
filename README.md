# The NoteTaker


## Dependencies

* [SQLite](https://www.sqlite.org/docs.html) with [FTS5 extension](https://www.sqlite.org/fts5.html#external_content_and_contentless_tables) + [`go-sqlite3`](https://github.com/mattn/go-sqlite3). SQLite is used to build a database from notes to speed up UI actions and make possible full-text searches.


## Developing

```shell
# Add new book reference
$ go run main.go reference new 0787960756

# Add new person reference
$ go run main.go reference new --kind=author Nelson

# Build the database
$ go run --tags "fts5" main.go build
```
