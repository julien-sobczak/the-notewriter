CREATE TABLE file (
  oid TEXT PRIMARY KEY,

  -- Parent file
  file_oid TEXT,

  -- Slug
  slug TEXT,

  -- Relative file path to the file
  relative_path TEXT NOT NULL,
  -- The full wikilink to this note
  wikilink TEXT NOT NULL,

  -- YAML document representing the Front Matter
  front_matter TEXT NOT NULL,

  -- Merged attributes in JSON
  attributes TEXT NOT NULL,

  -- Title including the kind but not the Markdown heading characters
  title TEXT NOT NULL,
  -- Same as title without the optional kind
  short_title TEXT NOT NULL,

  -- Body file content
  body TEXT NOT NULL,
  body_line INTEGER NOT NULL,

  -- Timestamps to track changes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
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

  -- Slug
  slug TEXT,

  -- Type of note: free, reference, ...
  kind TEXT NOT NULL,

  -- The relative path of the file containing the note (denormalized field)
  relative_path TEXT NOT NULL,
  -- The full wikilink to this note
  wikilink TEXT NOT NULL,

  -- Title including the kind but not the Markdown heading characters
  title TEXT NOT NULL,
  -- Same as short_title prefix by parent note/file's short titles.
  long_title TEXT NOT NULL,
  -- Same as title without the kind
  short_title TEXT NOT NULL,

  -- Merged attributes in JSON
  attributes TEXT NOT NULL,

  -- Merged tags in a comma-separated list
  tags TEXT NOT NULL,

  -- Line number (1-based index) of the note section title
  "line" INTEGER NOT NULL,

  -- Content without post-processing (including tags, attributes, ...)
  content TEXT NOT NULL,
  -- Hash of content_raw
  hashsum TEXT NOT NULL,
  -- Edited content in Markdown format
  body TEXT NOT NULL,
  -- Comment in Markdown format
  comment TEXT NOT NULL,

  -- Timestamps to track changes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_checked_at TEXT
);

CREATE VIRTUAL TABLE note_fts USING FTS5(oid UNINDEXED, kind UNINDEXED, short_title, content, content='note', content_rowid='rowid');

create trigger note_fts_after_insert after insert on note begin
  insert into note_fts (rowid, oid, kind, short_title, content) values (new.rowid, new.oid, new.kind, new.short_title, new.content);
end;

create trigger note_fts_after_update after update on note begin
  insert into note_fts (note_fts, rowid, oid, kind, short_title, content) values('delete', old.rowid, old.oid, old.kind, old.short_title, old.content);
  insert into note_fts (rowid, oid, kind, short_title, content) values (new.rowid, new.oid, new.kind, new.short_title, new.content);
end;

create trigger note_fts_after_delete after delete on note begin
  insert into note_fts (note_fts, rowid, oid, kind, short_title, content) values('delete', old.rowid, old.oid, old.kind, old.short_title, old.content);
end;

CREATE TABLE media (
  oid TEXT PRIMARY KEY,

  -- Relative path
  relative_path TEXT NOT NULL,

  -- Type of media: unknown, audio, picture, video, document
  kind TEXT NOT NULL,

  -- Media not present on disk
  dangling INTEGER NOT NULL DEFAULT 0,

  -- Extension
  extension TEXT NOT NULL,

  -- Content last modification date
  mtime TEXT NOT NULL,

  -- Checksum
  hashsum TEXT NOT NULL,

  -- Size of the file
  size INTEGER NOT NULL,
  -- These attributes can be used to find unused and/or large files

  -- Permission of the file
  mode INTEGER NOT NULL,

  -- Timestamps to track changes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_checked_at TEXT
);

CREATE TABLE blob (
  oid TEXT PRIMARY KEY,

  -- Media
  media_oid TEXT NOT NULL,

  -- Media type
  mime TEXT NOT NULL,

  -- YAML document representing the media
  attributes TEXT NOT NULL,

  -- Comma separated list of tags
  tags TEXT DEFAULT ''
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

  -- Slug (denormalized field)
  slug TEXT,

  -- Comma separated list of tags
  tags TEXT DEFAULT '',

  -- Fields in Markdown
  front TEXT NOT NULL,
  back TEXT NOT NULL,

  -- SRS
  due_at TEXT, -- null = suspended card
	studied_at TEXT, -- null = never studied
	settings TEXT, -- JSON document containing settings for the SRS algorithm

  -- Timestamps to track changes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
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
  description TEXT NOT NULL,

  -- Tag value containig the formula to determine the next occurence
  tag TEXT NOT NULL,

  -- Timestamps to track progress
  last_performed_at TEXT,
  next_performed_at TEXT NOT NULL,

  -- Timestamps to track changes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_checked_at TEXT
);


CREATE TABLE relation (
	-- The source note OID that references the target note
  source_oid TEXT NOT NULL,
  source_kind TEXT NOT NULL,
	-- The target note OID that is referenced by the source note
  target_oid TEXT NOT NULL,
  target_kind TEXT NOT NULL,
	-- The kind of relation (to classify)
	"type" TEXT NOT NULL,

	PRIMARY KEY (source_oid, target_oid, type)
);
