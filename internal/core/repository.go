package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"golang.org/x/exp/slices"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

var (
	// Lazy-load configuration and ensure a single read
	repositoryOnce      resync.Once
	repositorySingleton *Repository
)

type Repository struct {
	Path string `yaml:"path"`
}

func CurrentRepository() *Repository {
	repositoryOnce.Do(func() {
		var err error
		repositorySingleton, err = NewRepository()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to init current repository: %v\n", err)
			os.Exit(1)
		}
	})
	return repositorySingleton
}

func NewRepository() (*Repository, error) {
	config := CurrentConfig()

	absolutePath, err := filepath.Abs(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	c := &Repository{
		Path: absolutePath,
	}
	return c, nil
}

func (r *Repository) Close() {
	CurrentDB().Close()
}

// GetNoteRelativePath converts a relative path from a note to a relative path from the repository root directory.
func (r *Repository) GetNoteRelativePath(fileRelativePath string, srcPath string) (string, error) {
	return filepath.Rel(r.Path, filepath.Join(filepath.Dir(r.GetAbsolutePath(fileRelativePath)), srcPath))
}

// GetFileRelativePath converts a relative path of a file to a relative path from the repository.
func (r *Repository) GetFileRelativePath(fileAbsolutePath string) (string, error) {
	return filepath.Rel(r.Path, fileAbsolutePath)
}

// GetAbsolutePath converts a relative path from the repository to an absolute path on disk.
func (r *Repository) GetAbsolutePath(path string) string {
	if strings.HasPrefix(path, r.Path) {
		return path
	}
	return filepath.Join(r.Path, path)
}

/* Commands */

type MatchedFile struct {
	Path         string
	RelativePath string
	DirEntry     fs.DirEntry
	FileInfo     fs.FileInfo
}

// IndexFilesFirst ensures index files are processed first.
var IndexFilesFirst = func(a, b string) bool {
	dirA := filepath.Dir(a)
	dirB := filepath.Dir(b)
	if dirA != dirB {
		return a < b
	}
	baseA := text.TrimExtension(filepath.Base(a))
	baseB := text.TrimExtension(filepath.Base(b))
	// move index files up
	if strings.EqualFold(baseA, "index") {
		return true
	} else if strings.EqualFold(baseB, "index") {
		return false
	}
	return a < b // os.WalkDir already returns file in lexical order
}

func (r *Repository) walk(paths []string, fn func(path string, stat fs.FileInfo) error) error {
	config := CurrentConfig()

	var matchedFiles []string
	var fileInfos = make(map[string]*fs.FileInfo)
	var filePaths = make(map[string]string)

	for _, path := range paths {
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

			relpath, err := CurrentRepository().GetFileRelativePath(path)
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

			// A file found to process!
			fileInfos[relpath] = &fileInfo
			filePaths[relpath] = path
			matchedFiles = append(matchedFiles, relpath)

			return nil
		})
	}

	// Process the file in a given order:

	// Constraint 1: index.md must be processed before other notes under this directory
	slices.SortFunc(matchedFiles, IndexFilesFirst)

	// Constraint 2: Embedded notes must be processed before the file that referenced them
	var sortedFiles []string
	addedFileIndices := make(map[int]bool)
	changedDuringIteration := false
	for len(addedFileIndices) < len(matchedFiles) { // until all files are added or no more files can be added due to a cyclic dependency

		for i, relpath := range matchedFiles {
			if addedFileIndices[i] {
				// Already added
				continue
			}

			// A file can be added iff:
			// - no external link OR referenced files have been added first if present in the same batch

			b, err := os.ReadFile(filePaths[relpath])
			if err != nil {
				return err
			}
			wikilinks := ParseWikilinks(string(b))
			var externalLinks []*Wikilink
			for _, wikilink := range wikilinks {
				if wikilink.External() {
					externalLinks = append(externalLinks, wikilink)
				}
			}

			externalLinksSatisfied := true
			for _, wikilink := range externalLinks {
				wikipath := text.TrimExtension(wikilink.Path())
				for j, otherRelpath := range matchedFiles {
					if addedFileIndices[j] {
						// Already satisfied
						continue
					}
					if strings.HasSuffix(text.TrimExtension(otherRelpath), wikipath) && !addedFileIndices[j] {
						externalLinksSatisfied = false
					}
				}
			}

			if externalLinksSatisfied {
				addedFileIndices[i] = true
				sortedFiles = append(sortedFiles, relpath)
				changedDuringIteration = true
			}
		}

		if !changedDuringIteration {
			// cyclic dependency found
			CurrentLogger().Info("Cyclic dependency between files detected. Incomplete note(s) can result.")
			// Add remaining notes without taking care of dependencies...
			for i, relpath := range matchedFiles {
				if addedFileIndices[i] {
					// Already added
					continue
				}
				sortedFiles = append(sortedFiles, relpath)
			}
			break
		}
		changedDuringIteration = false
	}

	// Execute callbacks
	for _, relpath := range sortedFiles {
		err := fn(filePaths[relpath], *fileInfos[relpath])
		if err != nil {
			return err
		}
	}

	return nil
}

// normalizePaths converts to absolute paths.
func (r *Repository) normalizePaths(paths ...string) []string {
	if len(paths) == 0 {
		return []string{CurrentConfig().RootDirectory}
	}
	var results []string
	for _, path := range paths {
		if path == "." {
			// Process all files in the root directory
			path = CurrentConfig().RootDirectory
		} else if !filepath.IsAbs(path) {
			path = r.GetAbsolutePath(path)
		}
		results = append(results, path)
	}
	return results
}

// Add implements the command `nt add`.`
func (r *Repository) Add(paths ...string) error {
	// Start with command linter (do not stage invalid file)
	linterResult, err := r.Lint(nil, paths...)
	if err != nil {
		return err
	}
	if len(linterResult.Errors) > 0 {
		return fmt.Errorf("%d linter errors detected:\n%s", len(linterResult.Errors), linterResult)
	}

	// Any object not updated after this date will be considered as deletions
	buildTime := clock.Now()
	db := CurrentDB()
	paths = r.normalizePaths(paths...)

	// Keep notes of processed objects to avoid duplication of effort
	// when some objects like medias are referenced by different notes.
	traversedObjects := make(map[string]bool)
	var traversedNotes []*Note

	// Keep notes of unprocessed medias to generate blob using goroutines to speed up the execution
	var unprocessedMedias []*Media

	// Run all queries inside the same transaction
	err = db.BeginTransaction()
	if err != nil {
		return err
	}
	defer db.RollbackTransaction()

	// Traverse all given path to add files
	err = r.walk(paths, func(path string, stat fs.FileInfo) error {
		CurrentLogger().Debugf("Processing %s...\n", path)

		var parent *File = nil
		// Try to load the optional parent present in the same directory
		if filepath.Base(path) != "index.md" {
			parentRelativePath, err := r.GetFileRelativePath(filepath.Join(filepath.Dir(path), "index.md"))
			if err != nil {
				return err
			}
			parent, err = r.FindFileByRelativePath(parentRelativePath)
			if err != nil {
				return err
			}
		}

		file, err := NewOrExistingFile(parent, path)
		if err != nil {
			return err
		}

		if file.HasTag("ignore") {
			// Do not add to index files marked as ignorable
			return nil
		}

		if file.State() != None {
			if err := db.StageObject(file); err != nil {
				return fmt.Errorf("unable to stage modified object %s: %v", file, err)
			}
		}
		traversedObjects[file.UniqueOID()] = true
		if err := file.Save(); err != nil {
			return nil
		}

		for _, object := range file.SubObjects() {
			if _, found := traversedObjects[object.UniqueOID()]; found {
				// already processed
				continue
			}

			// Notes are processed in two passes
			if object.Kind() == "note" {
				if note, ok := object.(*Note); ok {
					traversedNotes = append(traversedNotes, note)
				}
			}

			if object.State() != None {
				if object.Kind() == "media" {
					unprocessedMedia := object.(*Media)
					if !unprocessedMedia.Dangling {
						unprocessedMedias = append(unprocessedMedias, unprocessedMedia)
					}
				}

				if err := db.StageObject(object); err != nil {
					return fmt.Errorf("unable to stage modified object %s: %v", object, err)
				}
			}
			traversedObjects[object.UniqueOID()] = true
			if err := object.Save(); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Generate blobs
	mediaJobs := make(chan *Media, len(unprocessedMedias))
	mediaResults := make(chan *Media, len(unprocessedMedias))
	countWorkers := CurrentConfig().ConfigFile.Medias.Parallel
	if countWorkers == 0 {
		countWorkers = 1
	}
	for w := 1; w <= countWorkers; w++ {
		go func(workerNum int, jobs <-chan *Media, results chan<- *Media) {
			for media := range jobs {
				CurrentLogger().Infof("[worker %d] Generating blobs for %s...\n", workerNum, media.RelativePath)
				media.UpdateBlobs()
				results <- media
			}
		}(w, mediaJobs, mediaResults)
	}
	for _, media := range unprocessedMedias {
		mediaJobs <- media
	}
	close(mediaJobs)
	// Then, wait for blob generation to end
	for i := 0; i < len(unprocessedMedias); i++ {
		mediaCompleted := <-mediaResults
		if err := mediaCompleted.InsertBlobs(); err != nil {
			return err
		}
		if err := db.StageObject(mediaCompleted); err != nil {
			return fmt.Errorf("unable to stage modified object %s: %v", mediaCompleted, err)
		}
	}

	// Find objects to delete for every path
	var deletions []StatefulObject
	for _, path := range paths {
		relpath, err := r.GetFileRelativePath(path)
		if err != nil {
			return err
		}
		pathDeletions, err := r.findObjectsLastCheckedBefore(buildTime, relpath)
		if err != nil {
			return err
		}
		deletions = append(deletions, pathDeletions...)
	}

	// Check for dead medias only when adding the root directory.
	// For example, when adding a file, it can contains references to medias stored in a directory outside the given path.
	if slices.Contains(paths, CurrentConfig().RootDirectory) { // ex: nt add .
		// As we walked the whole hierarchy, all medias must have be checked.
		mediaDeletions, err := CurrentRepository().FindMediasLastCheckedBefore(buildTime)
		if err != nil {
			return err
		}
		for _, mediaDeletion := range mediaDeletions {
			deletions = append(deletions, mediaDeletion)
		}
	}

	for _, deletion := range deletions {
		deletion.ForceState(Deleted)
		if err := deletion.Save(); err != nil {
			return err
		}
		if err := db.StageObject(deletion); err != nil {
			return fmt.Errorf("unable to stage deleted object %s: %v", deletion, err)
		}
	}

	// Second pass: Refresh all notes
	// Same logic but for dependent notes (avoid cycles)
	traversedRefreshedObjects := make(map[string]bool)

	var refreshDependencies func(oid string) error
	refreshDependencies = func(oid string) error {
		dependencies, err := r.FindRelationsTo(oid)
		if err != nil {
			return err
		}
		for _, relation := range dependencies {
			dependentObject, err := db.ReadLastStagedOrCommittedObjectFromDB(relation.SourceOID)
			if err != nil {
				return err
			}

			changed, err := dependentObject.Refresh()
			if err != nil {
				return err
			}
			if changed {
				if err := db.StageObject(dependentObject); err != nil {
					return fmt.Errorf("unable to stage modified dependent object %s: %v", dependentObject, err)
				}
				traversedRefreshedObjects[relation.SourceOID] = true
				if err := dependentObject.Save(); err != nil {
					return err
				}
				if err := refreshDependencies(relation.SourceOID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, note := range traversedNotes {
		CurrentLogger().Infof("Reprocessing note %s...", note)
		// Refresh content after having processed all notes (useful when a note include a note processed later)
		changed, err := note.Refresh()
		if err != nil {
			return err
		}
		// Save relations only now that we know existing dependencies really exist
		if err := r.UpdateRelations(note); err != nil {
			return err
		}
		if !changed {
			continue
		}
		if err := note.Save(); err != nil {
			return err
		}
		dependencies, err := r.FindRelationsTo(note.UniqueOID())
		if err != nil {
			return err
		}
		for _, relation := range dependencies {
			dependentObject, err := db.ReadLastStagedOrCommittedObjectFromDB(relation.SourceOID)
			if err != nil {
				return err
			}

			CurrentLogger().Infof("Reprocessing dependent object %s...", dependentObject)
			changed, err := dependentObject.Refresh()
			if err != nil {
				return err
			}
			if changed {
				if err := db.StageObject(dependentObject); err != nil {
					return fmt.Errorf("unable to stage modified dependent object %s: %v", dependentObject, err)
				}
				traversedRefreshedObjects[relation.SourceOID] = true
				if err := dependentObject.Save(); err != nil {
					return err
				}
				if err := r.UpdateRelations(dependentObject); err != nil {
					return err
				}
				if err := refreshDependencies(relation.SourceOID); err != nil {
					return err
				}
			}
		}
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := db.index.Save(); err != nil {
		return err
	}

	return nil
}

func (r *Repository) findObjectsLastCheckedBefore(buildTime time.Time, path string) ([]StatefulObject, error) {
	CurrentLogger().Debugf("Searching for %s", path)
	// Search for deleted objects...
	var deletions []StatefulObject

	links, err := CurrentRepository().FindLinksLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range links {
		deletions = append(deletions, object)
	}
	reminders, err := CurrentRepository().FindRemindersLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range reminders {
		deletions = append(deletions, object)
	}
	flashcards, err := CurrentRepository().FindFlashcardsLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range flashcards {
		deletions = append(deletions, object)
	}
	notes, err := CurrentRepository().FindNotesLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range notes {
		deletions = append(deletions, object)
	}
	files, err := CurrentRepository().FindFilesLastCheckedBefore(buildTime, path)
	if err != nil {
		return nil, err
	}
	for _, object := range files {
		deletions = append(deletions, object)
	}
	return deletions, nil
}

// Status displays current objects in staging area.
func (r *Repository) Status() (string, error) {
	// No side-effect with this command.
	// We only output results.
	var sb strings.Builder

	// Show staging area content
	sb.WriteString(`Changes to be committed:` + "\n")
	sb.WriteString(`  (use "nt restore..." to unstage)` + "\n")
	stagingArea := CurrentDB().index.StagingArea
	for _, obj := range stagingArea {
		sb.WriteString(fmt.Sprintf("\t%s:\t%s\n", obj.State, obj.Description))
	}

	// Show modified files not in staging area
	type ObjectStatus struct {
		RelativePath string
		OID          string
		Status       State
	}
	uncommittedFiles := make(map[string]ObjectStatus)

	root := CurrentConfig().RootDirectory
	err := r.walk([]string{root}, func(path string, stat fs.FileInfo) error {
		relpath, err := CurrentRepository().GetFileRelativePath(path)
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
	if err != nil {
		return "", err
	}

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

// Lint run linter rules on all files under the given paths.
func (r *Repository) Lint(ruleNames []string, paths ...string) (*LintResult, error) {
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

	paths = r.normalizePaths(paths...)
	err := r.walk(paths, func(path string, stat fs.FileInfo) error {
		CurrentLogger().Debugf("Processing %s...\n", path)

		// Work without the database
		file, err := ParseFile(path)
		if err != nil {
			return err
		}

		// Ignore ignorable files
		if file.HasTag("ignore") {
			return nil
		}

		// Check file
		violations, err := file.Lint(ruleNames)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			result.Append(violations...)
			result.AffectedFiles += 1
		}
		result.AnalyzedFiles += 1

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// CountObjectsByType returns the total number of objects for every type.
func (r *Repository) CountObjectsByType() (map[string]int, error) {
	// Count object per type
	countFiles, err := r.CountFiles()
	if err != nil {
		return nil, err
	}
	countNotes, err := r.CountNotes()
	if err != nil {
		return nil, err
	}
	countFlashcards, err := r.CountFlashcards()
	if err != nil {
		return nil, err
	}
	countMedias, err := r.CountMedias()
	if err != nil {
		return nil, err
	}
	countLinks, err := r.CountLinks()
	if err != nil {
		return nil, err
	}
	countReminders, err := r.CountReminders()
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"file":      countFiles,
		"note":      countNotes,
		"flashcard": countFlashcards,
		"media":     countMedias,
		"link":      countLinks,
		"reminder":  countReminders,
	}, nil
}

// Diff show changes between commits and working tree.
func (r *Repository) Diff(staged bool) (string, error) {
	// Enable dry-run mode to not generate blobs
	CurrentConfig().DryRun = true

	if staged {
		return CurrentDB().Diff()
	}

	// Any object not updated after this date will be considered as deletions
	buildTime := clock.Now()
	db := CurrentDB()
	path := CurrentConfig().RootDirectory

	// Keep notes of processed objects to avoid duplication of effort
	// when some objects like medias are referenced by different notes.
	traversedObjects := make(map[string]bool)

	// We will update the last-checked date of objects to find the deleted ones
	// and rollback the transaction to have no side-effects.
	err := db.BeginTransaction()
	if err != nil {
		return "", err
	}
	defer db.RollbackTransaction()

	// Traverse all given path to view note changes
	var updatedNotes []*Note
	err = r.walk([]string{path}, func(path string, stat fs.FileInfo) error {
		CurrentLogger().Debugf("Processing %s...\n", path)

		parentRelativePath, err := r.GetFileRelativePath(filepath.Join(filepath.Dir(path), "index.md"))
		if err != nil {
			return err
		}
		parent, err := r.FindFileByRelativePath(parentRelativePath)
		if err != nil {
			return err
		}

		file, err := NewOrExistingFile(parent, path)
		if err != nil {
			return err
		}

		if file.State() != None {
			for _, note := range file.GetNotes() {
				if note.State() != None {
					updatedNotes = append(updatedNotes, note)
				}
			}
		}
		traversedObjects[file.UniqueOID()] = true
		if err := file.Save(); err != nil { // to update last-checked timestamp to find deleted files later
			return nil
		}

		for _, object := range file.SubObjects() {
			if _, found := traversedObjects[object.UniqueOID()]; found {
				// already processed
				continue
			}
			traversedObjects[object.UniqueOID()] = true
			if err := object.Save(); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	// Find deleted notes for every path
	relpath, err := r.GetFileRelativePath(path)
	if err != nil {
		return "", err
	}
	deletedNotes, err := r.FindNotesLastCheckedBefore(buildTime, relpath)
	if err != nil {
		return "", err
	}

	var diff strings.Builder
	// Diff updated notes
	for _, noteAfter := range updatedNotes {
		objectBefore, err := db.ReadLastStagedOrCommittedObject(noteAfter.OID)
		if err != nil {
			return "", err
		}
		noteContentBefore := ""
		if objectBefore != nil {
			noteBefore := objectBefore.(*Note)
			noteContentBefore = noteBefore.ContentRaw
		}
		noteContentAfter := noteAfter.ContentRaw
		patch := godiffpatch.GeneratePatch(noteAfter.RelativePath, noteContentBefore, noteContentAfter)
		diff.WriteString(patch)
	}
	// Diff deleted notes
	for _, noteAfter := range deletedNotes {
		objectBefore, err := db.ReadLastStagedOrCommittedObject(noteAfter.OID)
		if err != nil {
			return "", err
		}
		noteBefore := objectBefore.(*Note)
		noteContentBefore := noteBefore.ContentRaw
		noteContentAfter := ""
		patch := godiffpatch.GeneratePatch(noteAfter.RelativePath, noteContentBefore, noteContentAfter)
		diff.WriteString(patch)
	}

	// Don't forget to rollback
	if err := db.RollbackTransaction(); err != nil {
		return "", err
	}

	return diff.String(), nil
}

/* Statistics */

type StatsInDB struct {
	Objects    map[string]int
	Kinds      map[NoteKind]int
	Tags       map[string]int
	Attributes map[string]int
	SizeKB     int64
}

func NewStatsInDBEmpty() *StatsInDB {
	return &StatsInDB{
		Objects: map[string]int{
			"file":      0,
			"note":      0,
			"flashcard": 0,
			"media":     0,
			"link":      0,
			"reminder":  0,
		},
		Kinds: map[NoteKind]int{
			KindFree:       0,
			KindReference:  0,
			KindNote:       0,
			KindFlashcard:  0,
			KindCheatsheet: 0,
			KindQuote:      0,
			KindJournal:    0,
			KindTodo:       0,
			KindArtwork:    0,
			KindSnippet:    0,
		},
		Tags:       map[string]int{},
		Attributes: map[string]int{},
		SizeKB:     0,
	}
}

// StatsInDB returns various statistics about the .nt/database.db file.
func (r *Repository) StatsInDB() (*StatsInDB, error) {
	dbPath := filepath.Join(CurrentConfig().RootDirectory, ".nt/database.db")

	// Ensure the objects directory exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Not exists (occurs before the first commit)
		return NewStatsInDBEmpty(), nil
	}

	// Count objects
	countObjectsInDB, err := r.CountObjectsByType()
	if err != nil {
		return nil, err
	}

	// Count notes
	countNotesInDB, err := r.CountNotesByKind()
	if err != nil {
		return nil, err
	}

	// Count tags
	countTagsInDB, err := r.CountTags()
	if err != nil {
		return nil, err
	}

	// Count attributes
	countAttributesInDB, err := r.CountAttributes()
	if err != nil {
		return nil, err
	}

	databaseSize, _ := filesystem.FileSize(dbPath)
	// Ignore error as file may not exist at first

	return &StatsInDB{
		Objects:    countObjectsInDB,
		Kinds:      countNotesInDB,
		Tags:       countTagsInDB,
		Attributes: countAttributesInDB,
		SizeKB:     databaseSize / filesystem.KB,
	}, nil
}

type Stats struct {
	OnDisk *StatsOnDisk
	InDB   *StatsInDB
}

// Stats returns various statitics about the storage.
func (r *Repository) Stats() (*Stats, error) {
	statsOnDisk, err := CurrentDB().StatsOnDisk()
	if err != nil {
		return nil, err
	}

	statsInDB, err := r.StatsInDB()
	if err != nil {
		return nil, err
	}

	return &Stats{
		OnDisk: statsOnDisk,
		InDB:   statsInDB,
	}, nil
}
