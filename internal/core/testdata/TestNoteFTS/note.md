# Note

## Reference: FTS5

FTS5 is an SQLite [virtual table module](https://www.sqlite.org/c3ref/module.html) that provides [full-text search](http://en.wikipedia.org/wiki/Full_text_search) functionality to database applications.

To use FTS5, the user creates an FTS5 virtual table with one or more columns. For example:

```
CREATE VIRTUAL TABLE email USING fts5(sender, title, body);
```
