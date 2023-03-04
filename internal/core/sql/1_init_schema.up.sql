CREATE TABLE collection (
    oid TEXT PRIMARY KEY,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_checked_at TEXT
);

CREATE TABLE file (
    oid TEXT PRIMARY KEY,

    -- Relative file path to the file
    relative_path TEXT NOT NULL,
    -- The full wikilink to this note
    wikilink TEXT NOT NULL,

    -- JSON document representing the Front Matter
    front_matter TEXT NOT NULL,

    -- Raw file content
    content TEXT NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT,

    -- Last modification of local file on disk
    mtime TEXT NOT NULL,
    size INTEGER NOT NULL,
    hashsum TEXT NOT NULL,
    mode INTEGER NOT NULL
);

CREATE TABLE note (
    oid TEXT PRIMARY KEY,

    -- File containing the note
    file_oid TEXT NOT NULL,

    -- Optional parent note containing the note
    note_oid TEXT,

    -- Type of note:
    --    0 Free (not persisted for now)
    --    1 Reference
    --    2 Note
    --    3 Flashcard
    --    4 Cheatsheet
    --    5 Quote
    --    6 Journal
    --    7 TODO
    --    8 Artwork
    --    9 Snippet
    kind INTEGER NOT NULL,

    -- The relative path of the file containing the note (denormalized field)
    relative_path TEXT NOT NULL,
    -- The full wikilink to this note
    wikilink TEXT NOT NULL,

    -- Title including the kind but not the Markdown heading characters
    title TEXT NOT NULL,

    -- Same as title without the kind
    short_title TEXT NOT NULL,

    -- Merged attributes
    attributes_yaml TEXT NOT NULL,
    attributes_json TEXT NOT NULL,

    -- Comma-separated list of tags
    tags TEXT NOT NULL,

    -- Line number (1-based index) of the note section title
    "line" INTEGER NOT NULL,

    -- Content without post-prcessing (including tags, attributes, ...)
    content_raw TEXT NOT NULL,
    -- Hash of content_raw
    hashsum TEXT NOT NULL,
    -- Content in Markdown format (best for editing)
    content_markdown TEXT NOT NULL,
    -- Content in HTML format (best for rendering)
    content_html TEXT NOT NULL,
    -- Content in raw text (best for indexing)
    content_text TEXT NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT
);

CREATE VIRTUAL TABLE note_fts USING FTS5(kind UNINDEXED, short_title, content_text, content='note', content_rowid='rowid');

create trigger note_fts_after_insert after insert on note begin
  insert into note_fts (rowid, kind, short_title, content_text) values (new.rowid, new.kind, new.short_title, new.content_text);
end;

create trigger note_fts_after_update after update on note begin
  insert into note_fts (note_fts, rowid, kind, short_title, content_text) values('delete', old.rowid, old.kind, old.short_title, old.content_text);
  insert into note_fts (rowid, kind, short_title, content_text) values (new.rowid, new.kind, new.short_title, new.content_text);
end;

create trigger note_fts_after_delete after delete on note begin
  insert into note_fts (note_fts, rowid, kind, short_title, content_text) values('delete', old.rowid, old.kind, old.short_title, old.content_text);
end;

CREATE TABLE media (
    oid TEXT PRIMARY KEY,

    -- Relative path
    relative_path TEXT NOT NULL,

    -- Type of media
    --    0 unknown
    --    1 audio
    --    2 picture
    --    3 video
    --    4 document
    kind INTEGER NOT NULL,

    -- Media not present on disk
    dangling INTEGER NOT NULL DEFAULT 0,

    -- Extension
    extension TEXT NOT NULL,

    -- Content last modification date
    mtime TEXT NOT NULL,

    -- Checksum
    hashsum TEXT NOT NULL,

    -- How many notes references this file
    links INTEGER NOT NULL DEFAULT 0,

    -- Size of the file
    size INTEGER NOT NULL,
    -- These attributes can be used to find unused and/or large files

    -- Permission of the file
    mode INTEGER NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT
);

CREATE TABLE link (
    oid TEXT PRIMARY KEY,

    -- Note representing the link
    note_oid TEXT NOT NULL,

    -- The relative path of the file containing the note (denormalized field)
    relative_path TEXT NOT NULL,

    "text" TEXT NOT NULL,

    url TEXT NOT NULL,

    title TEXT,

    go_name TEXT,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT
);
-- Ex (skills/node.md): [Link 2](https://docs.npmjs.com "Tutorial to creating Node.js modules #go/node/module")
-- insert into link(1, 'Link 2', 'https://docs.npmjs.com', 'Tutorial to creating Node.js', 'node/module', 'skills/node.md')

CREATE TABLE flashcard (
	  oid TEXT PRIMARY KEY,

    -- File representing the flashcard
    file_oid TEXT NOT NULL,

    -- Note representing the flashcard
    note_oid TEXT NOT NULL,

    -- The relative path of the file containing the note (denormalized field)
    relative_path TEXT NOT NULL,

    -- Note short title
    short_title TEXT NOT NULL,

    -- Comma separated list of tags
    tags TEXT DEFAULT '',

    -- 0=new, 1=learning, 2=review, 3=relearning
    "type" INTEGER NOT NULL DEFAULT 0,

    -- Queue types:
    --   -1=suspend     => leeches as manual suspension is not supported
    --    0=new         => new (never shown)
    --    1=(re)lrn     => learning/relearning
    --    2=rev         => review (as for type)
    --    3=day (re)lrn => in learning, next review in at least a day after the previous review
    queue INTEGER NOT NULL DEFAULT 0,

    -- Due is used differently for different card types:
    --    new: note oid or random int
    --    due: integer day, relative to the collection's creation time
    --    learning: integer timestamp in second
    due INTEGER NOT NULL DEFAULT 0,

    -- The interval. Negative = seconds, positive = days
    ivl INTEGER NOT NULL DEFAULT 0,

    -- The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
    ease_factor INTEGER NOT NULL DEFAULT 0,

    -- The number of reviews.
    repetitions INTEGER NOT NULL DEFAULT 0,

    -- The number of times the card went from a "was answered correctly" to "was answered incorrectly" state.
    lapses INTEGER NOT NULL DEFAULT 0,

    -- Of the form a*1000+b, with:
    --    a the number of reps left today
    --    b the number of reps left till graduation
    --    for example: '2004' means 2 reps left today and 4 reps till graduation
    left INTEGER NOT NULL DEFAULT 0,

    -- Fields in Markdown (best for editing)
    front_markdown TEXT NOT NULL,
    back_markdown TEXT NOT NULL,

    -- Fields in HTML (best for rendering)
    front_html TEXT NOT NULL,
    back_html TEXT NOT NULL,

    -- Fields in raw text (best for indexing)
    front_text TEXT NOT NULL,
    back_text TEXT NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT
);


CREATE TABLE reminder (
	  oid TEXT PRIMARY KEY,

    -- File representing the flashcard
    file_oid TEXT NOT NULL,

    -- Note representing the flashcard
    note_oid TEXT NOT NULL,

    -- The relative path of the file containing the note (denormalized field)
    relative_path TEXT NOT NULL,

    -- Description
    description_raw TEXT NOT NULL,
    description_markdown TEXT NOT NULL,
    description_html TEXT NOT NULL,
    description_text TEXT NOT NULL,

    -- Tag value containig the formula to determine the next occurence
    tag TEXT NOT NULL,

    -- Timestamps to track progress
    last_performed_at TEXT,
    next_performed_at TEXT NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    last_checked_at TEXT
);
