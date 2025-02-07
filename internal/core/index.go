package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

/*
 * Index
 */

// Index
// See https://git-scm.com/docs/index-format for inspiration.
//
// The index file is used to determine if an object is new
// and to quickly locate the pack file containing the object otherwise.
// The index is useful when running any command.
//
// See also https://git-scm.com/book/en/v2/Git-Tools-Reset-Demystified
type Index struct {
	// Last commit date
	CommittedAt time.Time `yaml:"committed_at"`
	// List of files known in the index
	Entries []*IndexEntry `yaml:"entries"`
	// Cache of all objects present in the pack files
	Objects []*IndexObject `yaml:"objects"`
	// Cache of all blobs present in the pack files
	Blobs []*IndexBlob `yaml:"blobs"`
}

// IndexEntry is a file entry in the index.
type IndexEntry struct {
	// Path to the file in working directory
	RelativePath string `yaml:"relative_path"`

	// Pack file OID representing this file under .nt/objects
	PackFileOID oid.OID `yaml:"packfile_oid"`
	// File last modification date
	MTime time.Time `yaml:"mtime"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size" json:"size"`

	// True when a file has been staged
	Staged bool `yaml:"staged"`
	// Save but when the file has been staged (= different object under .nt/objects)
	StagedPackFileOID oid.OID `yaml:"staged_packfile_oid"`
	// Timestamp when the file has been detected as deleted
	StagedTombstone time.Time `yaml:"staged_tombstone"`
	StagedMTime     time.Time `yaml:"staged_mtime"`
	StagedSize      int64     `yaml:"staged_size"`
}

// NewIndexEntry creates a new index entry from a pack file.
func NewIndexEntry(packFile *PackFile) *IndexEntry {
	return &IndexEntry{
		PackFileOID:  packFile.OID,
		RelativePath: packFile.FileRelativePath,
		MTime:        packFile.FileMTime,
		Size:         packFile.FileSize,
	}
}

// String returns a string representation of the index entry.
func (i IndexEntry) String() string {
	tombstoneFlag := ""
	stagedFlag := ""
	if i.HasTombstone() {
		tombstoneFlag = "!"
	} else if i.Staged {
		stagedFlag = fmt.Sprintf(" => %s", i.StagedPackFileOID)
	}
	return fmt.Sprintf("entry %q (packfile: %s%s%s)", i.RelativePath, tombstoneFlag, i.PackFileOID, stagedFlag)
}

// Ref returns a PackFileRef from the index entry.
func (i *IndexEntry) Ref() PackFileRef {
	return PackFileRef{
		OID:          i.PackFileOID,
		RelativePath: i.RelativePath,
		CTime:        i.MTime,
	}
}

func (i *IndexEntry) Stage(newPackFile *PackFile) {
	i.Staged = true
	i.StagedPackFileOID = newPackFile.OID
	i.StagedMTime = newPackFile.FileMTime
	i.StagedSize = newPackFile.FileSize
	i.StagedTombstone = time.Time{} // Zero value
}

func (i *IndexEntry) NeverCommitted() bool {
	return i.PackFileOID == oid.Nil || i.PackFileOID == i.StagedPackFileOID
}

func (i *IndexEntry) Reset() {
	if !i.Staged {
		return
	}
	i.Staged = false
	i.StagedPackFileOID = ""
	i.StagedMTime = time.Time{}
	i.StagedSize = 0
	i.StagedTombstone = time.Time{}
	// Let the pack file to be garbage collected.
	// If the user add the file again, the pack file will already be on disk.
}

func (i *IndexEntry) Commit() {
	if !i.Staged {
		return
	}
	i.Staged = false
	i.PackFileOID = i.StagedPackFileOID
	i.MTime = i.StagedMTime
	i.Size = i.StagedSize
	// Clear staged values
	i.StagedPackFileOID = ""
	i.StagedMTime = time.Time{}
	i.StagedSize = 0
	i.StagedTombstone = time.Time{}
}

func (i *IndexEntry) SetTombstone() {
	i.Staged = true
	i.StagedTombstone = clock.Now()
}

func (i *IndexEntry) HasTombstone() bool {
	return i.Staged && !i.StagedTombstone.IsZero()
}

// MatchPathSpecs returns true if the entry matches any of the pathSpecs.
func (i *IndexEntry) MatchPathSpecs(pathSpecs PathSpecs) bool {
	// return true if relativePath matches any of the pathSpecs
	// PathSpecs could match parent directories.
	for _, pathSpec := range pathSpecs {
		if pathSpec.Match(i.RelativePath) {
			return true
		}
	}
	return false
}

// IndexObject represents a single object present in a pack file.
type IndexObject struct {
	OID         oid.OID `yaml:"oid"`
	Kind        string  `yaml:"kind"`
	PackFileOID oid.OID `yaml:"packfile_oid"`
}

// IndexBlob represents a single blob.
type IndexBlob struct {
	OID         oid.OID `yaml:"oid"`
	MimeType    string  `yaml:"mime" json:"mime"`
	PackFileOID oid.OID `yaml:"packfile_oid"`
}

// ObjectPath returns the path to the object on disk.
func (i *IndexBlob) Ref() BlobRef {
	return BlobRef{
		OID:      i.OID,
		MimeType: i.MimeType,
	}
}

// NewIndex instantiates a new index.
func NewIndex() *Index {
	return &Index{
		Entries: []*IndexEntry{},
		Objects: []*IndexObject{},
		Blobs:   []*IndexBlob{},
	}
}

// NewIndexFromPath loads an index file from a file.
func NewIndexFromPath(path string) (*Index, error) {
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return NewIndex(), nil
	}
	if err != nil {
		return nil, err
	}
	index := new(Index)
	if err := index.Read(in); err != nil {
		return nil, err
	}
	in.Close()
	return index, nil
}

// ReadIndex reads the index file from the current repository.
func ReadIndex() (*Index, error) {
	return NewIndexFromPath(CurrentRepository().GetAbsolutePath(".nt/index"))
}

// MustReadIndex reads the index file from the current repository or fails otherwise.
func MustReadIndex() *Index {
	index, err := ReadIndex()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/index file: %v", err)
		os.Exit(1)
	}
	return index
}

// Reload reloads the index from the file.
func (i *Index) Reload() error {
	newIndex, err := NewIndexFromPath(filepath.Join(CurrentConfig().RootDirectory, ".nt/index"))
	if err != nil {
		return err
	}
	*i = *newIndex
	return nil
}

// Read read an index from the file.
func (i *Index) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&i)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps the index to a file.
func (i *Index) Write(w io.Writer) error {
	data, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Save persists the index on disk.
func (i *Index) Save() error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/index")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.Write(f)
}

// GetEntryByPackFileOID returns the entry associated with a pack file OID.
func (i *Index) GetEntryByPackFileOID(oid oid.OID) (*IndexEntry, bool) {
	for _, entry := range i.Entries {
		if entry.PackFileOID == oid {
			return entry, true
		}
	}
	return nil, false
}

// GetEntry returns the entry associated with a file path.
func (i *Index) GetEntry(path string) *IndexEntry {
	for _, entry := range i.Entries {
		if entry.RelativePath == path {
			return entry
		}
	}
	return nil
}

// GetParentEntry returns the parent entry of a file
// (which may still not exist in the index).
// For example, when adding a new file, we need to locate the parent file to inherit attributes.
func (i *Index) GetParentEntry(relativePath string) *IndexEntry {
	// Search for a index.md in the same directory
	absolutePath := CurrentRepository().GetAbsolutePath(relativePath)
	currentDir := filepath.Dir(absolutePath)
	rootDir := CurrentConfig().RootDirectory
	for {
		if !strings.HasPrefix(currentDir, rootDir) {
			return nil
		}
		parentAbsolutePath := filepath.Join(currentDir, "index.md")
		parentRelativePath := CurrentRepository().GetFileRelativePath(parentAbsolutePath)
		if entry := i.GetEntry(parentRelativePath); entry != nil {
			if parentRelativePath != relativePath {
				return entry
			}
			// Edge case: the file is an index.md itself, move up
		}
		currentDir = filepath.Dir(currentDir)
	}
}

// Stage indexes new pack files.
// The pack files can match files already indexed by a previous pack file.
func (i *Index) Stage(packFiles ...*PackFile) error {
	for _, packFile := range packFiles {
		entry := i.GetEntry(packFile.FileRelativePath)
		if entry == nil {
			entry = NewIndexEntry(packFile)
			i.Entries = append(i.Entries, entry)
		}
		entry.Stage(packFile)
		// Update caches
		for _, packObject := range packFile.PackObjects {
			i.Objects = append(i.Objects, &IndexObject{
				OID:         packObject.OID,
				Kind:        packObject.Kind,
				PackFileOID: packFile.OID,
			})
		}
		for _, blob := range packFile.BlobRefs {
			i.Blobs = append(i.Blobs, &IndexBlob{
				OID:         blob.OID,
				MimeType:    blob.MimeType,
				PackFileOID: packFile.OID,
			})
		}
	}
	return nil
}

// SetTombstone marks an entry as deleted.
func (i *Index) SetTombstone(path string) error {
	entry := i.GetEntry(path)
	if entry == nil {
		return fmt.Errorf("no entry for %q", path)
	}
	entry.SetTombstone()
	return i.Save()
}

// Commit persists the staged changes to the index.
func (i *Index) Commit() error {
	newEntries := []*IndexEntry{}
	for _, entry := range i.Entries {
		if !entry.HasTombstone() {
			entry.Commit()
			newEntries = append(newEntries, entry)
		} else {
			// to delete
		}
	}
	i.Entries = newEntries
	i.clearCache()
	return i.Save()
}

// SomethingToCommit returns true if there are staged changes.
func (i *Index) SomethingToCommit() bool {
	for _, entry := range i.Entries {
		if entry.Staged {
			return true
		}
	}
	return false
}

// NothingToCommit returns true if there are no staged changes.
func (i *Index) NothingToCommit() bool {
	return !i.SomethingToCommit()
}

// clearCache removes objects and blobs not referenced by any pack file.
func (i *Index) clearCache() {
	// Collect pack file OIDs from all entries
	packFileOIDs := []oid.OID{}
	for _, entry := range i.Entries {
		packFileOIDs = append(packFileOIDs, entry.PackFileOID)
		if entry.Staged {
			packFileOIDs = append(packFileOIDs, entry.StagedPackFileOID)
		}
	}

	// Clear objects and blobs not referenced by any pack file
	newObjects := []*IndexObject{}
	newBlobs := []*IndexBlob{}
	for _, object := range i.Objects {
		if slices.Contains(packFileOIDs, object.PackFileOID) {
			newObjects = append(newObjects, object)
		}
	}
	for _, blob := range i.Blobs {
		if slices.Contains(packFileOIDs, blob.PackFileOID) {
			newBlobs = append(newBlobs, blob)
		}
	}
	i.Objects = newObjects
	i.Blobs = newBlobs
}

// Reset clears the staged changes.
func (i *Index) Reset(pathSpecs PathSpecs) error {
	var newEntries []*IndexEntry
	for _, entry := range i.Entries {
		if !entry.Staged {
			newEntries = append(newEntries, entry)
			continue
		}
		if entry.MatchPathSpecs(pathSpecs) {
			if entry.NeverCommitted() {
				// Entry has been staged but wasn't present before
				continue
			}
			entry.Reset()
			newEntries = append(newEntries, entry)
		}
	}
	i.Entries = newEntries
	i.clearCache()
	return i.Save()
}

// ObjectsDir returns the directory where objects are stored.
func (i *Index) ObjectsDir() string {
	return filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/")
}

// GC reclaims objects not referenced by any pack file in the current index.
func (i *Index) GC() error {
	// Collect known OIDs for lookup later
	knownOIDs := make(map[oid.OID]bool)
	for _, entry := range i.Entries {
		knownOIDs[entry.PackFileOID] = true
		if entry.Staged {
			knownOIDs[entry.StagedPackFileOID] = true
		}
	}
	for _, blob := range i.Blobs {
		knownOIDs[blob.OID] = true
	}

	// Directory to traverse
	dir := CurrentIndex().ObjectsDir()

	// Traverse the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file has a .blob or .pack extension
		if strings.HasSuffix(info.Name(), ".blob") || strings.HasSuffix(info.Name(), ".pack") {
			// Get the filename without the extension
			filenameWithoutExt := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

			// Check if the filename without the extension is in knownOIDs
			if _, exists := knownOIDs[oid.OID(filenameWithoutExt)]; !exists {
				// Delete the file if it is not in knownOIDs
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

/* Extracting objects */

// ReadPackFile reads a pack file from the index.
func (i *Index) ReadPackFile(oid oid.OID) (*PackFile, error) {
	for _, entry := range i.Entries {
		if entry.PackFileOID == oid {
			return LoadPackFileFromPath(PackFilePath(oid))
		}
	}
	return nil, nil
}

// ReadPackFileData reads a pack file from the index.
func (i *Index) ReadPackFileData(oid oid.OID) ([]byte, error) {
	entry, ok := i.GetEntryByPackFileOID(oid)
	if !ok {
		return nil, fmt.Errorf("packfile %q is unknown", oid)
	}
	data, err := os.ReadFile(PackFilePath(entry.PackFileOID))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ReadPackObject reads a pack object from the index.
func (i *Index) ReadPackObject(oid oid.OID) (*PackObject, error) {
	for _, object := range i.Objects {
		if object.OID == oid {
			packFile, err := i.ReadPackFile(object.PackFileOID)
			if err != nil {
				return nil, err
			}
			packObject, ok := packFile.GetPackObject(oid)
			if !ok {
				return nil, fmt.Errorf("missing object %q in pack file %q", oid, packFile.FileRelativePath)
			}
			return packObject, nil
		}
	}
	return nil, nil
}

// ReadObject reads an object from the index.
func (i *Index) ReadObject(oid oid.OID) (Object, error) {
	packObject, err := i.ReadPackObject(oid)
	if err != nil {
		return nil, err
	}
	return packObject.ReadObject(), nil
}

// ReadBlob reads a blob from the index.
func (i *Index) ReadBlob(oid oid.OID) (*BlobRef, error) {
	for _, blob := range i.Blobs {
		if blob.OID == oid {
			packFile, err := i.ReadPackFile(blob.PackFileOID)
			if err != nil {
				return nil, err
			}
			for _, blob := range packFile.BlobRefs {
				if blob.OID == oid {
					return blob, nil
				}
			}
			return nil, fmt.Errorf("missing blob %q in pack file %q", oid, packFile.FileRelativePath)
		}
	}
	return nil, nil
}

// ReadBlobData reads a blob from the index.
func (i *Index) ReadBlobData(oid oid.OID) ([]byte, error) {
	ref, err := i.ReadBlob(oid)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(ref.ObjectPath())
	if err != nil {
		return nil, err
	}
	return data, nil
}

/* Walk */

// getPackFileObjects returns all objects for a pack file.
func (i *Index) getPackFileObjects(packFileOID oid.OID) []*IndexObject {
	objects := []*IndexObject{}
	for _, object := range i.Objects {
		if object.PackFileOID == packFileOID {
			objects = append(objects, object)
		}
	}
	return objects
}

// getPackFileBlobs returns all blobs for a pack file.
func (i *Index) getPackFileBlobs(packFileOID oid.OID) []*IndexBlob {
	blobs := []*IndexBlob{}
	for _, blob := range i.Blobs {
		if blob.PackFileOID == packFileOID {
			blobs = append(blobs, blob)
		}
	}
	return blobs
}

// Walk iterates over the index entries matching one of the path specs and applies the given function.
func (i *Index) Walk(pathSpecs PathSpecs, fn func(entry *IndexEntry, objects []*IndexObject, blobs []*IndexBlob) error) error {
	for _, entry := range i.Entries {
		if pathSpecs.Match(entry.RelativePath) {
			objects := i.getPackFileObjects(entry.PackFileOID)
			blobs := i.getPackFileBlobs(entry.PackFileOID)
			err := fn(entry, objects, blobs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/* Utilities */

// Modified returns true if the file has been modified since last indexation.
func (i *Index) Modified(relativePath string, mtime time.Time) bool {
	entry := i.GetEntry(relativePath)
	if entry == nil {
		return true
	}
	return mtime.After(entry.MTime)
}

// ShortOID returns a short version of the OID based on the known OIDs.
func (idx *Index) ShortOID(oid oid.OID) string {
	knownOIDs := []string{}
	for _, entry := range idx.Entries {
		knownOIDs = append(knownOIDs, entry.PackFileOID.String())
		if entry.Staged {
			knownOIDs = append(knownOIDs, entry.StagedPackFileOID.String())
		}
	}
	for _, object := range idx.Objects {
		knownOIDs = append(knownOIDs, object.OID.String())
	}
	for _, blob := range idx.Blobs {
		knownOIDs = append(knownOIDs, blob.OID.String())
	}

	// Find the shortest unique prefix
	shortOID := ShortenToUniquePrefix(oid.String(), knownOIDs)
	// No
	if len(shortOID) < 4 { // Minimum 4 characters, like Git short OID
		return oid.String()[0:4]
	}
	return shortOID
}

// ShortenToUniquePrefix shortens a string to the shortest unique prefix based on a list of string values.
func ShortenToUniquePrefix(value string, knownValues []string) string {
	for length := 1; length <= len(value); length++ {
		prefix := value[:length]
		unique := true
		for _, knownValue := range knownValues {
			if len(knownValue) >= length && knownValue[:length] == prefix {
				unique = false
				break
			}
		}
		if unique {
			return prefix
		}
	}
	return value // If no unique prefix is found, return the original value
}

/* Diff */

type IndexDiff struct {
	MissingPackFiles PackFileRefs
	MissingBlobs     BlobRefs
}

// Diff compares two indexes and returns what is missing in the first one.
// Invert the arguments to get what is missing in the second one.
func (i *Index) Diff(remote *Index) *IndexDiff {
	knownPackFileOIDs := make(map[oid.OID]bool)
	knownBlobOIDs := make(map[oid.OID]bool)

	// Collect known OIDs for lookup
	for _, entry := range i.Entries {
		if entry.NeverCommitted() {
			continue
		}
		knownPackFileOIDs[entry.PackFileOID] = true
	}
	for _, blob := range i.Blobs {
		if _, ok := knownPackFileOIDs[blob.PackFileOID]; ok {
			knownBlobOIDs[blob.OID] = true
		}
	}

	var diff IndexDiff

	// Traverse remote index
	missingPackFilesOID := make(map[oid.OID]bool)
	for _, remoteEntry := range remote.Entries {
		if _, ok := knownPackFileOIDs[remoteEntry.PackFileOID]; !ok {
			// Pack file is missing
			diff.MissingPackFiles = append(diff.MissingPackFiles, remoteEntry.Ref())
			missingPackFilesOID[remoteEntry.PackFileOID] = true
		}
	}
	for _, remoteBlob := range remote.Blobs {
		if _, ok := missingPackFilesOID[remoteBlob.PackFileOID]; ok {
			// Blob is missing
			diff.MissingBlobs = append(diff.MissingBlobs, remoteBlob.Ref())
		}
	}

	return &diff
}
