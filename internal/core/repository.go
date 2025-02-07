package core

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

var (
	// Lazy-load configuration and ensure a single read
	repositoryOnce      resync.Once
	repositorySingleton *Repository
)

type Repository struct{}

func CurrentRepository() *Repository {
	repositoryOnce.Do(func() {
		repositorySingleton = NewRepository()
	})
	return repositorySingleton
}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) Close() {
	CurrentDB().Close()
}

func (r *Repository) Path() string {
	return CurrentConfig().RootDirectory
}

// GetNoteRelativePath converts a relative path from a note to a relative path from the repository root directory.
func (r *Repository) GetNoteRelativePath(fileRelativePath string, srcPath string) (string, error) {
	return filepath.Rel(r.Path(), filepath.Join(filepath.Dir(r.GetAbsolutePath(fileRelativePath)), srcPath))
}

// GetFileRelativePath converts a relative path of a file to a relative path from the repository.
func (r *Repository) GetFileRelativePath(fileAbsolutePath string) string {
	return RelativePath(r.Path(), fileAbsolutePath)
}

// GetFileAbsolutePath converts a relative path from the repository to an absolute path on disk.
func (r *Repository) GetFileAbsolutePath(fileRelativePath string) string {
	return filepath.Join(r.Path(), fileRelativePath)
}

// GetAbsolutePath converts a relative path from the repository to an absolute path on disk.
func (r *Repository) GetAbsolutePath(path string) string {
	if strings.HasPrefix(path, r.Path()) {
		return path
	}
	return filepath.Join(r.Path(), path)
}

/* Commands Helpers */

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

func (r *Repository) Walk(pathSpecs PathSpecs, fn func(md *markdown.File) error) error {
	config := CurrentConfig()

	var matchedFiles []string
	var fileInfos = make(map[string]*fs.FileInfo)
	var filePaths = make(map[string]string)

	filepath.WalkDir(CurrentConfig().RootDirectory+"/", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." || path == ".." {
			return nil
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}
		if dirname == ".git" {
			return fs.SkipDir
		}

		relativePath := CurrentRepository().GetFileRelativePath(path)
		if relativePath == "." || relativePath == ".." {
			return nil
		}

		if !pathSpecs.Match(relativePath) {
			return nil
		}

		if config.IgnoreFile.MustExcludeFile(relativePath, info.IsDir()) {
			return nil
		}

		// We look only for specific extension
		if !info.IsDir() && !config.ConfigFile.SupportExtension(relativePath) {
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
		fileInfos[relativePath] = &fileInfo
		filePaths[relativePath] = path
		matchedFiles = append(matchedFiles, relativePath)

		return nil
	})

	// Process the files but ensure index.md are processed before other files under this directory
	slices.SortFunc(matchedFiles, IndexFilesFirst)

	// Execute callbacks
	for _, relativePath := range matchedFiles {
		md, err := markdown.ParseFile(filePaths[relativePath])
		if err != nil {
			return err
		}

		frontMatter, err := NewAttributeSetFromMarkdown(md)
		if err != nil {
			return err
		}
		if frontMatter.Tags().Includes("ignore") {
			continue
		}

		if err := fn(md); err != nil {
			return err
		}
	}

	return nil
}

/* Commands */

// Lint run linter rules on all files under the given paths.
func (r *Repository) Lint(ruleNames []string, paths PathSpecs) (*LintResult, error) {
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

	err := r.Walk(paths, func(mdFile *markdown.File) error {
		CurrentLogger().Debugf("Processing %s...\n", mdFile.AbsolutePath)

		// Work without the database
		file, err := ParseFile(mdFile, nil) // TODO load parent first
		if err != nil {
			return err
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

// MustLint enforces linter rules are respected.
func (r *Repository) MustLint(paths PathSpecs) {
	// Start with command linter (do not stage invalid file)
	linterResult, err := r.Lint(nil, paths)
	if err != nil {
		log.Fatalf("Unable to run linter: %v", err)
	}
	if len(linterResult.Errors) > 0 {
		log.Fatalf("%d linter errors detected:\n%s", len(linterResult.Errors), linterResult)
	}
}

// Add implements the command `nt add`
func (r *Repository) Add(paths PathSpecs) error {
	r.MustLint(paths)

	db := CurrentDB()

	var traversedPaths []string
	var packFilesToUpsert []*PackFile
	var packFilesToDelete []*PackFile

	// Traverse all given paths to detected updated medias/files
	err := r.Walk(paths, func(mdFile *markdown.File) error {
		CurrentLogger().Debugf("Processing %s...\n", mdFile.AbsolutePath)

		relativePath := RelativePath(CurrentConfig().RootDirectory, mdFile.AbsolutePath)

		traversedPaths = append(traversedPaths, relativePath)

		// A Markdown file must be parsed again if
		// - The file was modified since the last known timestamp
		// - The parent file was modified since the last known timestamp (ex: new attribute to propagate)

		var mdParentFile *markdown.File
		parentEntry := CurrentIndex().GetParentEntry(relativePath)
		if parentEntry != nil {
			packFile, err := CurrentIndex().ReadPackFile(parentEntry.PackFileOID)
			if err != nil {
				return err
			}
			blobRef := packFile.FindFirstBlobWithMimeType("text/markdown")
			if blobRef != nil {
				blobData, err := CurrentIndex().ReadBlobData(blobRef.OID)
				if err != nil {
					return err
				}
				if mdFile, err := markdown.ParseFileFromBytes(parentEntry.RelativePath, blobData, parentEntry.MTime, parentEntry.Size); err != nil {
					return err
				} else {
					mdParentFile = mdFile
				}
			}
		}

		mdFileModified := CurrentIndex().Modified(relativePath, mdFile.MTime)
		mdParentFileModified := false
		if parentEntry != nil {
			mdParentFileModified = CurrentIndex().Modified(parentEntry.RelativePath, mdFile.MTime)
		}
		if !mdFileModified && !mdParentFileModified {
			// Nothing changed = Nothing to parse
			return nil
		}

		// Reparse the new version
		parsedFile, err := ParseFile(mdFile, mdParentFile)
		if err != nil {
			return err
		}

		// Start with medias (more error-prone) => Better to fail with minimal side-effects
		for _, parsedMedia := range parsedFile.Medias {
			// Check if media has already been processed
			if slices.Contains(traversedPaths, parsedMedia.RelativePath) {
				// Already referenced by another file
				continue
			}
			traversedPaths = append(traversedPaths, parsedMedia.RelativePath)

			// Check if media has changed since last indexation
			mediaFileModified := CurrentIndex().Modified(parsedMedia.RelativePath, parsedMedia.MTime)
			if !mediaFileModified {
				// Media has not changed
				continue
			}

			packMedia, err := NewPackFileFromParsedMedia(parsedMedia)
			if err != nil {
				return err
			}
			packFilesToUpsert = append(packFilesToUpsert, packMedia)
		}

		// Finish with the file (less error-prone)
		packFile, err := NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}
		packFilesToUpsert = append(packFilesToUpsert, packFile)

		return nil
	})
	if err != nil {
		return err
	}

	// Walk the index to identify old files
	err = db.Index().Walk(paths, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		// Ignore medias.
		if !strings.HasSuffix(entry.RelativePath, ".md") {
			// We may not have found reference to a media in the processed markdown files
			// but some markdown files outside the path specs may still reference it.
			// The command `nt gc` is used to reclaim orphan medias instead.
			return nil
		}

		if !slices.Contains(traversedPaths, entry.RelativePath) {
			packFile, err := CurrentIndex().ReadPackFile(entry.PackFileOID)
			if err != nil {
				return err
			}
			packFilesToDelete = append(packFilesToDelete, packFile)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// We saved pack files on disk before starting a new transaction to keep it short
	if err := db.BeginTransaction(); err != nil {
		return err
	}
	db.UpsertPackFiles(packFilesToUpsert...)
	db.DeletePackFiles(packFilesToDelete...)
	// TODO Create .bak if Commit fails?
	db.Index().Stage(packFilesToUpsert...)
	db.Index().Stage(packFilesToDelete...)

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := db.Index().Save(); err != nil {
		return err
	}

	return nil
}

// Reset implements the command `nt reset`
func (r *Repository) Reset(pathSpecs PathSpecs) error {
	packFilesOIDToRestore := []oid.OID{}
	packFilesOIDToDelete := []oid.OID{}
	CurrentIndex().Walk(pathSpecs, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		if entry.Staged {
			packFilesOIDToDelete = append(packFilesOIDToDelete, entry.StagedPackFileOID)
			if entry.PackFileOID != entry.StagedPackFileOID {
				// A previous packfile existed for this entry
				packFilesOIDToRestore = append(packFilesOIDToRestore, entry.PackFileOID)
			}
		}
		return nil
	})

	// Load pack files to restore/delete
	var packFilesToRestore []*PackFile
	var packFilesToDelete []*PackFile
	for _, packFileOID := range packFilesOIDToRestore {
		packFile, err := CurrentIndex().ReadPackFile(packFileOID)
		if err != nil {
			return err
		}
		packFilesToRestore = append(packFilesToRestore, packFile)
	}
	for _, packFileOID := range packFilesOIDToDelete {
		packFile, err := CurrentIndex().ReadPackFile(packFileOID)
		if err != nil {
			return err
		}
		packFilesToDelete = append(packFilesToDelete, packFile)
	}

	// Start with DB changes (more error-prone)
	db := CurrentDB()
	if err := db.BeginTransaction(); err != nil {
		return err
	}
	db.UpsertPackFiles(packFilesToRestore...)
	db.DeletePackFiles(packFilesToDelete...)

	// Rewrite index before committing
	if err := db.Index().Reset(pathSpecs); err != nil {
		return err
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}

	return nil
}

// Commit implements the command `nt commit`
func (r *Repository) Commit(msg string) error {
	if CurrentIndex().NothingToCommit() {
		return errors.New("nothing to commit (create/copy files and use \"nt add\" to track")
	}
	return CurrentIndex().Commit()
}

// Status displays current objects in staging area.
func (r *Repository) Status() (string, error) {
	// No side-effect with this command.
	// We only output results.
	var sb strings.Builder

	// TODO implement

	// Show staging area content
	sb.WriteString(`Changes to be committed:` + "\n")
	sb.WriteString(`  (use "nt restore..." to unstage)` + "\n")

	return sb.String(), nil
}

// Push pushes new objects remotely.
func (r *Repository) Push(interactive, force bool) error {
	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteWriter is a personal tool and you are not expected to push from two repositories at the same time.

	if CurrentIndex().SomethingToCommit() {
		return errors.New("changes not commited (commit first and retry)")
	}

	origin := CurrentDB().Origin()
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read the origin index
	data, err := origin.GetObject("index")

	originIndex := NewIndex()
	if errors.Is(err, ErrObjectNotExist) {
		// First time we push
	} else if err != nil {
		return err
	} else {
		if err := originIndex.Read(bytes.NewReader(data)); err != nil {
			return err
		}
		if originIndex == nil {
			return errors.New("failed to read origin index")
		}
		if !originIndex.CommittedAt.IsZero() && originIndex.CommittedAt.After(CurrentIndex().CommittedAt) && !force {
			return fmt.Errorf("failed to push to origin as index has been modified since")
		}
	}

	diff := originIndex.Diff(CurrentIndex())
	diffReverse := CurrentIndex().Diff(originIndex)

	for _, missingPackFile := range diff.MissingPackFiles {
		data, err := CurrentIndex().ReadPackFileData(missingPackFile.OID)
		if err != nil {
			return err
		}
		if err := origin.PutObject(missingPackFile.ObjectRelativePath(), data); err != nil {
			return err
		}
	}
	for _, missingBlob := range diff.MissingBlobs {
		data, err := CurrentIndex().ReadBlobData(missingBlob.OID)
		if err != nil {
			return err
		}
		if err := origin.PutObject(missingBlob.ObjectRelativePath(), data); err != nil {
			return err
		}
	}

	// Override origin index with the local one
	buf := new(bytes.Buffer)
	if err := CurrentIndex().Write(buf); err != nil {
		return err
	}
	if err := origin.PutObject("index", buf.Bytes()); err != nil {
		return err
	}

	// Cleanup obsolete files
	for _, missingPackFile := range diffReverse.MissingPackFiles {
		_ = origin.DeleteObject(missingPackFile.ObjectRelativePath())
		// Ignore error as the file may have been deleted in a prior execution
	}
	for _, missingBlob := range diffReverse.MissingBlobs {
		_ = origin.DeleteObject(missingBlob.ObjectRelativePath())
		// Ignore error as the file may have been deleted in a prior execution
	}

	return nil
}

// Pull retrieves remote objects.
func (r *Repository) Pull(interactive, force bool) error {
	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteWriter is a personal tool and you are not expected to push/pull at the same time.

	if CurrentIndex().SomethingToCommit() {
		return errors.New("changes not commited (commit first and retry)")
	}

	origin := CurrentDB().Origin()
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read the origin index
	data, err := origin.GetObject("index")

	originIndex := NewIndex()
	if errors.Is(err, ErrObjectNotExist) {
		// First time we push
	} else if err != nil {
		return err
	} else {
		if err := originIndex.Read(bytes.NewReader(data)); err != nil {
			return err
		}
		if originIndex == nil {
			return errors.New("failed to read origin index")
		}
		if !originIndex.CommittedAt.IsZero() && originIndex.CommittedAt.After(CurrentIndex().CommittedAt) && !force {
			return fmt.Errorf("failed to push to origin as index has been modified since")
		}
	}

	diff := CurrentIndex().Diff(originIndex)
	diffReverse := originIndex.Diff(CurrentIndex())

	for _, missingPackFile := range diff.MissingPackFiles {
		data, err := origin.GetObject(missingPackFile.ObjectRelativePath())
		if err != nil {
			return err
		}
		writeObject(missingPackFile, data)
	}
	for _, missingBlob := range diff.MissingBlobs {
		data, err := origin.GetObject(missingBlob.ObjectRelativePath())
		if err != nil {
			return err
		}
		writeObject(missingBlob, data)
	}

	// Override local index with the remote one
	if err := originIndex.Save(); err != nil {
		return err
	}
	if err := CurrentIndex().Reload(); err != nil {
		return err
	}

	// Cleanup obsolete files
	for _, missingPackFile := range diffReverse.MissingPackFiles {
		_ = CurrentDB().DeletePackFileOnDisk(missingPackFile)
		// Ignore error as the file may have been deleted in a prior
	}
	for _, missingBlob := range diffReverse.MissingBlobs {
		_ = CurrentDB().DeleteBlobOnDisk(missingBlob)
		// Ignore error as the file may have been deleted in a prior
	}

	return nil
}

/* Stats */

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
	countLinks, err := r.CountGoLinks()
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

	// TODO implement

	return "", nil
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

/* Helpers */

// TODO move elsewhere
func writeObject(ref ObjectRef, data []byte) error {
	return writeBytesToFile(ref.ObjectPath(), data)
}

func writeBytesToFile(filePath string, data []byte) error {
	// Create the file if it doesn't exist, or truncate it if it does
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the byte slice to the file
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
