package core

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type BuildResult struct {
	files []*File
}

func (c *Collection) Build(outputDirectory string) error {

	populateData(c.db)
	queryNotes(c.db, "tutorial")

	return nil
}

func (c *Collection) update(buildResult *BuildResult) error {
	now := time.Now()
	fmt.Printf("%v\n", now)
	// Update all tables and their timestamps
	// + generate replication log + packfiles (only if change, a rebuild must not trigger new files)
	// TODO Mark as stale old records (deleted_at)
	// buildResult.files
	// file.GetNotes()
	// note.GetMedias()
	// media.GetExtension()
	// media.GetHash()
	// media.GetSize()

	// note.GetLinks() // or file.GetLinks()
	// note.AsMarkdown()
	// note.AsHTML()
	// note.AsText()

	return nil
}

func populateData(db *sql.DB) {
	records := `INSERT INTO posts(title, body) VALUES
('Learn SQlite FTS5', 'This tutorial teaches you how to perform full-text search in SQLite using FTS5'),
('Advanced SQlite Full-text Search', 'Show you some advanced techniques in SQLite full-text searching'),
('SQLite Tutorial', 'Help you learn SQLite quickly and effectively');`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec()
	if err != nil {
		log.Fatal(err)
	}
}

func queryNotes(db *sql.DB, queryTxt string) {
	queryFTS, err := db.Prepare("SELECT id FROM note_fts WHERE kind = 1 and note_fts MATCH ? ORDER BY rank LIMIT 10;")
	if err != nil {
		log.Fatal(err)
	}
	recordFTS, err := queryFTS.Query(queryTxt)
	if err != nil {
		log.Fatal(err)
	}
	defer recordFTS.Close()
	var ids []int
	for recordFTS.Next() {
		var id int
		recordFTS.Scan(&id)
		ids = append(ids, id)
	}

	query, err := db.Prepare("SELECT id, content_markdown FROM note WHERE id IN (?);")
	if err != nil {
		log.Fatal(err)
	}
	record, err := query.Query(queryTxt, ids)
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		var id int
		var content string
		record.Scan(&id)
		record.Scan(&content)
		fmt.Printf("%d\n%s\n\n", id, content)
	}
}
