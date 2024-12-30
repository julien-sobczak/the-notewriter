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
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"golang.org/x/exp/slices"
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

type WIP struct {
	notes []*Note
}

func (w *WIP) Register(note *Note) {
	w.notes = append(w.notes, note)
}

func (w *WIP) FindNoteByWikilink(wikilink string) *Note {
	for _, note := range w.notes {
		if strings.HasSuffix(text.TrimExtension(note.Wikilink), text.TrimExtension(wikilink)) {
			return note
		}
	}
	return nil
}

func (w *WIP) Flush() {
	w.notes = nil
}

type DB struct {
	// Notes in progress
	wip *WIP

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

	// In-progress transaction
	tx *sql.Tx
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
			wip:         new(WIP),
			index:       index,
			commitGraph: commitGraph,
			refs:        refs,
		}
	})
	return dbSingleton
}

func ReadIndex() *Index {
	index, err := NewIndexFromPath(CurrentRepository().GetAbsolutePath(".nt/index"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/index file: %v", err)
		os.Exit(1)
	}
	return index
}

func ReadCommitGraph() *CommitGraph {
	cg, err := NewCommitGraphFromPath(CurrentRepository().GetAbsolutePath(".nt/objects/info/commit-graph"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/objects/info/commit-graph file: %v", err)
		os.Exit(1)
	}
	return cg
}

func ReadRefs() map[string]string {
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

// WIP returns the registry of currently processed notes.
func (db *DB) WIP() *WIP {
	return db.wip
}

/* Row Management */

// ReadLastStagedOrCommittedObjectFromDB reads the last known version (in staging or committed) by rereading the database.
func (db *DB) ReadLastStagedOrCommittedObjectFromDB(oid string) (StatefulObject, error) { // FIXME delete?
	var kind string

	if stagedObject, ok := db.index.StagingArea.ReadStagingObject(oid); ok {
		// Check staging area first
		kind = stagedObject.Kind
	} else if indexObject, ok := db.index.ReadIndexObject(oid); ok {
		// Check commits second
		kind = indexObject.Kind
	}

	if kind == "" {
		return nil, nil
	}

	switch kind {
	case "file":
		return CurrentRepository().LoadFileByOID(oid)
	case "note":
		return CurrentRepository().LoadNoteByOID(oid)
	case "flashcard":
		return CurrentRepository().LoadFlashcardByOID(oid)
	case "link":
		return CurrentRepository().LoadGoLinkByOID(oid)
	case "reminder":
		return CurrentRepository().LoadReminderByOID(oid)
	case "media":
		return CurrentRepository().LoadMediaByOID(oid)
	default:
		return nil, fmt.Errorf("unsupported kind %s when reading object", kind)
	}
}

/* File Management */

// ReadCommit checks for a commit with the given id.
func (db *DB) ReadCommit(oid string) (*Commit, bool) {
	for _, commit := range db.commitGraph.Commits {
		if commit.OID == oid {
			return commit, true
		}
	}
	return nil, false
}

// Head returns the latest commit or nil if no commit exists.
func (db *DB) Head() *Commit {
	if len(db.commitGraph.Commits) == 0 {
		return nil
	}
	return db.commitGraph.Commits[len(db.commitGraph.Commits)-1]
}

// ReadCommittedObject reads the last known committed version of stateful object on disk.
func (db *DB) ReadCommittedObject(oid string) (StatefulObject, error) {
	indexObject, ok := db.index.objectsRef[oid]
	if !ok {
		return nil, nil
	}
	packFile, err := db.ReadPackFile(indexObject.PackFileOID)
	if err != nil {
		return nil, err
	}
	packObject, ok := packFile.GetPackObject(oid)
	if !ok {
		return nil, nil
	}
	return packObject.ReadObject(), nil
}

// ReadLastStagedOrCommittedObject reads the last known version of stateful object in staging area or in commits.
func (db *DB) ReadLastStagedOrCommittedObject(oid string) (StatefulObject, error) {
	// Check staging area first
	stagedObject, ok := db.index.StagingArea.ReadObject(oid)
	if ok {
		return stagedObject, nil
	}

	// Check commits second
	indexObject, ok := db.index.objectsRef[oid]
	if !ok {
		// in staging area and in commits
		return nil, nil
	}
	packFile, err := db.ReadPackFile(indexObject.PackFileOID)
	if err != nil {
		return nil, err
	}
	packObject, ok := packFile.GetPackObject(oid)
	if !ok {
		return nil, nil
	}
	return packObject.ReadObject(), nil
}

// ReadPackFile reads a pack file on disk.
func (db *DB) ReadPackFile(oid string) (*PackFile, error) {
	result := new(PackFile)
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := result.Read(in); err != nil {
		return nil, err
	}
	return result, nil
}

// DeletePackFile removes a single pack file on disk
func (db *DB) DeletePackFile(packFile *PackFile) error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(packFile.OID))
	err := os.Remove(path)
	if err != nil {
		return err
	}
	// Save blob as orphans
	db.index.OrphanPackFiles = append(db.index.OrphanPackFiles, &IndexOrphanPackFile{
		OID:   packFile.OID,
		DTime: clock.Now(),
	})
	CurrentLogger().Infof("💾 Deleted pack file %s", filepath.Base(path))
	return nil
}

// ReadPackFilesFromCommit reads all pack files referenced by the commit.
func (db *DB) ReadPackFilesFromCommit(commit *Commit) ([]*PackFile, error) {
	var results []*PackFile
	for _, packFileRef := range commit.PackFiles {
		packFile, err := db.ReadPackFile(packFileRef.OID)
		if err != nil {
			return nil, err
		}

		results = append(results, packFile)
	}
	return results, nil
}

// CompressPackFile parses a pack file to remove obsolete pack objects.
func (db *DB) CompressPackFile(packFile *PackFile) (bool, error) {
	var stillActualPackObjects []*PackObject
	for _, packObject := range packFile.PackObjects {
		indexObject, ok := db.index.objectsRef[packObject.OID]
		if ok && indexObject.PackFileOID == packFile.OID {
			// Still the latest known revision
			stillActualPackObjects = append(stillActualPackObjects, packObject)
			CurrentLogger().Debugf("Up-to-date pack object %s [%s] detected", packObject.OID, packObject.Kind)
		} else {
			CurrentLogger().Debugf("Obsolete pack object %s [%s] detected", packObject.OID, packObject.Kind)
		}
	}

	CurrentLogger().Debugf("Found %d/%d actual pack objects in pack file %s", len(stillActualPackObjects), len(packFile.PackObjects), packFile.OID)

	if len(stillActualPackObjects) == len(packFile.PackObjects) {
		// Do nothing if no change
		return false, nil
	}

	// Edit the pack file to remove obsolete pack objects
	packFile.PackObjects = stillActualPackObjects
	packFile.MTime = clock.Now()
	if err := packFile.Save(); err != nil {
		return false, err
	}

	return true, nil
}

// ReadBlob reads a blob file on disk.
func (db *DB) ReadBlob(oid string) ([]byte, error) {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	return os.ReadFile(path)
}

// WriteBlob writes a blob file on disk
func (db *DB) WriteBlob(oid string, data []byte) error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(oid))
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	CurrentLogger().Infof("💾 Saved blob %s", filepath.Base(path))
	return nil
}

// DeleteBlobs removes all blobs on disk from a media
func (db *DB) DeleteBlobs(media *Media) error {
	for _, blob := range media.BlobRefs {
		if err := db.DeleteBlob(media, blob); err != nil {
			return err
		}
	}
	return nil
}

// DeleteBlob removes a single blob on disk
func (db *DB) DeleteBlob(media *Media, blob *BlobRef) error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(blob.OID))
	err := os.Remove(path)
	if err != nil {
		return err
	}
	// Save blob as orphans
	db.index.OrphanBlobs = append(db.index.OrphanBlobs, &IndexOrphanBlob{
		OID:      blob.OID,
		DTime:    clock.Now(),
		MediaOID: media.OID,
	})
	CurrentLogger().Infof("💾 Deleted blob %s", filepath.Base(path))
	return nil
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

func (db *DB) StagePackFile(packFile *PackFile) {
	db.index.StagePackFile(packFile)
}

func (db *DB) StagePackFileWithBlobs(packFile *PackFile, blobs []BlobRef) {
	db.index.StagePackFileWithBlobs(packFile, blobs)
}

// Commit creates a new commit object and clear the staging area.
func (db *DB) Commit(msg string) error {
	changesAdded := db.index.StagingArea.CountByState(Added)
	changesModified := db.index.StagingArea.CountByState(Modified)
	changesDeleted := db.index.StagingArea.CountByState(Deleted)
	changesTotal := db.index.StagingArea.Count()

	if changesTotal == 0 {
		return errors.New(`nothing to commit (create/copy files and use "nt add" to track)`)
	}

	// Run Hooks first for user to fix note issues
	// if a hook fails due to a malformed note.
	for _, obj := range db.index.StagingArea {
		if obj.PackObject.Kind != "note" {
			// We execute hooks only on note objects
			continue
		}
		note := obj.ReadObject().(*Note)
		if err := note.RunHooks(nil); err != nil {
			return err
		}
	}

	// Convert the staging area to a new commit file
	commit, packFiles := db.index.CreateCommitFromStagingArea()
	for _, packFile := range packFiles {
		if err := packFile.Save(); err != nil {
			return err
		}
		db.index.putPackFile(commit.OID, packFile)
	}

	// Save updates staging area
	if err := db.index.Save(); err != nil {
		return err
	}

	// Update the commit graph
	if err := db.commitGraph.AppendCommit(commit); err != nil {
		return err
	}
	if err := db.commitGraph.Save(); err != nil {
		return err
	}

	// Update the main ref
	db.updateRef("main", commit.OID)

	// Output result
	fmt.Printf("[%7s] %s\n", commit.OID, msg)
	fmt.Printf(" %d objects changes", changesTotal)
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
	for _, packFile := range packFiles {
		for _, obj := range packFile.PackObjects {
			action := "modify"
			switch obj.State {
			case Added:
				action = "create"
			case Deleted:
				action = "delete"
			}
			fmt.Printf(" %s %s\n", action, obj.Description)
		}
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
	diff := db.commitGraph.Diff(cg)
	commits := diff.MissingCommits
	for _, commit := range commits {

		// Download each commit in a single transaction
		err := db.BeginTransaction()
		if err != nil {
			return err
		}
		defer db.RollbackTransaction()

		for _, packFileRef := range commit.PackFiles {
			// Retrieve the pack file content
			data, err = origin.GetObject(OIDToPath(packFileRef.OID))
			if errors.Is(err, ErrObjectNotExist) {
				return fmt.Errorf("missing pack file %q", packFileRef.OID)
			} else if err != nil {
				return err
			}

			// Read the content
			packFile := new(PackFile)
			if err := packFile.Read(bytes.NewReader(data)); err != nil {
				return err
			}

			// Parse the objects and blobs
			for _, packObject := range packFile.PackObjects {
				remoteObject := packObject.ReadObject()

				// Retrieve optional blobs
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

				newState := db.determineState(packObject)
				remoteObject.ForceState(newState)

				// Add in SQL database
				if err := remoteObject.Save(); err != nil {
					return err
				}

			}

			// Write on disk
			if err := packFile.Save(); err != nil {
				return fmt.Errorf("unable to write retrieved pack file %q: %v", packFile.OID, err)
			}

			// Enrich index
			db.index.putPackFile(commit.OID, packFile)
		}

		if err := db.CommitTransaction(); err != nil {
			return err
		}

		db.commitGraph.AppendCommit(commit)

		// Update the main ref
		db.updateRef("main", commit.OID)
	}

	// Persist local commit-graph including downloaded commits
	if err := db.commitGraph.Save(); err != nil {
		return err
	}

	// Keep note of last origin retrieved commit
	db.updateRef("origin", cg.Ref())

	return nil
}

// Push pushes new objects remotely.
func (db *DB) Push() error {
	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteWriter is a personal tool and you are not expected to push from two repositories at the same time.

	origin := db.Origin()
	if origin == nil {
		return errors.New("no remote found")
	}

	// List of changes to push
	var commitsToPush []*Commit
	// + local changes made during gc:
	var packFilesToPush []*PackFileRef
	var packFilesToDelete []*PackFileRef
	var blobsToDelete []string

	// Read remote's commit-graph to find commits to push
	data, err := origin.GetObject("info/commit-graph")
	if errors.Is(err, ErrObjectNotExist) {
		// Push all local commits
		commitsToPush = db.commitGraph.Commits
	} else if err != nil {
		return err
	} else {

		// Read the origin commit graph
		originCommitGraph := new(CommitGraph)
		if err := originCommitGraph.Read(bytes.NewReader(data)); err != nil {
			return err
		}

		// Important! Check if we miss some commits locally
		diff := db.commitGraph.Diff(originCommitGraph)
		if len(diff.MissingCommits) > 0 {
			return errors.New("missing commits from origin")
		}

		// Read the origin index (must exist if commit-graph exists)
		data, err := origin.GetObject("index")
		if errors.Is(err, ErrObjectNotExist) {
			return errors.New("missing index in remote")
		}
		if err != nil {
			return err
		}
		originIndex := new(Index)
		if err := originIndex.Read(bytes.NewReader(data)); err != nil {
			return err
		}

		// Compare local and remote databases
		commitGraphDiff := originCommitGraph.Diff(db.commitGraph)
		indexDiff := originIndex.Diff(db.index)

		// Find only missing commits
		commitsToPush = commitGraphDiff.MissingCommits

		// Apply changes made during previous invocation of GC
		packFilesToPush = append(packFilesToPush, commitGraphDiff.MissingPackFiles...)
		packFilesToPush = append(packFilesToPush, commitGraphDiff.EditedPackFiles...)
		packFilesToDelete = commitGraphDiff.ObsoletePackFiles
		blobsToDelete = indexDiff.MissingOrphanBlobs
	}

	CurrentLogger().Infof("Found %d new commit(s), %d updated new pack file(s), %d updated old pack file(s), %d obsolete blobs",
		len(commitsToPush),
		len(packFilesToPush),
		len(packFilesToDelete),
		len(blobsToDelete))

	// Iterate over commits to push
	for _, commit := range commitsToPush {

		for _, packFileRef := range commit.PackFiles {
			packFile, err := db.ReadPackFile(packFileRef.OID)
			if err != nil {
				return err
			}

			// Upload blobs first (if the commit upload fails, it will be retried at least)
			for _, packObject := range packFile.PackObjects {
				object := packObject.ReadObject()
				for _, blobRef := range object.Blobs() {
					blobData, err := db.ReadBlob(blobRef.OID)
					if err != nil {
						return err
					}
					CurrentLogger().Debugf("Uploading blob %s...", blobRef.OID)
					if err := origin.PutObject(OIDToPath(blobRef.OID), blobData); err != nil {
						return err
					}
				}
			}

			// Upload the pack file
			buf := new(bytes.Buffer)
			if err := packFile.Write(buf); err != nil {
				return err
			}
			CurrentLogger().Debugf("Uploading new commit including pack file %s...", packFileRef.OID)
			if err := origin.PutObject(OIDToPath(packFileRef.OID), buf.Bytes()); err != nil {
				return err
			}
		}
	}

	// Iterate over pack files to push
	for _, packFileRef := range packFilesToPush {
		// Upload the pack file
		packFile, err := CurrentDB().ReadPackFile(packFileRef.OID)
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		if err := packFile.Write(buf); err != nil {
			return err
		}
		CurrentLogger().Debugf("Uploading merged pack file %s...", packFileRef.OID)
		if err := origin.PutObject(OIDToPath(packFile.OID), buf.Bytes()); err != nil {
			return err
		}
	}

	// Iterate over pack files to remove
	for _, packFileRef := range packFilesToDelete {
		// Remove the pack file
		CurrentLogger().Debugf("Deleting obsolete pack file %s...", packFileRef.OID)
		if err := origin.DeleteObject(OIDToPath(packFileRef.OID)); err != nil {
			return err
		}
	}

	// Iterate over blobs to remove
	for _, blobOID := range blobsToDelete {
		// Remove the blob
		CurrentLogger().Debugf("Deleting obsolete blob %s...", blobOID)
		if err := origin.DeleteObject(OIDToPath(blobOID)); err != nil {
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

	// Update remote index
	buf = new(bytes.Buffer)
	if err := db.index.CloneForRemote().Write(buf); err != nil {
		return err
	}
	if err := origin.PutObject("index", buf.Bytes()); err != nil {
		return err
	}

	// Push local config for mobile app to retrieve settings
	data, err = os.ReadFile(filepath.Join(CurrentConfig().RootDirectory, ".nt/config"))
	if err != nil {
		return err
	}
	if err := origin.PutObject("config", data); err != nil {
		return err
	}

	// Update the origin ref
	db.updateRef("origin", db.refs["main"])

	return nil
}

// Reset reverts the latest add command.
func (db *DB) Reset() error {
	// Run all queries inside the same transaction
	err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer db.RollbackTransaction()

	// We must clear the staging area
	for _, obj := range db.index.StagingArea {

		switch obj.State {
		case Added:
			// Deleted the object in SQL database
			object := obj.ReadObject()
			if object == nil {
				return fmt.Errorf("unknown object %q", obj.OID)
			}
			// Mark for deletion
			object.ForceState(Deleted)
			if err := object.Save(); err != nil {
				return err
			}
		case Deleted:
			// Re-read object from latest commit
			parentPackFile, err := db.ReadPackFile(obj.PreviousPackFileOID)
			if err != nil {
				return fmt.Errorf("missing parent pack file %q: %v", obj.PreviousPackFileOID, err)
			}
			original, found := parentPackFile.GetPackObject(obj.OID)
			if !found {
				return fmt.Errorf("missing object %q in pack file %s", obj.OID, obj.PreviousPackFileOID)
			}
			originalObject := original.ReadObject()
			if originalObject == nil {
				return fmt.Errorf("unknown object %q", obj.OID)
			}
			// Mark for restoration
			originalObject.ForceState(Added)
			originalObject.Save()
		case Modified:
			// Re-read object from latest commit
			parentPackFile, err := db.ReadPackFile(obj.PreviousPackFileOID)
			if err != nil {
				return fmt.Errorf("missing parent pack file %q: %v", obj.PreviousPackFileOID, err)
			}
			original, found := parentPackFile.GetPackObject(obj.OID)
			if !found {
				return fmt.Errorf("missing object %q in pack file %s", obj.OID, obj.PreviousPackFileOID)
			}
			originalObject := original.ReadObject()
			if originalObject == nil {
				return fmt.Errorf("unknown object %q", obj.OID)
			}
			// Nothing to change. Simply save back.
			originalObject.ForceState(Modified)
			originalObject.Save()
		}
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	db.index.StagingArea = nil
	err = db.index.Save()

	return err
}

// Diff show the changes in the staging area.
func (db *DB) Diff() (string, error) {
	var diff strings.Builder
	for _, stagedObj := range db.index.StagingArea {
		if stagedObj.Kind != "note" {
			continue
		}
		stagedNote := stagedObj.ReadObject().(*Note)

		commitObj, err := db.ReadCommittedObject(stagedObj.OID)
		if err != nil {
			return "", err
		}
		noteContentBefore := ""
		if commitObj != nil {
			commitNote := commitObj.(*Note)
			noteContentBefore = string(commitNote.Content)
		}
		noteContentAfter := string(stagedNote.Content)
		patch := godiffpatch.GeneratePatch(stagedNote.RelativePath, noteContentBefore, noteContentAfter)
		diff.WriteString(patch)
	}

	return diff.String(), nil
}

func (db *DB) PrintIndex() {
	fmt.Println("\n\n.nt/objects/info/commit-graph")
	for _, commit := range db.commitGraph.Commits {
		fmt.Printf("  - %s (%d pack files)\n", commit.OID, len(commit.PackFiles))
	}

	fmt.Println("\n\n.nt/objects/info/index")
	fmt.Printf("  - %d objects:\n", len(db.index.Objects))
	for _, indexObject := range db.index.Objects {
		fmt.Printf("     %s %s (%s/%s)\n", indexObject.OID, indexObject.Kind, indexObject.CommitOID, indexObject.PackFileOID)
	}
	fmt.Printf("  - %d pack files:\n", len(db.index.PackFiles))
	for packFileOID, commitOID := range db.index.PackFiles {
		fmt.Printf("     %s => %s\n", packFileOID, commitOID)
	}
	fmt.Printf("   - %d orphan blobs\n", len(db.index.OrphanBlobs))
	fmt.Printf("   - %d orphan pack files\n", len(db.index.OrphanPackFiles))
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
	// * Notes can be edited over times and be committed again and again. We don't want to store all revisions in pack files
	//   (Git is recommended to version your notes).
	//   The GC traverses the commit graph to analyze pack files and remove old revisions
	//   that are no longer relevant. Pack files will be rewritten to remove old data.
	//   In addition, we also want to limit the number of files on disk (preferable
	//   when using an object storage as remotes). The GC merges packfiles present
	// in a single commit when their number of objects becomes too low.

	// Implementation: We use a multi-stage algorithm even when a single pass would be possible.
	// The only motivation is to keep the code approachable for every stage.

	// Stage 1: Blob reclaiming
	// -------

	CurrentLogger().Info("Reclaiming blobs...")

	// Walk the commits to locate all medias
	var allMedias []*Media
	for _, commit := range db.commitGraph.Commits {
		for _, packFileRef := range commit.PackFiles {
			packFile, err := db.ReadPackFile(packFileRef.OID)
			if err != nil {
				return err
			}

			for _, object := range packFile.PackObjects {
				if object.Kind == "media" {
					// Read the media
					media := new(Media)
					if err := object.Data.Unmarshal(media); err != nil {
						return err
					}

					allMedias = append(allMedias, media)
				}
			}
		}
	}

	// Traverse in reverse order to find used blobs
	traversedMedias := make(map[string]bool)
	usedBlobs := make(map[string]bool)
	for i := len(allMedias) - 1; i >= 0; i-- {
		media := allMedias[i]
		if _, ok := traversedMedias[media.OID]; ok {
			// Old media version
			continue
		}
		traversedMedias[media.OID] = true
		if !media.DeletedAt.IsZero() {
			// Media no longer exists = blobs are no longer truly referenced by it
			continue
		}
		for _, blob := range media.BlobRefs {
			usedBlobs[blob.OID] = true
		}
	}

	// Traverse all medias to detect unused blobs based on the previous list
	for _, media := range allMedias {
		for _, blob := range media.BlobRefs {
			if db.index.IsOrphanBlob(blob.OID) {
				// Already deleted
				continue
			}
			if _, ok := usedBlobs[blob.OID]; !ok {
				db.DeleteBlob(media, blob)
			}
		}
	}

	// Stage 2: Pack File Optimization
	// -------
	// Walk the commits to list all currently actual objects

	CurrentLogger().Info("Reclaiming pack files...")

	// Memorize if a commit was edited to know if we need to save the commit graph
	commitRevised := false

	for _, commit := range db.commitGraph.Commits {
		oldPackFilesCount := len(commit.PackFiles)
		changed, err := db.CompressCommit(commit)
		if err != nil {
			return err
		}
		if changed {
			newPackFilesCount := len(commit.PackFiles)
			CurrentLogger().Infof("Commit %s was changed (%d => %d pack files)", commit.OID, oldPackFilesCount, newPackFilesCount)
			commitRevised = true
		}
	}

	if commitRevised {
		CurrentLogger().Info("Saving .nt/objects/info/commit-graph")
		if err := db.commitGraph.Save(); err != nil {
			return err
		}
	}

	return db.index.Save()
}

// CompressCommit remove obsolete pack objects and merge small pack files together.
func (db *DB) CompressCommit(commit *Commit) (bool, error) {
	commitRevised := false

	packFiles, err := db.ReadPackFilesFromCommit(commit)
	if err != nil {
		return false, err
	}

	CurrentLogger().Debugf("Analyzing %d pack files in commit %s...", len(packFiles), commit.OID)

	// The resulting pack files after compressing/merging
	var newPackFiles []*PackFile

	for _, packFile := range packFiles {
		changed, err := db.CompressPackFile(packFile)
		if err != nil {
			return false, err
		}
		if changed {
			commitRevised = true
			// Drop the pack file if no object are still actual
			if len(packFile.PackObjects) == 0 {
				// Packfile can be removed
				if err := db.DeletePackFile(packFile); err != nil {
					return false, err
				}
			} else {
				newPackFiles = append(newPackFiles, packFile)
			}
		} else {
			newPackFiles = append(newPackFiles, packFile)
		}
	}

	if !commitRevised {
		// Nothing has changed, no need to try to merge pack files
		return false, nil
	}

	CurrentLogger().Debugf("Commit %s has changed. Trying to merge pack files...", commit.OID)

	// Try to merge pack files
	i := 0
	for i < len(newPackFiles)-2 {
		currentPackFile := newPackFiles[i]
		nextPackFile := newPackFiles[i+1]
		newPackFile, ok := currentPackFile.Merge(nextPackFile)
		if !ok {
			i++
		} else {
			// Save the new pack file
			err := newPackFile.Save()
			if err != nil {
				return false, err
			}
			// Delete the old pack file
			CurrentLogger().Debugf("Merged pack files %s and %s", currentPackFile.OID, nextPackFile.OID)
			db.DeletePackFile(currentPackFile)
			db.DeletePackFile(nextPackFile)
			newPackFiles = slices.Replace[[]*PackFile](newPackFiles, i, i+1, newPackFile)
		}
	}

	// Update the commit with new pack files OIDs
	var packFilesRefs []*PackFileRef
	for _, packFile := range newPackFiles {
		packFilesRefs = append(packFilesRefs, packFile.Ref())
	}
	commit.PackFiles = packFilesRefs
	commit.MTime = clock.Now()

	return commitRevised, nil
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
	db.refs[name] = commitOID
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

func (db *DB) determineState(packObject *PackObject) State {
	indexObject, found := db.index.objectsRef[packObject.OID]

	// The remote object doesn't exist locally
	if !found {
		if packObject.State != Deleted {
			return Added
		}
		return None
	}

	// The remote object exist locally
	if indexObject.MTime.After(packObject.MTime) {
		// we have a more recent version
		return None
	}
	return Modified
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
	objectsPath := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/")

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
		oid := filepath.Base(file)

		result.ObjectFiles++

		if _, ok := db.index.PackFiles[oid]; ok {
			// It's a pack file, check the content to count objects/notes
			packFile, err := NewPackFileFromPath(file)
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

	result.Commits = len(db.commitGraph.Commits)

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
