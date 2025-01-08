package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	PackFileOID string `yaml:"packfile_oid"`
	// File last modification date
	MTime time.Time `yaml:"mtime"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size" json:"size"`

	// True when a file has been staged
	Staged bool `yaml:"staged"`
	// Save but when the file has been staged (= different object under .nt/objects)
	StagedPackFileOID string `yaml:"staged_packfile_oid"`
	// Timestamp when the file has been detected as deleted
	StagedTombstone time.Time `yaml:"staged_tombstone"`
	StagedMTime     time.Time `yaml:"staged_mtime"`
	StagedSize      int64     `yaml:"staged_size"`
}

func NewIndexEntry(packFile *PackFile) *IndexEntry {
	return &IndexEntry{
		PackFileOID:  packFile.OID,
		RelativePath: packFile.FileRelativePath,
		MTime:        packFile.FileMTime,
		Size:         packFile.FileSize,
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
	return i.PackFileOID == i.StagedPackFileOID
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
	i.StagedTombstone = time.Now()
}

// MatchPathSpecs returns true if the entry matches any of the pathSpecs.
func (i *IndexEntry) MatchPathSpecs(pathSpecs []string) bool {
	// return true if relativePath matches any of the pathSpecs
	// PathSpecs could match parent directories.
	for _, pathSpec := range pathSpecs {
		if i.RelativePath == pathSpec {
			return true
		}
		if strings.HasPrefix(i.RelativePath, pathSpec+"/") {
			return true
		}
	}
	return false
}

// IndexObject represents a single object present in a pack file.
type IndexObject struct {
	OID         string `yaml:"oid"`
	Kind        string `yaml:"kind"`
	PackFileOID string `yaml:"packfile_oid"`
}

// IndexBlob represents a single blob.
type IndexBlob struct {
	OID         string `yaml:"oid"`
	MimeType    string `yaml:"mime" json:"mime"`
	PackFileOID string `yaml:"packfile_oid"`
}

// ReadIndex reads the index file from the current repository.
func ReadIndex() *Index {
	index, err := NewIndexFromPath(CurrentRepository().GetAbsolutePath(".nt/index"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read current .nt/index file: %v", err)
		os.Exit(1)
	}
	return index
}

// NewIndex instantiates a new index.
func NewIndex() *Index {
	return &Index{
		Entries: []*IndexEntry{},
		Objects: []*IndexObject{},
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

func (i *Index) GetEntryByPackFileOID(oid string) (*IndexEntry, bool) {
	for _, entry := range i.Entries {
		if entry.PackFileOID == oid {
			return entry, true
		}
	}
	return nil, false
}

func (i *Index) GetEntry(path string) *IndexEntry {
	for _, entry := range i.Entries {
		if entry.RelativePath == path {
			return entry
		}
	}
	return nil
}

// Stage indexes a new pack file.
// The pack file can match a file already indexed by a previous pack file.
func (i *Index) Stage(packFile *PackFile) error {
	entry := i.GetEntry(packFile.FileRelativePath)
	if entry == nil {
		entry = NewIndexEntry(packFile)
		i.Entries = append(i.Entries, entry)
	}
	entry.Stage(packFile)
	return i.Save()
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
	for _, entry := range i.Entries {
		entry.Commit()
	}
	return i.Save()
}

// Reset clears the staged changes.
func (i *Index) Reset(pathSpecs ...string) error {
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
	return i.Save()
}

/* Extracting objects */

// ReadPackFile reads a pack file from the index.
func (i *Index) ReadPackFile(oid string) (*PackFile, error) {
	for _, entry := range i.Entries {
		if entry.PackFileOID == oid {
			return NewPackFileFromPath(entry.RelativePath)
		}
	}
	return nil, nil
}

// ReadPackObject reads a pack object from the index.
func (i *Index) ReadPackObject(oid string) (*PackObject, error) {
	for _, object := range i.Objects {
		if object.OID == oid {
			packFile, err := NewPackFileFromPath(object.PackFileOID)
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
func (i *Index) ReadObject(oid string) (Object, error) {
	packObject, err := i.ReadPackObject(oid)
	if err != nil {
		return nil, err
	}
	return packObject.ReadObject(), nil
}

// ReadBlob reads a blob from the index.
func (i *Index) ReadBlob(oid string) (*BlobRef, error) {
	for _, blob := range i.Blobs {
		if blob.OID == oid {
			packFile, err := NewPackFileFromPath(blob.PackFileOID)
			if err != nil {
				return nil, err
			}
			for _, packObject := range packFile.PackObjects {
				media := packObject.ReadObject().(*Media)
				for _, blob := range media.BlobRefs {
					if blob.OID == oid {
						return blob, nil
					}
				}
			}
			return nil, fmt.Errorf("missing blob %q in pack file %q", oid, packFile.FileRelativePath)
		}
	}
	return nil, nil
}

/* Walk */

// Walk iterates over the index entries matching one of the path specs and applies the given function.
func (i *Index) Walk(pathSpecs PathSpecs, fn func(entry *IndexEntry)) {
	for _, entry := range i.Entries {
		if pathSpecs.Match(entry.RelativePath) {
			fn(entry)
		}
	}
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
