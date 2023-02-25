package core

import (
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
	buildTime := clock.Now()

	db := CurrentDB()

	for _, path := range paths {
		if path == "." {
			path = c.Path
		}
		c.walk(path, func(path string, stat fs.FileInfo) error {
			CurrentLogger().Debugf("Processing %s...\n", path) // TODO emit notif for tests?

			file, err := NewOrExistingFile(path)
			if err != nil {
				return err
			}

			if file.State() != None {
				if err := db.StageObject(file); err != nil {
					return fmt.Errorf("unable to stage modified object %s: %v", file, err)
				}
			}
			for _, object := range file.SubObjects() {
				if object.State() != None {
					if err := db.StageObject(object); err != nil {
						return fmt.Errorf("unable to stage modified object %s: %v", object, err)
					}
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
