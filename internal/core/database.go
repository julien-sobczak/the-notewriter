package core

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/julien-sobczak/the-notetaker/pkg/resync"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

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
		config := CurrentConfig()
		db, err := sql.Open("sqlite3", filepath.Join(config.RootDirectory, ".nt/database.db"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		dbSingleton.client = db

		instance, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			log.Fatal(err)
		}

		// Run migrations
		d, err := iofs.New(migrationsFS, "sql")
		if err != nil {
			log.Fatalf("Error while reading migrations: %v", err)
		}
		m, err := migrate.NewWithInstance("iofs", d, "sqlite3", instance)
		if err != nil {
			log.Fatalf("Error while initializing migrations: %v", err)
		}

		err = m.Up() // Create/Update table schema_migrations
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Error while running migrations: %v", err)
		}
	})
	return dbSingleton.client
}

// Add adds new objects to staging area.
func (d *DB) Add(paths ...string) error {
	// TODO
	// Check paths are located inside the root directory
	// Check for .|file|dir
	return nil
}

// Commit creates a new commit object and clear the staging area.
func (d *DB) Commit() error {
	// TODO
	return nil
}

// Pull retrieves remote objects.
func (d *DB) Pull() error {
	// TODO
	// check refs/origin exists
	return nil
}

// Push pushes new objects remotely.
func (d *DB) Push() error {
	// TODO
	// check refs/origin exists
	return nil
}

func (d *DB) Restore() error {
	// TODO
	return nil
}
