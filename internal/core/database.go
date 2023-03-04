package core

import (
	"bytes"
	"context"
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
	// .nt/objects/info/commit-graph
	commitGraph *CommitGraph
	// .nt/refs/*
	refs map[string]string
	// .nt/refs/origin
	origin Remote
	// .nt/database.sql
	client *sql.DB
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		// Load index
		index := ReadIndex()

		// Load refs
		refs := ReadRefs()

		// Load the commit graph
		commitGraph := ReadCommitGraph()

		// Create the database
		dbSingleton = &DB{
			index:       index,
			commitGraph: commitGraph,
			refs:        refs,
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

func ReadCommitGraph() *CommitGraph {
	cg, err := NewCommitGraphFromPath(CurrentCollection().GetAbsolutePath(".nt/objects/info/commit-graph"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/objects/info/commit-graph file: %v", err)
		os.Exit(1)
	}
	return cg
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
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := c.Read(in); err != nil {
		return nil, err
	}
	return c, nil
}

// ReadBlob reads a blob file on disk.
func (db *DB) ReadBlob(oid string) ([]byte, error) {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	return os.ReadFile(path)
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
			remote, err := NewS3RemoteWithCredentials(configRemote.Endpoint, configRemote.BucketName, configRemote.AccessKey, configRemote.SecretKey)
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

func (db *DB) AddBlob(raw []byte, blob BlobRef) error {
	// TODO where to save blob OID? Must be in objects in commit files. New column? ✅
	return nil
}

func (db *DB) StageObject(obj StatefulObject) error {
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

	// Update the commit graph
	if err := db.commitGraph.AppendCommit(c.OID); err != nil {
		return err
	}
	if err := db.commitGraph.Save(); err != nil {
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
	origin := db.Origin()
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read remote's commit-graph to find new commits to pull
	data, err := origin.GetObject("info/commit-graph")
	if errors.Is(err, ErrObjectNotExist) {
		// Nothing to pull
		return nil
	}
	cg := new(CommitGraph)
	if err := cg.Read(bytes.NewReader(data)); err != nil {
		return err
	}

	// Iterate over missing commits
	commitOIDs := db.commitGraph.MissingCommitsFrom(cg)
	for _, oid := range commitOIDs {

		// Download each commit in a single transaction
		tx, err := db.Client().BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		data, err = origin.GetObject(OIDToPath(oid))
		if errors.Is(err, ErrObjectNotExist) {
			return fmt.Errorf("missing commit %q", oid)
		} else if err != nil {
			return err
		}

		commit := new(Commit)
		if err := commit.Read(bytes.NewReader(data)); err != nil {
			return err
		}

		for _, commitObject := range commit.Objects {
			remoteObject := commitObject.ReadObject()

			for _, blobRef := range remoteObject.Blobs() {
				// Check if blob exists
				blobPath := OIDToPath(blobRef.OID)
				if db.BlobExists(blobRef.OID) {
					continue
				}

				// Download the file
				blobData, err := origin.GetObject(blobPath)
				if err != nil {
					return err
				}
				blobFile := new(BlobFile)
				blobFile.Ref = blobRef
				if err := blobFile.Read(bytes.NewReader(blobData)); err != nil {
					return err
				}
				if err := blobFile.Save(); err != nil {
					return err
				}
			}

			newState := db.determineState(commitObject)
			remoteObject.ForceState(newState)

			// Add in SQL database
			remoteObject.Save(tx)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		db.commitGraph.AppendCommit(commit.OID)

		// Write on disk at the disk
		if err := commit.Save(); err != nil {
			return fmt.Errorf("unable to write retrieved commit %q after processing: %v", oid, err)
		}

		// Update the main ref
		db.updateRef("main", commit.OID)
	}

	// Keep note of last origin retrieved commit
	db.updateRef("origin", cg.Ref())

	return nil
}

// Push pushes new objects remotely.
func (db *DB) Push() error {
	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteTaker is a personal tool and you are not expected to push from two repositories at the same time.

	origin := db.Origin()
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read remote's commit-graph to find new commits to pull
	data, err := origin.GetObject("info/commit-graph")
	if errors.Is(err, ErrObjectNotExist) {
		// Nothing to pull
		return nil
	}
	cg := new(CommitGraph)
	if err := cg.Read(bytes.NewReader(data)); err != nil {
		return err
	}

	// Iterate over missing commits
	commitOIDs := cg.MissingCommitsFrom(db.commitGraph)
	for _, commitOID := range commitOIDs {

		commit, err := db.ReadCommit(commitOID)
		if err != nil {
			return err
		}

		// Upload blobs first (if the commit upload fails, it will be retried at least)
		for _, commitObject := range commit.Objects {
			object := commitObject.ReadObject()
			for _, blobRef := range object.Blobs() {
				blobData, err := db.ReadBlob(blobRef.OID)
				if err != nil {
					return err
				}
				if err := origin.PutObject(OIDToPath(blobRef.OID), blobData); err != nil {
					return err
				}
			}
		}

		// Upload the commit
		buf := new(bytes.Buffer)
		if err := commit.Write(buf); err != nil {
			return err
		}
		if err := origin.PutObject(OIDToPath(commitOID), buf.Bytes()); err != nil {
			return err
		}
	}

	// Update remote commit-graph
	buf := new(bytes.Buffer)
	if err := db.commitGraph.Write(buf); err != nil {
		return err
	}
	if err := origin.PutObject("info/commit-graph", buf.Bytes()); err != nil {
		return err
	}

	// Update the origin ref
	db.updateRef("origin", db.refs["main"])

	return nil
}

func (db *DB) Restore() error {
	// Run all queries inside the same transaction
	tx, err := db.Client().BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// We must clear the staging area
	for _, obj := range db.index.StagingArea.Added {
		object := obj.ReadObject()
		if object == nil {
			return fmt.Errorf("unknown object %q", obj.OID)
		}
		// Mark for deletion
		object.ForceState(Deleted)
		if err := object.Save(tx); err != nil {
			return err
		}
	}
	for _, obj := range db.index.StagingArea.Deleted {
		// Re-read object from latest commit
		parentCommit, err := db.ReadCommit(obj.PreviousCommitOID)
		if err != nil {
			return fmt.Errorf("missing parent commit %q: %v", obj.PreviousCommitOID, err)
		}
		original, found := parentCommit.GetCommitObject(obj.OID)
		if !found {
			return fmt.Errorf("missing object %q in commit %s", obj.OID, obj.PreviousCommitOID)
		}
		originalObject := original.ReadObject()
		if originalObject == nil {
			return fmt.Errorf("unknown object %q", obj.OID)
		}
		// Mark for restoration
		originalObject.ForceState(Added)
		originalObject.Save(tx)
	}
	for _, obj := range db.index.StagingArea.Modified {
		// Re-read object from latest commit
		parentCommit, err := db.ReadCommit(obj.PreviousCommitOID)
		if err != nil {
			return fmt.Errorf("missing parent commit %q: %v", obj.PreviousCommitOID, err)
		}
		original, found := parentCommit.GetCommitObject(obj.OID)
		if !found {
			return fmt.Errorf("missing object %q in commit %s", obj.OID, obj.PreviousCommitOID)
		}
		originalObject := original.ReadObject()
		if originalObject == nil {
			return fmt.Errorf("unknown object %q", obj.OID)
		}
		// Nothing to change. Simply save back.
		originalObject.ForceState(Modified)
		originalObject.Save(tx)
	}

	// Don't forget to commit
	if err := tx.Commit(); err != nil {
		return err
	}
	// And to persist the index
	db.index.StagingArea.Added = nil
	db.index.StagingArea.Deleted = nil
	db.index.StagingArea.Modified = nil
	err = db.index.Save()

	return err
}

/* Utility */

// Ref returns the commit OID for the given ref
func (db *DB) Ref(name string) (string, bool) {
	value, ok := db.refs[name]
	return value, ok
}

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

// BlobExists checks if a blob exists locally.
func (db *DB) BlobExists(oid string) bool {
	return db.fileExists(oid)
}

// ObjectExists checks if a blob exists locally.
func (db *DB) ObjectExists(oid string) bool {
	return db.fileExists(oid)
}

// BlobExists checks if a blob exists locally.
func (db *DB) fileExists(oid string) bool {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (db *DB) determineState(commitObject *CommitObject) State {
	indexObject, found := db.index.objectsRef[commitObject.OID]

	// The remote object doesn't exist locally
	if !found {
		if commitObject.State != Deleted {
			return Added
		}
		return None
	}

	// The remote object exist locally
	if indexObject.MTime.After(commitObject.MTime) {
		// we have a more recent version
		return None
	}
	return Modified
}
