CREATE TABLE file (
    id INTEGER PRIMARY KEY,

    -- Relative file path to the file
    filepath TEXT NOT NULL,

    -- JSON document representing the Front Matter
    front_matter TEXT NOT NULL,

    -- Raw file content
    content TEXT NOT NULL,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    -- Last modification of local file on disk
    mtime TEXT NOT NULL
);

CREATE TABLE note (
    id INTEGER PRIMRAY KEY,

    -- File containing the note
    file_id INTEGER NOT NULL,

    -- Type of note:
    --    1 Note
    --    2 Flashcard
    --    3 Cheatsheet
    --    4 Quote
    --    5 Journal
    kind INTEGER NOT NULL,

    -- The filepath of the file containing the note (denormalized field)
    filepath TEXT NOT NULL,

    -- Merged Front Matter containing file attributes + note-specific attributes
    front_matter TEXT NOT NULL,

    -- Comma-separated list of tags
    tags TEXT NOT NULL,

    -- Line number (1-based index) of the note section title
    "line" INTEGER NOT NULL,

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

    FOREIGN KEY(file_id) REFERENCES file(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE VIRTUAL TABLE note_fts USING FTS5(id UNINDEXED, kind UNINDEXED, content_text);
-- TODO add other fields? Contentless table?

CREATE TABLE media (
    id INTEGER PRIMARY KEY,

    -- Relative path
    filepath TEXT NOT NULL,

    -- Type of media
    --    1 audio
    --    2 picture
    --    3 document
    kind INTEGER NOT NULL,

    -- Extension
    extension TEXT NOT NULL,

    -- Content last modification date
    mtime TEXT NOT NULL,

    -- Checksum
    hash TEXT NOT NULL,

    -- How many notes references this file
    links INTEGER NOT NULL DEFAULT 0,
    -- Size of the file
    size INTEGER NOT NULL,
    -- These attributes can be used to find unused and/or large files

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE link (
    id INTEGER PRIMARY KEY,

    -- Note representing the link
    note_id INTEGER NOT NULL,

    "text" TEXT NOT NULL,

    url TEXT NOT NULL,

    title TEXT,

    goName TEXT,

    -- Timestamps to track changes
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY(note_id) REFERENCES note(id) ON DELETE CASCADE ON UPDATE CASCADE
    -- TODO add filepath? line? absolute path?
);
-- Ex (skills/node.md): [Link 2](https://docs.npmjs.com "Tutorial to creating Node.js modules #go/node/module")
-- insert into link(1, 'Link 2', 'https://docs.npmjs.com', 'Tutorial to creating Node.js', 'node/module', 'skills/node.md')

CREATE TABLE flashcard (
	id INTEGER PRIMARY KEY,

    -- Note representing the flashcard
    note_id INTEGER NOT NULL,

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
    --    new: note id or random int
    --    due: integer day, relative to the collection's creation time
    --    learning: integer timestamp in second
    due INTEGER NOT NULL DEFAULT 0,

    -- The interval. Negative = seconds, positive = days
    ivl INTEGER NOT NULL DEFAULT 0,

    -- The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
    factor INTEGER NOT NULL DEFAULT 0,

    -- The number of reviews.
    reps INTEGER NOT NULL DEFAULT 0,

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

    FOREIGN KEY(note_id) REFERENCES note(id) ON DELETE CASCADE ON UPDATE CASCADE
);
-- TODO add custom template name?


