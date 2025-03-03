package main

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/index.go

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
	StagedPackFileOID oid.OID   `yaml:"staged_packfile_oid"`
	StagedMTime       time.Time `yaml:"staged_mtime"`
	StagedSize        int64     `yaml:"staged_size"`
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

func (i *IndexEntry) Stage(newPackFile *PackFile) {
	i.Staged = true
	i.StagedPackFileOID = newPackFile.OID
	i.StagedMTime = newPackFile.FileMTime
	i.StagedSize = newPackFile.FileSize
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
}

// ReadIndex loads the index file.
func ReadIndex() *Index {
	path := filepath.Join(CurrentRepository().Path, ".nt/index")
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return &Index{
			Entries: []*IndexEntry{},
		}
	}
	if err != nil {
		log.Fatalf("Unable to open index: %v", err)
	}
	index := new(Index)
	if err := index.Read(in); err != nil {
		log.Fatalf("Unable to read index: %v", err)
	}
	in.Close()
	return index
}

// Save persists the index on disk.
func (i *Index) Save() error {
	path := filepath.Join(CurrentRepository().Path, ".nt/index")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.Write(f)
}

// Read reads an index from the file.
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

// GetEntry returns the entry associated with a file path.
func (i *Index) GetEntry(path string) *IndexEntry { // TODO inline?
	for _, entry := range i.Entries {
		if entry.RelativePath == path {
			return entry
		}
	}
	return nil
}

// Modified returns true if the file has been modified since last indexation.
func (i *Index) Modified(relativePath string, mtime time.Time) bool { // TODO inline?
	entry := i.GetEntry(relativePath)
	if entry == nil {
		return true
	}
	return mtime.After(entry.MTime)
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
	}
	return nil
}

// Commit persists the staged changes to the index.
func (i *Index) Commit() error {
	for _, entry := range i.Entries {
		if entry.Staged {
			entry.Commit()
		}
	}
	return i.Save()
}
