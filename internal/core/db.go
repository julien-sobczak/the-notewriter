package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notetaker/pkg/resync"
)

var (
	// Lazy-load ensuring a single read
	dbOnce       resync.Once
	dbSingleton  *DB
	dbClientOnce resync.Once
)

type DB struct {
	client *sql.DB
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		dbSingleton = &DB{}
	})
	return dbSingleton
}

func (db *DB) Close() error {
	if db.client != nil {
		return db.client.Close()
	}
	return nil
}

func (db *DB) Client() *sql.DB {
	dbClientOnce.Do(func() {
		var err error
		config := CurrentConfig()
		dbSingleton.client, err = sql.Open("sqlite3", filepath.Join(config.RootDirectory, "database.db"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		// TODO run migrations
	})
	return dbSingleton.client
}
