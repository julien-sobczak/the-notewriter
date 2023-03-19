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

	"github.com/julien-sobczak/the-notetaker/internal/reference"
	"github.com/julien-sobczak/the-notetaker/internal/reference/wikipedia"
	"github.com/julien-sobczak/the-notetaker/internal/reference/zotero"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/resync"

	"golang.org/x/exp/slices"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

var (
	// Lazy-load configuration and ensure a single read
	collectionOnce      resync.Once
	collectionSingleton *Collection
)

type Collection struct {
	Path          string `yaml:"path"`
	bookManager   reference.Manager
	personManager reference.Manager
}

func CurrentCollection() *Collection {
	collectionOnce.Do(func() {
		var err error
		zoteroManager := zotero.NewReferenceManager()
		wikipediaManager := wikipedia.NewReferenceManager()
		collectionSingleton, err = NewCollection(zoteroManager, wikipediaManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to init current collection: %v\n", err)
			os.Exit(1)
		}
	})
	return collectionSingleton
}

func NewCollection(bookManager reference.Manager, personManager reference.Manager) (*Collection, error) {
	config := CurrentConfig()

	absolutePath, err := filepath.Abs(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	c := &Collection{
		Path:          absolutePath,
		bookManager:   bookManager,
		personManager: personManager,
	}
	return c, nil
}

func (c *Collection) CreateNewReferenceFile(identifier string, kind string) (*File, error) {
	var ref reference.Reference
	var err error

	switch kind {
	case ReferenceKindBook:
		ref, err = c.bookManager.Search(identifier)
	case ReferenceKindAuthor:
		ref, err = c.personManager.Search(identifier)
	}
	if err != nil {
		return nil, err
	}

	var attributes []Attribute
	for _, refAttribute := range ref.Attributes() {
		attributes = append(attributes, Attribute{
			Key:   refAttribute.Key,
			Value: refAttribute.Value,
		})
	}

	return NewFileFromAttributes("", attributes), nil // FIXME use a name
}

/* Reference Management */

func (c *Collection) AddNewReferenceFile(identifier string, kind string) error {
	f, err := c.CreateNewReferenceFile(identifier, kind)
	if err != nil {
		return err
	}
	return f.SaveOnDisk()
}

func (c *Collection) Close() {
	CurrentDB().Close()
}

// GetNoteRelativePath converts a relative path from a note to a relative path from the collection root directory.
func (c *Collection) GetNoteRelativePath(fileRelativePath string, srcPath string) (string, error) {
	return filepath.Rel(c.Path, filepath.Join(filepath.Dir(c.GetAbsolutePath(fileRelativePath)), srcPath))
}

// GetFileRelativePath converts a relative path of a file to a relative path from the collection.
func (c *Collection) GetFileRelativePath(fileAbsolutePath string) (string, error) {
	return filepath.Rel(c.Path, fileAbsolutePath)
}

// GetAbsolutePath converts a relative path from the collection to an absolute path on disk.
func (c *Collection) GetAbsolutePath(path string) string {
	if strings.HasPrefix(path, c.Path) {
		return path
	}
	return filepath.Join(c.Path, path)
}

/* Commands */

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

		if config.IgnoreFile.MustExcludeFile(relpath, info.IsDir()) {
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
		// Check for dead medias only when adding the root directory.
		// For example, when adding a file, it can contains references to medias stored in a directory outside the given path.
		if path == CurrentConfig().RootDirectory { // nt add .
			// As we walked the whole hierarchy, all medias must have be checked.
			mediaDeletions, err := FindMediasLastCheckedBefore(buildTime)
			if err != nil {
				return err
			}
			for _, mediaDeletion := range mediaDeletions {
				deletions = append(deletions, mediaDeletion)
			}
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

func (c *Collection) Lint(paths ...string) (*LintResult, error) {
	/*
	 * Implementation: The linter must only considering local files and
	 * ignore commits or the staging area completely.
	 *
	 * Indeed, the linter can be run initially before any files have been added or committed.
	 * In the same way, a file can reference a media that existed
	 * and is still present in the database objects even so the media has been deleted and
	 * not added since.
	 */
	var result LintResult
	rules := CurrentConfig().LintFile.Rules

	// Check all rules are valid before checking anything else
	for _, rule := range rules {
		ruleName := rule.Name
		_, ok := LintRules[ruleName]
		if !ok {
			return nil, fmt.Errorf("unknown lint rule %q", rule.Name)
		}
		if rule.Severity != "" && slices.Contains([]string{"error", "warning"}, rule.Severity) {
			return nil, fmt.Errorf("unknown severity %q for lint rule %q", rule.Severity, rule.Name)
		}
	}

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

			// Work without the database
			file, err := ParseFile(path)
			if err != nil {
				return err
			}

			foundViolations := false

			for _, configRule := range rules {
				rule := LintRules[configRule.Name]

				// Check path restrictions
				matchAllIncludes := true
				for _, include := range configRule.Includes {
					if !include.Match(file.RelativePath) {
						matchAllIncludes = false
					}
				}
				if !matchAllIncludes {
					continue
				}

				violations, err := rule.Eval(file, configRule.Args)
				if err != nil {
					return err
				}
				if len(violations) > 0 {
					foundViolations = true
				}
				if configRule.Severity == "warning" {
					result.Warnings = append(result.Warnings, violations...)
				} else {
					result.Errors = append(result.Errors, violations...)
				}
			}

			// Update stats
			if foundViolations {
				result.AffectedFiles += 1
			}
			result.AnalyzedFiles += 1

			return nil
		})
	}

	return &result, nil
}
