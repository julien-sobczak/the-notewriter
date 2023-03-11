package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
)

func (c *Collection) walk(path string, fn func(path string, stat fs.FileInfo) error) {
	config := CurrentConfig()

	CurrentLogger().Infof("Reading %s...\n", path)

	filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}
		if dirname == ".git" {
			return fs.SkipDir
		}

		relpath, err := CurrentCollection().GetFileRelativePath(path)
		if err != nil {
			// ignore the file
			return nil
		}

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

		if err := fn(path, fileInfo); err != nil {
			return err
		}

		return nil
	})
}

// Add implements the command `nt add`.`
func (c *Collection) Add(paths ...string) error {
	// Any object not updated after this date will be considered as deletions
	buildTime := clock.Now()

	db := CurrentDB()

	// Keep notes of processed objects to avoid duplication of effort
	// when some objects like medias are referenced by different notes.
	traversedObjects := make(map[string]bool)

	// Run all queries inside the same transaction
	tx, err := db.Client().BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Traverse all given paths
	for _, path := range paths {

		if path == "." {
			// Process all files in the root directory
			path = CurrentConfig().RootDirectory
		} else if !filepath.IsAbs(path) {
			path = CurrentCollection().GetAbsolutePath(path)
		}

		c.walk(path, func(path string, stat fs.FileInfo) error {
			CurrentLogger().Debugf("Processing %s...\n", path)

			file, err := NewOrExistingFile(path)
			if err != nil {
				return err
			}

			if file.State() != None {
				if err := db.StageObject(file); err != nil {
					return fmt.Errorf("unable to stage modified object %s: %v", file, err)
				}
			}
			traversedObjects[file.UniqueOID()] = true
			if err := file.Save(tx); err != nil {
				return nil
			}

			for _, object := range file.SubObjects() {
				if _, found := traversedObjects[object.UniqueOID()]; found {
					// already processed
					continue
				}
				if object.State() != None {
					if err := db.StageObject(object); err != nil {
						return fmt.Errorf("unable to stage modified object %s: %v", object, err)
					}
				}
				traversedObjects[object.UniqueOID()] = true
				if err := object.Save(tx); err != nil {
					return err
				}
			}

			return nil
		})

		deletions, err := c.findObjectsLastCheckedBefore(buildTime, path)
		if err != nil {
			return err
		}
		for _, deletion := range deletions {
			deletion.ForceState(Deleted)
			if err := db.StageObject(deletion); err != nil {
				return fmt.Errorf("unable to stage deleted object %s: %v", deletion, err)
			}
		}
	}

	// Don't forget to commit
	if err := tx.Commit(); err != nil {
		return err
	}
	// And to persist the index
	if err := db.index.Save(); err != nil {
		return err
	}

	return nil
}

func (c *Collection) findObjectsLastCheckedBefore(buildTime time.Time, path string) ([]StatefulObject, error) {
	CurrentLogger().Debugf("Searching for %s", path)
	// Search for deleted objects...
	var deletions []StatefulObject

	links, err := FindLinksLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range links {
		deletions = append(deletions, object)
	}
	reminders, err := FindRemindersLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range reminders {
		deletions = append(deletions, object)
	}
	flashcards, err := FindFlashcardsLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range flashcards {
		deletions = append(deletions, object)
	}
	notes, err := FindNotesLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range notes {
		deletions = append(deletions, object)
	}
	files, err := FindFilesLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range files {
		deletions = append(deletions, object)
	}
	return deletions, nil
}

// Status displays current objects in staging area.
func (c *Collection) Status() (string, error) {
	// No side-effect with this command.
	// We only output results.
	var sb strings.Builder

	// Show staging area content
	sb.WriteString(`Changes to be committed:` + "\n")
	sb.WriteString(`  (use "nt restore..." to unstage)` + "\n")
	stagingArea := CurrentDB().index.StagingArea
	for _, obj := range stagingArea.Added {
		sb.WriteString(fmt.Sprintf("\tadded:\t%s\n", obj.Description))
	}
	for _, obj := range stagingArea.Modified {
		sb.WriteString(fmt.Sprintf("\tmodified:\t%s\n", obj.Description))
	}
	for _, obj := range stagingArea.Deleted {
		sb.WriteString(fmt.Sprintf("\tdeleted:\t%s\n", obj.Description))
	}

	// Show modified files not in staging area
	type ObjectStatus struct {
		RelativePath string
		OID          string
		Status       State
	}
	uncommittedFiles := make(map[string]ObjectStatus)

	root := CurrentConfig().RootDirectory
	c.walk(root, func(path string, stat fs.FileInfo) error {
		relpath, err := CurrentCollection().GetFileRelativePath(path)
		if err != nil {
			return err
		}

		// Use index to determine if the file is new or changed
		indexObject, ok := CurrentDB().index.StagingArea.ContainsFile(relpath)
		if ok {
			if indexObject.MTime.Equal(stat.ModTime()) {
				// File was not updated since added to staging area = still OK
				return nil
			}
			if indexObject.MTime.Before(stat.ModTime()) {
				uncommittedFiles[relpath] = ObjectStatus{
					RelativePath: relpath,
					OID:          indexObject.OID,
					Status:       Modified,
				}
			} else {
				uncommittedFiles[relpath] = ObjectStatus{
					RelativePath: relpath,
					OID:          indexObject.OID,
					Status:       None,
				}
			}
		} else {
			uncommittedFiles[relpath] = ObjectStatus{
				RelativePath: relpath,
				Status:       Added,
			}
		}

		return nil
	})
	// Traverse index to find known files not traversed by the walk
	for relpath, indexObject := range CurrentDB().index.filesRef {
		_, found := uncommittedFiles[relpath]
		if !found {
			uncommittedFiles[relpath] = ObjectStatus{
				RelativePath: relpath,
				OID:          indexObject.OID,
				Status:       Deleted,
			}
		}
	}

	if len(uncommittedFiles) > 0 {
		// Sort map entries
		keys := make([]string, 0, len(uncommittedFiles))
		for k := range uncommittedFiles {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sb.WriteString("\n")
		sb.WriteString(`Changes not staged for commit:` + "\n")
		sb.WriteString(`  (use "nt add <file>..." to update what will be committed)` + "\n")
		for _, key := range keys {
			obj := uncommittedFiles[key]
			if obj.Status == None {
				continue
			}
			sb.WriteString(fmt.Sprintf("\t%s:\t%s\n", obj.Status, obj.RelativePath))
		}
	}

	return sb.String(), nil
}
