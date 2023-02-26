package core

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
)

func (c *Collection) walk(path string, fn func(path string, stat fs.FileInfo) error) {
	config := CurrentConfig()

	CurrentLogger().Infof("Reading %s...\n", path)

	filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
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
	}

	deletions, err := c.findObjectsLastCheckedBefore(buildTime)
	if err != nil {
		return err
	}
	for _, deletion := range deletions {
		deletion.SetTombstone()
		if err := db.StageObject(deletion); err != nil {
			return fmt.Errorf("unable to stage deleted object %s: %v", deletion, err)
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

func (c *Collection) findObjectsLastCheckedBefore(buildTime time.Time) ([]Object, error) {
	// Search for deleted objects...
	var deletions []Object

	links, err := FindLinksLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range links {
		deletions = append(deletions, object)
	}
	reminders, err := FindRemindersLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range reminders {
		deletions = append(deletions, object)
	}
	flashcards, err := FindFlashcardsLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range flashcards {
		deletions = append(deletions, object)
	}
	medias, err := FindMediasLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range medias {
		deletions = append(deletions, object)
	}
	notes, err := FindNotesLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range notes {
		deletions = append(deletions, object)
	}
	files, err := FindFilesLastCheckedBefore(buildTime)
	if err != nil {
		return nil, err
	}
	for _, object := range files {
		deletions = append(deletions, object)
	}
	return deletions, nil
}
