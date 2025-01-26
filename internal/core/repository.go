package core

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
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
func (r *Repository) GetFileRelativePath(fileAbsolutePath string) string {
	relativePath, err := filepath.Rel(r.Path, fileAbsolutePath)
	if err != nil {
		// Must not happen (fail abruptly)
		log.Fatalf("Unable to determine relative path for %q from root %q: %v", fileAbsolutePath, r.Path, err)
	}
	return relativePath
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

func (r *Repository) Walk(pathSpecs PathSpecs, fn func(md *markdown.File) error) error {
	config := CurrentConfig()

	var matchedFiles []string
	var fileInfos = make(map[string]*fs.FileInfo)
	var filePaths = make(map[string]string)

	filepath.WalkDir(".", func(path string, info fs.DirEntry, err error) error {
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

		relativePath := CurrentRepository().GetFileRelativePath(path)

		if config.IgnoreFile.MustExcludeFile(relativePath, info.IsDir()) {
			return nil
		}

		// We look for only specific extension
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

		// TODO refactor markdown.FrontMatter.AsAttributeSet(BaseSchema).Tags().Include("ignore")
		// TODO refactor AttributeSet(markdown.FrontMatter).Tags().Include("ignore")
		// -> return AttributeSet() range markdown.FrontMatter) + Cast(ReservedAttributesDefinition)
		frontMatter, err := markdown.FrontMatter.AsMap() // Add methods for requires attributes
		if err != nil {
			return err
		}
		if value, ok := frontMatter["tags"]; ok {
			if typedValue, ok := CastAttribute(value, "[]string"); ok {
				if slices.Contains(typedValue.([]string), "ignore") {
					continue
				}
			}
		}

		if err := fn(md); err != nil {
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

// Commit implements the command `nt commit`
func (r *Repository) Commit(msg string) error {
	return nil
}

// Add implements the command `nt add`
func (r *Repository) Add(paths ...PathSpec) error {
	r.MustLint(paths...)

	// Any object not updated after this date will be considered as deletions
	buildTime := clock.Now()
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
		parentEntry := CurrentDB().Index().GetParentEntry(relativePath)
		if parentEntry != nil {
			packFile, err := parentEntry.ReadPackFile()
			if err != nil {
				return err
			}
			blobRef := packFile.FindFirstBlobWithMimeType("text/markdown")
			if blobRef != nil {
				blobData, err := CurrentDB().Index().ReadBlobData(blobRef.OID)
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

		mdFileModified := CurrentDB().Index().Modified(relativePath, mdFile.MTime)
		mdParentFileModified := false
		if parentEntry != nil {
			mdParentFileModified = CurrentDB().Index().Modified(parentEntry.RelativePath, mdFile.MTime)
		}
		if !mdFileModified && !mdParentFileModified {
			// Nothing changed = Nothing to parse
			return nil
		}

		// Reparse the new version
		parsedFile, err := ParseFile(CurrentConfig().RootDirectory, mdFile, mdParentFile)
		if err != nil {
			return err
		}

		packFile, err := r.NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}
		packFilesToUpsert = append(packFilesToUpsert, packFile)

		for _, parsedMedia := range parsedFile.Medias {
			// Check if media has already been processed
			if slices.Contains(traversedPaths, parsedMedia.RelativePath) {
				// Already referenced by another file
				continue
			}
			traversedPaths = append(traversedPaths, parsedMedia.RelativePath)

			// Check if media has changed since last indexation
			mediaFileModified := CurrentDB().Index().Modified(parsedMedia.RelativePath, parsedMedia.MTime)
			if !mediaFileModified {
				// Media has not changed
				continue
			}

			packMedia, err := r.NewPackFileFromParsedMedia(parsedMedia)
			if err != nil {
				return err
			}
			packFilesToUpsert = append(packFilesToUpsert, packMedia)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println(buildTime)

	// Walk the index to identify old files
	err = db.Index().Walk(paths, func(entry *IndexEntry) error {
		// Ignore medias.
		if !strings.HasSuffix(entry.RelativePath, ".md") {
			// We may not have found reference to a media in the processed markdown files
			// but some markdown files outside the path specs may still reference it.
			// The command `nt gc` is used to reclaim orphan medias instead.
			return nil
		}

		if !slices.Contains(traversedPaths, entry.RelativePath) {
			packFile, err := entry.ReadPackFile()
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

func (r *Repository) NewPackFileFromParsedFile(parsedFile *ParsedFile) (*PackFile, error) {
	// Use the hash of the parsed file as OID (if the file changes = new OID)
	oid := MustParseOID(parsedFile.Hash())

	// Check first if a previous execution already created the pack file
	// (ex: the command was aborted with Ctrl+C and restarted)
	existingPackFile, err := CurrentDB().ReadPackFileOnDisk(oid)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if existingPackFile != nil {
		return existingPackFile, nil
	}

	packFile := &PackFile{
		OID: oid,

		// Init file properties
		FileRelativePath: parsedFile.RelativePath,
		FileMTime:        parsedFile.Markdown.MTime,
		FileSize:         parsedFile.Markdown.Size,

		// Init pack file properties
		CTime: clock.Now(),
		MTime: clock.Now(),
	}

	// Create objects
	var objects []Object

	// Process the File
	file, err := NewOrExistingFile(NilOID, parsedFile)
	if err != nil {
		return nil, err
	}
	objects = append(objects, file)

	// Process the Note(s)
	for _, parsedNote := range parsedFile.Notes {
		note, err := NewOrExistingNote(packFile.OID, file, parsedNote)
		if err != nil {
			return nil, err
		}
		objects = append(objects, note)

		// Process the Flashcard
		if parsedNote.Flashcard != nil {
			parsedFlashcard := parsedNote.Flashcard
			flashcard, err := NewOrExistingFlashcard(packFile.OID, file, note, parsedFlashcard)
			if err != nil {
				return nil, err
			}
			objects = append(objects, flashcard)
		}

		// Process the Reminder(s)
		for _, parsedReminder := range parsedNote.Reminders {
			reminder, err := NewOrExistingReminder(packFile.OID, note, parsedReminder)
			if err != nil {
				return nil, err
			}
			objects = append(objects, reminder)
		}

		// Process the Golink(s)
		for _, parsedGoLink := range parsedNote.GoLinks {
			goLink, err := NewOrExistingGoLink(packFile.OID, note, parsedGoLink)
			if err != nil {
				return nil, err
			}
			objects = append(objects, goLink)
		}
	}

	// Fill the pack file
	for _, obj := range objects {
		if statefulObj, ok := obj.(StatefulObject); ok {
			if err := packFile.AppendObject(statefulObj); err != nil {
				return nil, err
			}
		}
		if fileObj, ok := obj.(FileObject); ok {
			if err := packFile.AppendBlobs(fileObj.Blobs()); err != nil {
				return nil, err
			}
		}
	}

	// Save the pack file on disk
	if err := packFile.Save(); err != nil {
		return nil, err
	}

	return packFile, nil
}

func (r *Repository) NewPackFileFromParsedMedia(parsedMedia *ParsedMedia) (*PackFile, error) {
	// Use the hash of the raw original media as OID (if the media is even slightly edited = new OID)
	oid := MustParseOID(parsedMedia.FileHash())

	// Check first if a previous execution already created the pack file
	// (ex: the command was aborted with Ctrl+C and restarted)
	existingPackFile, err := CurrentDB().ReadPackFileOnDisk(oid)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if existingPackFile != nil {
		return existingPackFile, nil
	}

	packFile := &PackFile{
		OID: oid,

		// Init file properties
		FileRelativePath: parsedMedia.RelativePath,
		FileMTime:        parsedMedia.MTime,
		FileSize:         parsedMedia.Size,

		// Init pack file properties
		CTime: clock.Now(),
		MTime: clock.Now(),
	}

	// Process the Media
	media, err := NewOrExistingMedia(packFile.OID, parsedMedia)
	if err != nil {
		return nil, err
	}

	// Fill the pack file
	if err := packFile.AppendObject(media); err != nil {
		return nil, err
	}
	if err := packFile.AppendBlobs(media.Blobs()); err != nil {
		return nil, err
	}

	// Save the pack file on disk
	if err := packFile.Save(); err != nil {
		return nil, err
	}

	return packFile, nil
}

func (r *Repository) MustLint(paths ...PathSpec) {
	// Start with command linter (do not stage invalid file)
	linterResult, err := r.Lint(nil, paths...)
	if err != nil {
		log.Fatalf("Unable to run linter: %v", err)
	}
	if len(linterResult.Errors) > 0 {
		log.Fatalf("%d linter errors detected:\n%s", len(linterResult.Errors), linterResult)
	}
}

// Status displays current objects in staging area.
func (r *Repository) Status() (string, error) {
	// No side-effect with this command.
	// We only output results.
	var sb strings.Builder

	// Show staging area content
	sb.WriteString(`Changes to be committed:` + "\n")
	sb.WriteString(`  (use "nt restore..." to unstage)` + "\n")

	return sb.String(), nil
}

// Lint run linter rules on all files under the given paths.
func (r *Repository) Lint(ruleNames []string, paths ...PathSpec) (*LintResult, error) {
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
		file, err := ParseFile(CurrentConfig().RootDirectory, mdFile, nil)
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
