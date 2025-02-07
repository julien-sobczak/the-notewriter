package core

import (
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
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
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
	// .nt/refs/*
	refs map[string]string
	// .nt/refs/origin
	origin Remote
	// .nt/database.sql
	client *sql.DB

	// In-progress transaction
	tx *sql.Tx
}

func CurrentDB() *DB {
	dbOnce.Do(func() {
		// Load index
		index := MustReadIndex()

		// Load refs
		refs := MustReadRefs()

		// Create the database
		dbSingleton = &DB{
			index: index,
			refs:  refs,
		}
	})
	return dbSingleton
}

func CurrentIndex() *Index {
	return CurrentDB().Index()
}

func (db *DB) initClient() *sql.DB {
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

func MustReadRefs() map[string]string {
	refs := make(map[string]string)
	refdir := CurrentRepository().GetAbsolutePath(".nt/refs")
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

func (db *DB) Index() *Index {
	return db.index
}

/* Transaction Management */

// BeginTransaction starts a new transaction.
func (db *DB) BeginTransaction() error {
	tx, err := db.initClient().BeginTx(context.Background(), nil)
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

// Client returns the client to use to query the database.
func (db *DB) Client() SQLClient {
	if db.tx != nil {
		// Execute queries in current transaction
		return db.tx
	}
	// Basic client = no transaction
	return db.initClient()
}

/* PackFile Management */

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

// DeletePackFiles removes pack files from the database.
func (db *DB) DeletePackFiles(packFiles ...*PackFile) error {
	for _, packFile := range packFiles {
		for _, object := range packFile.PackObjects {
			obj := object.ReadObject()
			if statefulObj, ok := obj.(StatefulObject); ok {
				if err := statefulObj.Delete(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

/*
 * Object Management
 */

// ReadPackFile reads a pack file on disk.
func (db *DB) ReadPackFileOnDisk(oid oid.OID) (*PackFile, error) {
	result := new(PackFile)
	in, err := os.Open(PackFilePath(oid))
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := result.Read(in); err != nil {
		return nil, err
	}
	return result, nil
}

// WritePackFileOnDisk writes a blob file on disk
func (db *DB) WritePackFileOnDisk(packFile *PackFile) error {
	if err := packFile.Save(); err != nil {
		return err
	}
	CurrentLogger().Infof("💾 Saved pack file %s", filepath.Base(packFile.ObjectPath()))
	return nil
}

// DeletePackFileOnDisk removes a single pack file on disk
func (db *DB) DeletePackFileOnDisk(packFile PackFileRef) error {
	err := os.Remove(packFile.ObjectPath())
	if err != nil {
		return err
	}
	CurrentLogger().Infof("💾 Deleted pack file %s", filepath.Base(packFile.ObjectPath()))
	return nil
}

// ReadBlobOnDisk reads a blob file on disk.
func (db *DB) ReadBlobOnDisk(oid oid.OID) ([]byte, error) {
	return os.ReadFile(BlobPath(oid))
}

// WriteBlobOnDisk writes a blob file on disk
func (db *DB) WriteBlobOnDisk(oid oid.OID, data []byte) error {
	path := BlobPath(oid)
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	CurrentLogger().Infof("💾 Saved blob %s", filepath.Base(path))
	return nil
}

// DeleteBlobOnDisk removes a blob on disk
func (db *DB) DeleteBlobOnDisk(blob BlobRef) error {
	err := os.Remove(blob.ObjectPath())
	if err != nil {
		return err
	}
	CurrentLogger().Infof("💾 Deleted blob %s", filepath.Base(blob.ObjectPath()))
	return nil
}

/*
 * Remote Management
 */

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
			remote, err := NewS3RemoteWithCredentials(configRemote.Endpoint, configRemote.BucketName, configRemote.AccessKey, configRemote.SecretKey, configRemote.Secure)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to init S3 remote: %v\n", err)
				os.Exit(1)
			}
			db.origin = remote
		case "storj":
			remote, err := NewStorjRemoteWithCredentials(configRemote.BucketName, configRemote.AccessKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to init Storj remote: %v\n", err)
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

// Diff show the changes in the staging area.
func (db *DB) Diff() (string, error) {
	var diff strings.Builder

	// TODO implement

	return diff.String(), nil
}

// GC removes non referenced objects/blobs in the local directory.
func (db *DB) GC() error {
	// Why GC is required? Why commits cannot do the housekeeping directly?
	//
	// The main reason is to reclaim disk space (and thus limit the storage consumption, especially useful for remotes).
	//
	// For example:
	//
	// * Notes can embed medias. Notes can later be rewritten and no longer reference this media. The media is maybe
	//   referenced by another note. It's not easy to find out when adding the edited note.
	//   The GC searches for all these orphan blobs at once.
	// * Pack files are created when running 'nt add' but are not deleted on disk when running 'nt reset' (only removed in index)
	//   to avoid recreating blobs (especially using for medias which require conversion).
	//   If a file never added again, the packfile can be safely removed.

	CurrentLogger().Info("Reclaiming blobs...")



	return nil
}

/* Utility */

// Ref returns the commit OID for the given ref
func (db *DB) Ref(name string) (string, bool) {
	value, ok := db.refs[name]
	return value, ok
}

// BlobExists checks if a blob exists locally.
func (db *DB) BlobExists(oid oid.OID) bool {
	_, err := os.Stat(BlobPath(oid))
	return !os.IsNotExist(err)
}

// PackFileExists checks if a blob exists locally.
func (db *DB) PackFileExists(oid oid.OID) bool {
	_, err := os.Stat(PackFilePath(oid))
	return !os.IsNotExist(err)
}

/* Stats */

type StatsOnDisk struct {
	// Number of files under .nt/objects
	ObjectFiles int
	// Number of commits in .nt/commit-graph
	Commits int
	// Number of blobs under .nt/objects
	Blobs int
	// Number of objects (file, note, etc.) present in commits
	Objects map[string]int
	// Number of objects listed in .nt/objects/index
	IndexObjects int
	// Total size of directory .nt/objects
	TotalSizeKB int64
}

func NewStatsOnDiskEmpty() *StatsOnDisk {
	return &StatsOnDisk{
		ObjectFiles: 0,
		Commits:     0,
		Blobs:       0,
		Objects: map[string]int{
			"file":      0,
			"note":      0,
			"flashcard": 0,
			"media":     0,
			"link":      0,
			"reminder":  0,
		},
		IndexObjects: 0,
		TotalSizeKB:  0,
	}
}

// StatsOnDisk returns various statistics about the .nt/objects directory.
func (db *DB) StatsOnDisk() (*StatsOnDisk, error) {
	objectsPath := CurrentIndex().ObjectsDir()

	// Ensure the objects directory exists
	if _, err := os.Stat(objectsPath); os.IsNotExist(err) {
		// Not exists (occurs before the first commit)
		return NewStatsOnDiskEmpty(), nil
	}

	files, err := filesystem.ListFiles(objectsPath)
	if err != nil {
		return nil, err
	}

	result := NewStatsOnDiskEmpty()

	for _, file := range files {
		oid := oid.MustParse(filepath.Base(file))

		result.ObjectFiles++

		if _, ok := db.index.GetEntryByPackFileOID(oid); ok {
			// It's a pack file, check the content to count objects/notes
			packFile, err := LoadPackFileFromPath(file)
			if err != nil {
				return nil, err
			}
			for _, object := range packFile.PackObjects {
				result.Objects[object.Kind]++
			}
		} else {
			// Must be a blob
			result.Blobs++
		}
	}

	totalSize, err := filesystem.DirSize(objectsPath)
	if err != nil {
		return nil, err
	}

	if db.index != nil {
		result.IndexObjects = len(db.index.Objects)
	}

	result.TotalSizeKB = totalSize / filesystem.KB

	return result, nil
}
