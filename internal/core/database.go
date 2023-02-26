package core

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	dbRemoteOnce resync.Once
	dbClientOnce resync.Once
)

type DB struct {
	// .nt/index
	index *Index
	// .nt/refs/origin
	origin Remote
	// .nt/database.sql
	client *sql.DB
	// .nt/refs/*
	refs map[string]string
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		// Load index
		index := ReadIndex()

		// Load refs
		refs := ReadRefs()

		// Create the database
		dbSingleton = &DB{
			index: index,
			refs:  refs,
		}
	})
	return dbSingleton
}

func ReadIndex() *Index {
	index, err := NewIndexFromPath(CurrentCollection().GetAbsolutePath(".nt/index"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/index file: %v", err)
		os.Exit(1)
	}
	return index
}

func ReadRefs() map[string]string {
	refs := make(map[string]string)
	refdir := CurrentCollection().GetAbsolutePath(".nt/refs")
	files, err := os.ReadDir(refdir)
	if os.IsNotExist(err) {
		// No existing refs (occurs before the first commit)
		return refs
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read refs under .nt/refs directory: %v", err)
		os.Exit(1)
	}
	for _, file := range files {
		if !file.IsDir() {
			data, err := os.ReadFile(filepath.Join(refdir, file.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to read .nt/refs/%s directory: %v", file.Name(), err)
				os.Exit(1)
			}
			refs[file.Name()] = strings.TrimSpace(string(data))
		}
	}
	return refs
}

func (db *DB) Close() error {
	if db.client != nil {
		return db.client.Close()
	}
	return nil
}

// ReadCommit reads an object file on disk.
func (db *DB) ReadCommit(oid string) (*Commit, error) {
	c := new(Commit)
	filepath := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", oid[0:2], oid)
	in, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := c.Read(in); err != nil {
		return nil, err
	}
	return c, nil
}

// Origin returns the origin implementation based on the optional configured type.
func (db *DB) Origin() Remote {
	dbRemoteOnce.Do(func() {
		config := CurrentConfig()
		configRemote := config.ConfigFile.Remote
		if configRemote.Type == "" {
			return
		}
		switch configRemote.Type {
		case "fs":
			remote, err := NewFSRemote(configRemote.Dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to init FS remote: %v\n", err)
				os.Exit(1)
			}
			db.origin = remote
		case "s3":
			remote, err := NewS3RemoteWithCredentials(configRemote.BucketName, configRemote.AccessKey, configRemote.SecretKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to init S3 remote: %v\n", err)
				os.Exit(1)
			}
			db.origin = remote
		default:
			fmt.Fprintf(os.Stderr, "Unknow remote type %q\n", configRemote.Type)
			os.Exit(1)
		}
	})
	return db.origin
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

func (db *DB) AddBlob(raw []byte, blob Blob) error {
	// TODO where to save blob OID? Must be in objects in commit files. New column? ✅
	return nil
}
func (db *DB) StageObject(obj Object) error {
	return db.index.StageObject(obj)
}

// Commit creates a new commit object and clear the staging area.
func (db *DB) Commit(msg string) error {
	changesAdded := len(db.index.StagingArea.Added)
	changesModified := len(db.index.StagingArea.Modified)
	changesDeleted := len(db.index.StagingArea.Deleted)
	changesTotal := changesAdded + changesModified + changesDeleted

	if changesTotal == 0 {
		return errors.New(`nothing to commit (create/copy files and use "nt add" to track)`)
	}

	// Convert the staging area to a new commit file
	c := db.index.CreateCommitFromStagingArea()
	if err := c.Save(); err != nil {
		return err
	}

	// Save updates staging area
	if err := db.index.Save(); err != nil {
		return err
	}

	// Update the main ref
	db.updateRef("main", c.OID)

	// Output result
	fmt.Printf("[%7s] %s\n", c.OID, msg)
	fmt.Printf(" %d objects changes", len(c.Objects))
	if changesAdded > 0 {
		fmt.Printf(", %d insertion(s)", changesAdded)
	}
	if changesModified > 0 {
		fmt.Printf(", %d modification(s)", changesModified)
	}
	if changesDeleted > 0 {
		fmt.Printf(", %d deletion(s)", changesDeleted)
	}
	fmt.Println()
	for _, obj := range c.Objects {
		action := "modify"
		switch obj.State {
		case Added:
			action = "create"
		case Deleted:
			action = "delete"
		}
		fmt.Printf(" %s %s\n", action, obj.Description)
	}
	return nil
}

// Pull retrieves remote objects.
func (db *DB) Pull() error {
	// TODO
	// check refs/origin exists
	return nil
}

// Push pushes new objects remotely.
func (db *DB) Push() error {
	// TODO
	// check refs/origin exists
	return nil
}

func (db *DB) Restore() error {
	// TODO
	return nil
}

/* Utility */

// updateRef updates a ref with a new commit OID.
func (db *DB) updateRef(name, commitOID string) error {
	refdir := filepath.Join(CurrentConfig().RootDirectory, ".nt/refs/")
	if err := os.MkdirAll(refdir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(refdir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(commitOID)
	return err
}
