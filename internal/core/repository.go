package core

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/remote"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
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
var IndexFilesFirst = func(a, b string) int {
	dirA := filepath.Dir(a)
	dirB := filepath.Dir(b)
	if dirA != dirB {
		return strings.Compare(a, b)
	}
	baseA := text.TrimExtension(filepath.Base(a))
	baseB := text.TrimExtension(filepath.Base(b))
	// move index files up
	if strings.EqualFold(baseA, "index") {
		return -1
	} else if strings.EqualFold(baseB, "index") {
		return 1
	}
	return strings.Compare(a, b) // os.WalkDir already returns file in lexical order
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
		md, err := markdown.ParseFileRaw(filePaths[relativePath])
		if err != nil {
			return err
		}

		// Check front matter for special file attributes
		frontMatter, err := NewAttributeSetFromMarkdown(md)
		if err != nil {
			return err
		}
		if frontMatter.Tags().Includes("ignore") {
			continue
		}
		if frontMatter.Tags().Includes("secure") {
			return fmt.Errorf("file %s is marked as secure but unencrypted", relativePath)
		}
		if md.Encrypted() {
			md, err = md.DecryptToMarkdown()
			if err != nil {
				return err
			}
		}

		if err := fn(md); err != nil {
			return err
		}
	}

	return nil
}

// IndexPackFiles indexes the given pack files into the repository.
func (r *Repository) IndexPackFiles(absolutePaths []string) error {
	// Step 1: Load pack files ensuring their proper location
	var packFilesToIndex []*PackFile
	for _, originalPath := range absolutePaths {
		fileBasename := filepath.Base(originalPath)
		fileName := text.TrimExtension(fileBasename)
		fileOID := oid.ParseOrNil(fileName)
		if fileOID.IsNil() {
			return fmt.Errorf("invalid pack file name %s", fileName)
		}
		destPath := PackFilePath(fileOID)
		if originalPath != destPath {
			// Copy the file to expected location
			if err := filesystem.CopyFileIfNotExists(originalPath, destPath); err != nil {
				return fmt.Errorf("failed to copy pack file to %s: %w", destPath, err)
			}
		}
		packFile, err := LoadPackFileFromPath(destPath)
		if err != nil {
			return fmt.Errorf("failed to load pack file %s: %w", destPath, err)
		}
		packFilesToIndex = append(packFilesToIndex, packFile)
	}

	// Step 2: Insert pack files to apply operations on stateful objects
	db := CurrentDB()
	if err := db.BeginTransaction(); err != nil {
		return err
	}
	// TODO add packfile to index with their ingestion time?
	if err := db.UpsertPackFiles(packFilesToIndex...); err != nil {
		return err
	}
	if err := db.CommitTransaction(); err != nil {
		return err
	}

	return nil
}

// Lint run linter rules on all files under the given paths.
func (r *Repository) Lint(paths PathSpecs, ruleNames []string) (*LintResult, error) {
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

		violationsFoundInFile := false

		// Check schema (if file matches a file type)
		violations, err := CheckSchema(file)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			violationsFoundInFile = true
			result.Append(violations...)
		}

		// Check attributes
		violations, err = CheckAttributes(file)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			violationsFoundInFile = true
			result.Append(violations...)
		}

		// Check file
		violations, err = file.Lint(ruleNames)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			violationsFoundInFile = true
			result.Append(violations...)
		}
		if violationsFoundInFile {
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
	linterResult, err := r.Lint(paths, nil)
	if err != nil {
		log.Fatalf("Unable to run linter: %v", err)
	}
	if len(linterResult.Errors) > 0 {
		log.Fatalf("%d linter errors detected:\n%s", len(linterResult.Errors), linterResult)
	}
}

type AddResult struct {
	// Pack files that were upserted
	Upserted []oid.OID
	// Pack files that were deleted
	Deleted []oid.OID
}

func (r *AddResult) Upsert(packFiles ...*PackFile) {
	for _, packFile := range packFiles {
		if !slices.Contains(r.Upserted, packFile.OID) {
			r.Upserted = append(r.Upserted, packFile.OID)
		}
	}
}

func (r *AddResult) Delete(packFiles ...*PackFile) {
	for _, packFile := range packFiles {
		if !slices.Contains(r.Deleted, packFile.OID) {
			r.Deleted = append(r.Deleted, packFile.OID)
		}
	}
}

// extractImplicitLinks creates implicit links based on note parent relationships.
// When a note has a parent (different heading levels), create a link from the parent to the child.
func extractImplicitLinks(parsedFile *ParsedFile, packFile *PackFile) ([]*Link, error) {
	var implicitLinks []*Link

	// Get all notes from the pack file
	packNotes, err := packFile.ReadNotes()
	if err != nil {
		return nil, err
	}

	// Create a map for O(1) lookup of notes by slug
	notesBySlug := make(map[string]*Note)
	for _, note := range packNotes {
		notesBySlug[note.Slug] = note
	}

	// Traverse parsed notes to check for Parent attribute
	for _, parsedNote := range parsedFile.Notes {
		if parsedNote.Parent == nil {
			continue
		}

		// Find the parent note using the map
		parentNote, ok := notesBySlug[parsedNote.Parent.Slug]
		if !ok {
			// Must not happen
			continue
		}

		// Find the child note using the map
		childNote, ok := notesBySlug[parsedNote.Slug]
		if !ok {
			// Must not happen
			continue
		}

		// Create implicit link: parent includes child
		implicitLinks = append(implicitLinks, &Link{
			SourceOID:  parentNote.OID,
			SourceKind: "note",
			TargetOID:  childNote.OID,
			TargetKind: "note",
			Type:       "includes",
		})
	}

	return implicitLinks, nil
}

// Add implements the command `nt add`
func (r *Repository) Add(paths PathSpecs) (*AddResult, error) {
	r.MustLint(paths)

	CurrentConfig().DryRun = false

	db := CurrentDB()

	var traversedPaths []string
	var referencedMediaPaths []string // List of media paths present in Markdown files
	var packFilesToUpsert []*PackFile
	var packFilesToDelete []*PackFile
	implicitLinksMap := make(map[oid.OID][]*Link) // Map from note OID to its implicit links

	// Traverse all given paths to detected updated medias/files
	err := r.Walk(paths, func(mdFile *markdown.File) error {
		CurrentLogger().Debugf("Processing %s...\n", mdFile.AbsolutePath)

		relativePath := RelativePath(CurrentConfig().RootDirectory, mdFile.AbsolutePath)

		mdMedias := mdFile.Body.ExtractInternalImages()
		// Convert to relative paths
		mdMedias, err := mdMedias.Transform(
			markdown.ResolveAbsoluteURL(mdFile.AbsolutePath),
			markdown.ResolveRelativeURL(CurrentConfig().RootDirectory),
		)
		if err != nil {
			return err
		}

		traversedPaths = append(traversedPaths, relativePath)
		referencedMediaPaths = append(referencedMediaPaths, mdMedias.URLs()...)

		// A Markdown file must be parsed again if
		// - The file was modified since the last known timestamp
		// - The parent file was modified since the last known timestamp (ex: new attribute to propagate)

		entry := CurrentIndex().GetEntry(relativePath)
		var mdParentFile *markdown.File
		parentEntry := CurrentIndex().GetParentEntry(relativePath)
		if parentEntry != nil {
			// Read on disk to have the latest version (NB: pack files are written on disk before being upserted in database)
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
				if mdFile, err := markdown.Parse(bytes.NewReader(blobData), markdown.FileInfo{
					Path:  parentEntry.RelativePath,
					MTime: parentEntry.MTime,
					Size:  parentEntry.Size,
				}); err != nil {
					return err
				} else {
					mdParentFile = mdFile
				}
			}
		}

		mdFileModified := true
		if entry != nil {
			mdFileModified = entry.ModifiedBefore(mdFile.MTime)
		}
		mdParentFileModified := false
		if parentEntry != nil {
			mdParentFileModified = parentEntry.ModifiedAfter(mdFile.MTime)
			// Ignore parent file that was modified after the current file
			// if the current file had been added after.
			if parentEntry.ModifiedBeforeLastIndexation(entry) {
				mdParentFileModified = false
			}
		}
		if !mdFileModified && !mdParentFileModified {
			// Nothing changed = Nothing to parse
			CurrentLogger().Debug("👌 Markdown file was not modified since last pack file")
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
			mediaFileModified := CurrentIndex().ModifiedBefore(parsedMedia.RelativePath, parsedMedia.MTime)
			if !mediaFileModified {
				CurrentLogger().Info("👌 Media was not modified since last pack file")
				// Media has not changed
				continue
			}

			packMedia, err := NewPackFileFromParsedMedia(parsedMedia)
			if err != nil {
				return err
			}
			packFilesToUpsert = append(packFilesToUpsert, packMedia)
			// Insert immediately like for Markdown files (not a requirement but to be consistent)
			if err := db.Index().Stage(packMedia); err != nil {
				return err
			}
		}

		// Finish with the file (less error-prone)
		packFile, err := NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}
		packFilesToUpsert = append(packFilesToUpsert, packFile)
		// Insert immediately the pack file (ex: index.md must be inserted for child files to find them)
		if err := db.Index().Stage(packFile); err != nil {
			return err
		}

		// Extract implicit links from parent-child note relationships
		implicitLinks, err := extractImplicitLinks(parsedFile, packFile)
		if err != nil {
			return err
		}
		for _, link := range implicitLinks {
			implicitLinksMap[link.SourceOID] = append(implicitLinksMap[link.SourceOID], link)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Walk the index to identify old files
	err = db.Index().Walk(paths, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		// Process medias only when adding all files ("nt add .")
		if !entry.MarkdownBased() { // medias
			if !paths.MatchAll() {
				// We may not have found reference to a media in the processed markdown files
				// but some markdown files outside the path specs may still reference it.
				// The command `nt gc` is used to reclaim orphan medias instead.
				return nil
			}

			if !slices.Contains(referencedMediaPaths, entry.RelativePath) {
				packFile, err := CurrentIndex().ReadPackFile(entry.PackFileOID)
				if err != nil {
					return err
				}
				packFilesToDelete = append(packFilesToDelete, packFile)
			}
		} else { // Markdown files
			if !slices.Contains(traversedPaths, entry.RelativePath) {
				packFile, err := CurrentIndex().ReadPackFile(entry.PackFileOID)
				if err != nil {
					return err
				}
				packFilesToDelete = append(packFilesToDelete, packFile)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// We saved pack files on disk before starting a new transaction to keep it short
	if err := db.BeginTransaction(); err != nil {
		return nil, err
	}
	if err := db.UpsertPackFiles(packFilesToUpsert...); err != nil {
		return nil, err
	}
	if err := db.DeletePackFiles(packFilesToDelete...); err != nil {
		return nil, err
	}
	if err := db.Index().Unstage(packFilesToDelete...); err != nil {
		return nil, err
	}

	// Update links for all objects in upserted pack files
	for _, packFile := range packFilesToUpsert {
		// Process notes with ReadNotes()
		notes, err := packFile.ReadNotes()
		if err != nil {
			return nil, err
		}
		for _, note := range notes {
			additionalLinks := implicitLinksMap[note.UniqueOID()]
			if err := r.UpdateLinks(note, additionalLinks); err != nil {
				return nil, err
			}
		}

		// Process other objects
		for _, packObject := range packFile.PackObjects {
			if packObject.Kind == "note" {
				// Already processed above
				continue
			}
			obj := packObject.Read()
			if obj, ok := obj.(Object); ok {
				if err := r.UpdateLinks(obj, nil); err != nil {
					return nil, err
				}
			}
		}
	}

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return nil, err
	}
	// And to persist the index
	if err := db.Index().Save(); err != nil {
		return nil, err
	}

	var result AddResult
	result.Upsert(packFilesToUpsert...)
	result.Delete(packFilesToDelete...)

	return &result, nil
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
func (r *Repository) Commit() error {
	idx := CurrentIndex()

	if idx.NothingToCommit() {
		return errors.New("nothing to commit (create/copy files and use \"nt add\" to track)")
	}

	// Run hooks
	idx.Walk(AnyPath, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		CurrentLogger().Debugf("Processing %s...\n", entry.RelativePath)
		if !entry.Staged || entry.HasTombstone() {
			// Run hooks only on staged, non-deleted entries
			return nil
		}

		// Extract the notes from the pack file
		packFile, err := idx.ReadPackFile(entry.PackFileOID)
		if err != nil {
			return err
		}
		for _, packObject := range packFile.PackObjects {
			if packObject.Kind == "note" {
				note := packObject.Read().(*Note)
				if note.HasHooks() {
					if err := note.RunHooks(nil, false); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})

	return idx.Commit()
}

type FileStatus struct {
	RelativePath    string
	Status          string
	ObjectsAdded    int
	ObjectsModified int
	ObjectsDeleted  int
}

type FileStatuses []*FileStatus

func (fs FileStatuses) Sort() {
	slices.SortFunc(fs, func(a, b *FileStatus) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
}

type StatusResult struct {
	ChangesStaged    FileStatuses
	ChangesNotStaged FileStatuses
}

// Status displays current objects in staging area.
func (r *Repository) Status(paths PathSpecs) (*StatusResult, error) {
	// No side-effect with this command
	CurrentConfig().DryRun = true

	// We only output results
	var result StatusResult

	index := CurrentIndex()

	// Staged changes are easy to determine using the index
	if index.SomethingToCommit() {
		// Traverse staging area content
		for _, entry := range index.SortedEntriesMatching(paths) {
			if entry.Staged {
				objectsAdded := 0
				objectsModified := 0
				objectsDeleted := 0
				verb := "modified"
				if entry.NeverCommitted() {
					verb = "added"
					objectsAdded += len(index.GetPackFileObjects(entry.StagedPackFileOID))
				} else if entry.HasTombstone() {
					verb = "deleted"
					objectsDeleted += len(index.GetPackFileObjects(entry.PackFileOID))
				}
				if verb == "modified" {
					// Check the pack file content to understand the number of changes
					packFile, err := index.ReadPackFile(entry.StagedPackFileOID)
					if err != nil {
						return nil, err
					}
					for _, object := range packFile.PackObjects {
						if !object.CTime.Before(packFile.CTime) {
							objectsModified += 1
						}
					}
				}
				result.ChangesStaged = append(result.ChangesStaged, &FileStatus{
					RelativePath:    entry.RelativePath,
					Status:          verb,
					ObjectsAdded:    objectsAdded,
					ObjectsModified: objectsModified,
					ObjectsDeleted:  objectsDeleted,
				})
			}
		}
	}

	// Finding changes not staged required to traverse the repository files
	var traversedEntryPaths []string

	entryVerbs := make(map[string]string) // path -> verb

	err := r.Walk(paths, func(mdFile *markdown.File) error {
		CurrentLogger().Debugf("Processing %s...\n", mdFile.AbsolutePath)

		relativePath := RelativePath(CurrentConfig().RootDirectory, mdFile.AbsolutePath)

		traversedEntryPaths = append(traversedEntryPaths, relativePath)

		// We ignore changes on parents

		mdFileModified := CurrentIndex().ModifiedBefore(relativePath, mdFile.MTime)
		if !mdFileModified {
			// Nothing changed = Nothing to parse
			return nil
		}
		parsedFile, err := ParseFile(mdFile, markdown.EmptyFile)
		if err != nil {
			return err
		}

		if !index.Exists(relativePath) {
			entryVerbs[relativePath] = "added"
		} else if index.ModifiedBefore(relativePath, mdFile.MTime) {
			entryVerbs[relativePath] = "modified"
		}

		// Don't forget referenced medias
		for _, parsedMedia := range parsedFile.Medias {
			// Check if media has already been processed
			if !slices.Contains(traversedEntryPaths, parsedMedia.RelativePath) {
				traversedEntryPaths = append(traversedEntryPaths, parsedMedia.RelativePath)
				// Check if media has changed since last indexation
				if !index.Exists(parsedMedia.RelativePath) {
					entryVerbs[parsedMedia.RelativePath] = "added"
				} else if index.ModifiedBefore(parsedMedia.RelativePath, parsedMedia.MTime) {
					entryVerbs[parsedMedia.RelativePath] = "modified"
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Walk the index to identify old files
	err = index.Walk(paths, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		// Ignore medias.
		if !strings.HasSuffix(entry.RelativePath, ".md") {
			// See comment in Add() method
			return nil
		}

		if !slices.Contains(traversedEntryPaths, entry.RelativePath) {
			entryVerbs[entry.RelativePath] = "deleted"
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	changesNotStaged := len(entryVerbs) > 0
	if changesNotStaged {
		var keys []string
		for key := range entryVerbs {
			keys = append(keys, key)
		}
		for _, relativePath := range sort.StringSlice(keys) {
			verb := entryVerbs[relativePath]
			result.ChangesNotStaged = append(result.ChangesNotStaged, &FileStatus{
				RelativePath: relativePath,
				Status:       verb,
			})
		}
	}

	result.ChangesNotStaged.Sort()
	result.ChangesStaged.Sort()

	return &result, nil
}

// Push pushes new objects remotely.
func (r *Repository) Push(remoteName string, interactive, force bool) error {
	CurrentConfig().DryRun = true

	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteWriter is a personal tool and you are not expected to push from two repositories at the same time.

	if CurrentIndex().SomethingToCommit() {
		return errors.New("changes not commited (commit first and retry)")
	}

	origin := CurrentDB().Remote(remoteName)
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read the origin index
	CurrentLogger().Trace("👀 Reading remote index...")
	data, err := origin.GetObject("index")

	originIndex := NewIndex()
	if errors.Is(err, remote.ErrObjectNotExist) {
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
		CurrentLogger().Tracef("👀 Pushing pack file %q...", missingPackFile.ObjectRelativePath())
		if err := origin.PutObject(missingPackFile.ObjectRelativePath(), data); err != nil {
			return fmt.Errorf("failed to put object %q: %v", missingPackFile.ObjectRelativePath(), err)
		}
		CurrentLogger().Infof("🚀 Pushed pack file %q", missingPackFile.ObjectRelativePath())
	}
	for _, missingBlob := range diff.MissingBlobs {
		data, err := CurrentIndex().ReadBlobData(missingBlob.OID)
		if err != nil {
			return err
		}
		CurrentLogger().Tracef("👀 Pushing blob %q...", missingBlob.ObjectRelativePath())
		if err := origin.PutObject(missingBlob.ObjectRelativePath(), data); err != nil {
			return fmt.Errorf("failed to put object %q: %v", missingBlob.ObjectRelativePath(), err)
		}
		CurrentLogger().Infof("🚀 Pushed blob %q", missingBlob.ObjectRelativePath())
	}

	// Override origin index with the local one
	buf := new(bytes.Buffer)
	if err := CurrentIndex().Write(buf); err != nil {
		return err
	}
	CurrentLogger().Tracef("👀 Pushing index...")
	if err := origin.PutObject("index", buf.Bytes()); err != nil {
		return fmt.Errorf("failed to put %q: %v", "index", err)
	}
	CurrentLogger().Infof("🚀 Pushed index")

	// Override origin config with the local one
	data, err = os.ReadFile(CurrentConfig().GetGeneratedPath())
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %v", CurrentConfig().GetGeneratedPath(), err)
	}
	CurrentLogger().Tracef("👀 Pushing config.json...")
	if err := origin.PutObject("config.json", data); err != nil {
		return fmt.Errorf("failed to put %q: %v", "config.json", err)
	}
	CurrentLogger().Info("🚀 Pushed config.json")

	// Cleanup obsolete files
	for _, missingPackFile := range diffReverse.MissingPackFiles {
		CurrentLogger().Tracef("👀 Trying to delete pack file %q...", missingPackFile.ObjectRelativePath())
		err = origin.DeleteObject(missingPackFile.ObjectRelativePath())
		if err == nil {
			CurrentLogger().Infof("🚀 Deleted pack file %q", missingPackFile.ObjectRelativePath())
		}
		// Ignore error as the file may have been deleted in a prior execution
	}
	for _, missingBlob := range diffReverse.MissingBlobs {
		CurrentLogger().Tracef("👀 Trying to delete blob %q...", missingBlob.ObjectRelativePath())
		err = origin.DeleteObject(missingBlob.ObjectRelativePath())
		if err == nil {
			CurrentLogger().Infof("🚀 Deleted blob %q", missingBlob.ObjectRelativePath())
		}
		// Ignore error as the file may have been deleted in a prior execution
	}

	return nil
}

// Pull retrieves remote objects.
func (r *Repository) Pull(remoteName string, interactive, force bool) error {
	CurrentConfig().DryRun = false

	// Implementation: We don't use a locking mechanism to prevent another repository to push at the same time.
	// The NoteWriter is a personal tool and you are not expected to push/pull at the same time.

	if CurrentIndex().SomethingToCommit() {
		return errors.New("changes not commited (commit first and retry)")
	}

	origin := CurrentDB().Remote(remoteName)
	if origin == nil {
		return errors.New("no remote found")
	}

	// Read the origin index
	data, err := origin.GetObject("index")

	originIndex := NewIndex()
	if errors.Is(err, remote.ErrObjectNotExist) {
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
	countLinks, err := r.CountGotos()
	if err != nil {
		return nil, err
	}
	countReminders, err := r.CountReminders()
	if err != nil {
		return nil, err
	}
	countMemories, err := r.CountMemories()
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
		"memory":    countMemories,
	}, nil
}

// Diff show changes between commits and working tree.
func (r *Repository) Diff(paths PathSpecs, staged bool) (ObjectDiffs, error) {
	// Enable dry-run mode to not generate blobs
	CurrentConfig().DryRun = true

	if staged {
		return r.diffStaged(paths)
	} else {
		return r.diffUnstaged(paths)
	}
}

func (r *Repository) diffStaged(paths PathSpecs) (ObjectDiffs, error) {
	index := CurrentIndex()

	if index.NothingToCommit() {
		return nil, nil
	}

	var result ObjectDiffs

	for _, entry := range index.SortedEntriesMatching(paths) {
		if !entry.Staged {
			continue
		}

		before := NilPackFile
		after := NilPackFile

		if entry.HasTombstone() || entry.AlreadyCommitted() {
			before = index.MustReadPackFile(entry.PackFileOID)
		}
		if !entry.HasTombstone() {
			after = index.MustReadPackFile(entry.StagedPackFileOID)
		}

		result = append(result, before.Diff(after)...)
	}

	return result, nil
}

func (r *Repository) diffUnstaged(paths PathSpecs) (ObjectDiffs, error) {
	// Finding changes not staged required to traverse the repository files
	var traversedEntryPaths []string

	index := CurrentIndex()

	var result ObjectDiffs

	err := r.Walk(paths, func(mdFile *markdown.File) error {
		CurrentLogger().Debugf("Processing %s...\n", mdFile.AbsolutePath)

		relativePath := RelativePath(CurrentConfig().RootDirectory, mdFile.AbsolutePath)

		traversedEntryPaths = append(traversedEntryPaths, relativePath)

		// We ignore changes on parents

		mdFileModified := index.ModifiedBefore(relativePath, mdFile.MTime)
		if !mdFileModified {
			// Nothing changed = Nothing to parse
			return nil
		}

		before := NilPackFile
		if index.Exists(relativePath) {
			before = index.MustReadLastPackFile(relativePath)
		}

		parsedFile, err := ParseFile(mdFile, markdown.EmptyFile)
		if err != nil {
			return err
		}
		after, err := NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}

		result = append(result, before.Diff(after)...)

		// Don't forget referenced medias
		for _, parsedMedia := range parsedFile.Medias {
			// Check if media has already been processed
			if slices.Contains(traversedEntryPaths, parsedMedia.RelativePath) {
				continue
			}
			traversedEntryPaths = append(traversedEntryPaths, parsedMedia.RelativePath)

			// Check if media has not changed
			if !index.ModifiedBefore(parsedMedia.RelativePath, parsedMedia.MTime) {
				continue
			}

			before := NilPackFile
			after, err := NewPackFileFromParsedMedia(parsedMedia)
			if err != nil {
				return err
			}

			if index.Exists(parsedMedia.RelativePath) {
				before = index.MustReadLastPackFile(parsedMedia.RelativePath)
			}

			result = append(result, before.Diff(after)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Walk the index to identify old files
	err = index.Walk(paths, func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error {
		// Ignore medias.
		if !strings.HasSuffix(entry.RelativePath, ".md") {
			// See comment in Add() method
			return nil
		}

		if !slices.Contains(traversedEntryPaths, entry.RelativePath) {
			before := index.MustReadLastPackFile(entry.RelativePath)
			after := NilPackFile
			result = append(result, before.Diff(after)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(result, func(a, b *ObjectDiff) int {
		return strings.Compare(a.RelativePath(), b.RelativePath())
	})

	return result, nil
}

/* Statistics */

type StatsInDB struct {
	Objects    map[string]int
	Types      map[string]int
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
			"memory":    0,
		},
		Types:      map[string]int{},
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
	countNotesInDB, err := r.CountNotesByTypes()
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
		Types:      countNotesInDB,
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
