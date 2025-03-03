package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Lite version of internal/core/database.go

const schema = `
CREATE TABLE file (
	oid TEXT PRIMARY KEY,
	packfile_oid TEXT NOT NULL,
	relative_path TEXT NOT NULL,
	body TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	indexed_at TEXT,
	mtime TEXT NOT NULL,
	size INTEGER NOT NULL,
	hashsum TEXT NOT NULL
);

CREATE TABLE note (
	oid TEXT PRIMARY KEY,
	packfile_oid TEXT NOT NULL,
	file_oid TEXT NOT NULL,
	relative_path TEXT NOT NULL,
	title TEXT NOT NULL,
	content_raw TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	indexed_at TEXT
);
`

var (
	dbOnce      sync.Once
	dbSingleton *DB
)

type DB struct {
	// .nt/index
	index *Index
	// .nt/database.sql
	client *sql.DB
	// In-progress transaction
	tx *sql.Tx
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		dbSingleton = &DB{
			index:  ReadIndex(),
			client: InitClient(),
		}
	})
	return dbSingleton
}

func CurrentIndex() *Index {
	return CurrentDB().index
}

// Client returns the client to use to query the database.
func (db *DB) Client() SQLClient {
	if db.tx != nil {
		// Execute queries in current transaction
		return db.tx
	}
	// Basic client = no transaction
	return db.client
}

func InitClient() *sql.DB {
	db, err := sql.Open("sqlite3", filepath.Join(CurrentRepository().Path, ".nt/database.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Create schema
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("Error while initializing database: %v", err)
	}

	return db
}

// Queryable provides a common interface between sql.DB and sql.Tx to make methods compatible with both.
type SQLClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

/* Transaction Management */

// BeginTransaction starts a new transaction.
func (db *DB) BeginTransaction() error {
	tx, err := db.client.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	db.tx = tx
	return nil
}

// RollbackTransaction aborts the current transaction.
func (db *DB) RollbackTransaction() error {
	if db.tx == nil {
		return errors.New("no transaction started")
	}
	err := db.tx.Rollback()
	db.tx = nil
	return err
}

// CommitTransaction ends the current transaction.
func (db *DB) CommitTransaction() error {
	if db.tx == nil {
		return errors.New("no transaction started")
	}
	err := db.tx.Commit()
	if err != nil {
		return err
	}
	db.tx = nil
	return nil
}

// UpsertPackFiles inserts or updates pack files in the database.
func (db *DB) UpsertPackFiles(packFiles ...*PackFile) error {
	for _, packFile := range packFiles {
		for _, object := range packFile.PackObjects {
			obj := object.ReadObject()
			if statefulObj, ok := obj.(StatefulObject); ok {
				if err := statefulObj.Save(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
