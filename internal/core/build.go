package core

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type BuildResult struct {
	Files []*File
}

func (c *Collection) Build(outputDirectory string) error {

	config := CurrentConfig()

	log.Printf("Reading %s...\n", config.RootDirectory)
	filepath.WalkDir(config.RootDirectory, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err) // FIXME not visible in stderr
			return err
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}
		if dirname == ".git" {
			return fs.SkipDir
		}

		relpath := strings.TrimPrefix(path, config.RootDirectory+"/")

		if !config.IgnoreFile.Include(relpath) {
			return nil
		}

		// We look for only specific extension
		if !info.IsDir() && !config.ConfigFile.SupportExtension(relpath) {
			// Nothing to do
			return nil
		}

		// Ignore certain file modes like symlinks
		fileInfo, err := os.Lstat(path) // NB: os.Stat follows symlinks
		if err != nil {
			// Ignore the file
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			// Exclude any file with a mode bit set (device, socket, named pipe, ...)
			// See https://pkg.go.dev/io/fs#FileMode
			return nil
		}

		// Process file
		log.Printf("Processing %s...\n", path) // TODO emit notif for tests? + Parse file
		file, err := NewFileFromPath(path)
		if err != nil {
			return err
		}
		err = file.Save()
		if err != nil {
			return err
		}

		return nil
	})

	// DemoPopulateData(c.db)
	// DemoQueryNotes(c.db, "tutorial")

	return nil
}

func (c *Collection) Update(buildResult *BuildResult) error {
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

func DemoPopulateData(db *sql.DB) {
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

func DemoQueryNotes(db *sql.DB, queryTxt string) {
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
