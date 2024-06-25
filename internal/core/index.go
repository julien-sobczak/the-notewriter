package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"gopkg.in/yaml.v3"
)

// State describes an object status.
type State string

const (
	None     State = "none"
	Added    State = "added"
	Modified State = "modified"
	Deleted  State = "deleted"
)

/*
 * Index
 */

// Index
// See https://git-scm.com/docs/index-format for inspiration.
//
// The index file is used to determine if an object is new
// and to quickly locate which the commit file containing the object otherwise.
// Useful when adding or restoring objects.
type Index struct {
	Objects []*IndexObject `yaml:"objects" json:"objects"`
	// Same as objects when searching by OID
	objectsRef map[string]*IndexObject `yaml:"-" json:"-"`
	// Same as objects when searching by relative path
	filesRef map[string]*IndexObject `yaml:"-" json:"-"`

	// Mapping between pack files and their commit OID
	PackFiles map[string]string `yaml:"packfiles" json:"packfiles"`

	// A list of pack files that is known to be orphans
	OrphanPackFiles []*IndexOrphanPackFile `yaml:"orphan_packfiles" json:"orphan_packfiles"`

	// A list of blobs that is known to be orphans
	OrphanBlobs []*IndexOrphanBlob `yaml:"orphan_blobs" json:"orphan_blobs"`

	StagingArea StagingArea `yaml:"staging" json:"staging"`
}

type IndexObject struct {
	OID   string    `yaml:"oid" json:"oid"`
	Kind  string    `yaml:"kind" json:"kind"`
	MTime time.Time `yaml:"mtime" json:"mtime"`
	// The commit (and its packfile) containing the latest version (empty for uncommitted object)
	CommitOID   string `yaml:"commit_oid" json:"commit_oid"`
	PackFileOID string `yaml:"packfile_oid" json:"packfile_oid"`
}

type IndexOrphanPackFile struct {
	OID   string    `yaml:"oid" json:"oid"`
	DTime time.Time `yaml:"dtime" json:"dtime"`
}

type IndexOrphanBlob struct {
	OID   string    `yaml:"oid" json:"oid"`
	DTime time.Time `yaml:"dtime" json:"dtime"`
	// The media that introduced this blob
	MediaOID string `yaml:"media_oid" json:"media_oid"`
}

type StagingObject struct {
	PackObject
	PreviousCommitOID   string `yaml:"previous_commit_oid" json:"previous_commit_oid"`
	PreviousPackFileOID string `yaml:"previous_packfile_oid" json:"previous_packfile_oid"`
}

func (i IndexObject) String() string {
	return fmt.Sprintf("%s (%s)", i.Kind, i.OID)
}

func (s StagingObject) String() string {
	return fmt.Sprintf("%s %s (%s)", s.State, s.Kind, s.OID)
}

type StagingArea []*StagingObject

// ReadStagingObject searches for the given staging object in staging area
func (sa *StagingArea) ReadStagingObject(objectOID string) (*StagingObject, bool) {
	for _, obj := range *sa {
		if obj.OID == objectOID {
			return obj, true
		}
	}
	return nil, false
}

// ReadObject searches for the given object in staging area
func (sa *StagingArea) ReadObject(objectOID string) (StatefulObject, bool) {
	obj, ok := sa.ReadStagingObject(objectOID)
	if !ok {
		return nil, false
	}
	return obj.ReadObject(), true
}

// Contains file returns the staging object from a given file path.
func (sa *StagingArea) ContainsFile(relpath string) (*StagingObject, bool) {
	for _, obj := range *sa {
		if obj.Kind == "file" {
			file := new(File)
			obj.Data.Unmarshal(file)
			if file.RelativePath == relpath {
				return obj, true
			}
		}
	}
	return nil, false
}

// Count returns the number of objects inside the staging area.
func (sa *StagingArea) Count() int {
	return len(*sa)
}

// CountByState returns the number of objects inside the staging area in a given state.
func (sa *StagingArea) CountByState(state State) int {
	count := 0
	for _, obj := range *sa {
		if obj.State == state {
			count++
		}
	}
	return count
}

// NewIndex instantiates a new index.
func NewIndex() *Index {
	return &Index{
		objectsRef: make(map[string]*IndexObject),
		filesRef:   make(map[string]*IndexObject),
		PackFiles:  make(map[string]string),
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
	if index.objectsRef == nil {
		// Repopulate transient map
		index.objectsRef = make(map[string]*IndexObject)
		for _, object := range index.Objects {
			index.objectsRef[object.OID] = object
		}
	}
	if index.filesRef == nil {
		index.filesRef = make(map[string]*IndexObject)
	}
	in.Close()
	return index, nil
}

// CountChanges returns the number of changes currently present in the staging area.
func (i *Index) CountChanges() int {
	return i.StagingArea.Count()
}

// CloneForRemote prepares a cleaned index before a push.
func (i *Index) CloneForRemote() *Index {
	return &Index{
		Objects:         i.Objects,
		objectsRef:      i.objectsRef,
		filesRef:        i.filesRef,
		PackFiles:       i.PackFiles,
		OrphanPackFiles: i.OrphanPackFiles,
		OrphanBlobs:     i.OrphanBlobs,
		// IMPORTANT: StagingArea must not be pushed remotely
	}
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

// ReadIndexObject searches for the given object in the index.
func (i *Index) ReadIndexObject(objectOID string) (*IndexObject, bool) {
	obj, ok := i.objectsRef[objectOID]
	return obj, ok
}

// FindCommitContaining returns the commit associated with a given object.
func (i *Index) FindCommitContaining(objectOID string) (string, bool) {
	indexFile, ok := i.objectsRef[objectOID]
	if !ok {
		return "", false
	}
	return indexFile.CommitOID, true
}

// FindPackFileContaining returns the pack file associated with a given object.
func (i *Index) FindPackFileContaining(objectOID string) (string, bool) {
	indexFile, ok := i.objectsRef[objectOID]
	if !ok {
		return "", false
	}
	return indexFile.PackFileOID, true
}

// IsOrphanBlob checks if the blob has already beeing deleted.
func (i *Index) IsOrphanBlob(oid string) bool {
	for _, b := range i.OrphanBlobs {
		if b.OID == oid {
			return true
		}
	}
	// not found
	return false
}

// AppendPackFile completes the index with object from a pack file.
func (i *Index) AppendPackFile(commitOID string, packFile *PackFile) {
	for _, packObject := range packFile.PackObjects {
		i.putPackObject(commitOID, packFile.OID, packObject)
	}
	i.PackFiles[packFile.OID] = commitOID
}

// StageObject registers a changed object into the staging area
func (i *Index) StageObject(obj StatefulObject) error {
	objData, err := NewObjectData(obj)
	if err != nil {
		return err
	}

	// Update staging area
	newStagingObject := &StagingObject{
		PackObject: PackObject{
			OID:         obj.UniqueOID(),
			Kind:        obj.Kind(),
			State:       obj.State(),
			MTime:       obj.ModificationTime(),
			Description: obj.String(),
			Data:        objData,
		},
	}
	if commitObject, ok := i.objectsRef[obj.UniqueOID()]; ok {
		newStagingObject.PreviousCommitOID = commitObject.CommitOID
		newStagingObject.PreviousPackFileOID = commitObject.PackFileOID
	}

	// Check if object was already added
	for j, stagedObject := range i.StagingArea {
		if stagedObject.OID == newStagingObject.OID {
			// Do not update other properties
			// Ex: when staging a media after the generation of blobs,
			// the state must stay "added" even if the media has already been saved in database since.
			i.StagingArea[j].Data = newStagingObject.Data
			return nil
		}
	}

	// Otherwise, append the new object
	i.StagingArea = append(i.StagingArea, newStagingObject)

	return nil
}

// CreateCommit generates a new commit from current changes in the staging area.
func (i *Index) CreateCommitFromStagingArea() (*Commit, []*PackFile) {
	commit := NewCommit()

	// Group pack objects
	var packFiles []*PackFile

	// Rebuild a new pack file after every X objects
	packFile := NewPackFile()
	objectsInPackFile := 0

	for _, obj := range i.StagingArea {
		// Append to pack file
		packFile.AppendPackObject(&obj.PackObject)
		objectsInPackFile++

		// Register in index
		i.putPackObject(commit.OID, packFile.OID, &obj.PackObject)

		if objectsInPackFile == CurrentConfig().ConfigFile.Core.MaxObjectsPerPackFile {
			packFiles = append(packFiles, packFile)
			commit.PackFiles = append(commit.PackFiles, packFile.Ref())
			i.PackFiles[packFile.OID] = commit.OID
			// Start a new pack file
			packFile = NewPackFile()
			objectsInPackFile = 0
		}
	}
	if objectsInPackFile > 0 {
		packFiles = append(packFiles, packFile)
		commit.PackFiles = append(commit.PackFiles, packFile.Ref())
		i.PackFiles[packFile.OID] = commit.OID
	}

	// Clear the staging area
	i.StagingArea = nil

	return commit, packFiles
}

// putPackFile registers a new pack file inside the index.
func (i *Index) putPackFile(commitOID string, packFile *PackFile) {
	for _, packObject := range packFile.PackObjects {
		i.putPackObject(commitOID, packFile.OID, packObject)
	}
	i.PackFiles[packFile.OID] = commitOID
}

// putPackObject registers a new pack object inside the index.
func (i *Index) putPackObject(commitOID string, packFileOID string, obj *PackObject) {
	if indexObject, ok := i.objectsRef[obj.OID]; ok {
		// Simply updates the commit OID for existing objects
		indexObject.CommitOID = commitOID
		indexObject.PackFileOID = packFileOID
		return
	}

	indexObject := &IndexObject{
		OID:         obj.OID,
		Kind:        obj.Kind,
		MTime:       obj.MTime,
		CommitOID:   commitOID,
		PackFileOID: packFileOID,
	}

	// Update inner mappings
	i.objectsRef[obj.OID] = indexObject
	if obj.Kind == "file" {
		_, found := i.filesRef[obj.OID]
		// Update mapping path -> object
		if !found {
			file := new(File)
			obj.Data.Unmarshal(file)
			if file.RelativePath != "" {
				i.filesRef[file.RelativePath] = indexObject
			}
		}
	}

	i.Objects = append(i.Objects, indexObject)
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

type IndexDiff struct {
	// Objects present only in the compared index
	MissingObjects []*IndexObject
	// Pack Files no longer present in the compared index
	MissingOrphanPackFiles []string
	// Blobs no longer present in the compared index
	MissingOrphanBlobs []string
}

// Diff reports differences to reconcile the receiver with the given index.
func (i *Index) Diff(other *Index) *IndexDiff {
	result := new(IndexDiff)

	// Search for objects present in the given index and not present in the current index
	for _, objectOther := range other.Objects {
		if _, ok := i.objectsRef[objectOther.OID]; !ok {
			result.MissingObjects = append(result.MissingObjects, objectOther)
		}
	}
	// Search for orphan pack files present in the given index and not declared as such in the current index
	for _, orphanOther := range other.OrphanPackFiles {
		found := false
		for _, orphanLocal := range i.OrphanPackFiles {
			if orphanLocal.OID == orphanOther.OID {
				found = true
				break
			}
		}
		if !found {
			result.MissingOrphanPackFiles = append(result.MissingOrphanPackFiles, orphanOther.OID)
		}
	}
	// Search for orphan blobs present in the given index and not declared as such in the current index
	for _, orphanOther := range other.OrphanBlobs {
		found := false
		for _, orphanLocal := range i.OrphanBlobs {
			if orphanLocal.OID == orphanOther.OID {
				found = true
				break
			}
		}
		if !found {
			result.MissingOrphanBlobs = append(result.MissingOrphanBlobs, orphanOther.OID)
		}
	}

	return result
}

/*
 * Commit Graph
 */

// CommitGraph represents a .nt/objects/info/commit-graph file.
// See https://git-scm.com/docs/commit-graph for inspiration.
//
// The commit graph is used to quickly finds commit to download
// and/or diffs between local and remote directories.
// Useful when pulling or pushing commits.
type CommitGraph struct {
	UpdatedAt time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
	Commits   []*Commit `yaml:"commits,omitempty" json:"commits,omitempty"`
}

// NewCommitGraph instantiates a new commit graph.
func NewCommitGraph() *CommitGraph {
	return &CommitGraph{
		UpdatedAt: clock.Now(),
	}
}

// NewCommitGraphFromPath loads a commit-graph file from a path.
func NewCommitGraphFromPath(path string) (*CommitGraph, error) {
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return NewCommitGraph(), nil
	}
	if err != nil {
		return nil, err
	}
	cg := new(CommitGraph)
	if err := cg.Read(in); err != nil {
		return nil, err
	}
	in.Close()
	return cg, nil
}

// Read instantiates a commit graph from an existing file
func (cg *CommitGraph) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&cg)
	if err != nil {
		return err
	}
	return nil
}

// AppendCommit pushes a new commit.
func (c *CommitGraph) AppendCommit(commit *Commit) error {
	c.UpdatedAt = clock.Now()
	c.Commits = append(c.Commits, commit)
	return nil
}

// Ref returns the commit OID of the last commit.
func (c *CommitGraph) Ref() string {
	if len(c.Commits) == 0 {
		return ""
	}
	return c.Commits[len(c.Commits)-1].OID
}

// LastCommits returns all commits pushed after head.
func (c *CommitGraph) LastCommitsFrom(head string) ([]*Commit, error) {
	var results []*Commit

	found := false
	for _, commit := range c.Commits {
		if found {
			// Already found head = recent commit
			results = append(results, commit)
		}
		if commit.OID == head {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("unknown commit %s", head)
	}

	return results, nil
}

type CommitGraphDiff struct {
	// Commits present only in the compared commit graph
	MissingCommits []*Commit
	// Pack files present only in the compared commit graph
	MissingPackFiles PackFileRefs
	// Pack files no longer present in the compared commit graph
	ObsoletePackFiles PackFileRefs
	// Pack files edited since by a gc
	EditedPackFiles PackFileRefs
}

// Diff reports differences to reconcile the receiver with the given commit graph.
func (c *CommitGraph) Diff(other *CommitGraph) *CommitGraphDiff {
	result := new(CommitGraphDiff)

	for _, commitOther := range other.Commits {
		found := false
		for _, commitLocal := range c.Commits {
			if commitLocal.OID == commitOther.OID {
				found = true

				// Great, the commit is present on both sides.
				// Let's compare the content to see if GC has updated it
				for _, packFileOther := range commitOther.PackFiles {
					packFileLocal, ok := commitLocal.IncludePackFile(packFileOther.OID)
					if !ok {
						result.MissingPackFiles = append(result.MissingPackFiles, packFileOther)
						continue
					}
					if packFileLocal.MTime.Before(packFileOther.MTime) {
						result.EditedPackFiles = append(result.EditedPackFiles, packFileOther)
					}
				}
				for _, packFileLocal := range commitLocal.PackFiles {
					_, ok := commitOther.IncludePackFile(packFileLocal.OID)
					if !ok {
						result.ObsoletePackFiles = append(result.ObsoletePackFiles, packFileLocal)
					}
				}

				break
			}
		}
		if !found {
			result.MissingCommits = append(result.MissingCommits, commitOther)
		}
	}

	return result
}

// Dump must be used for debug purpose only.
func (c *CommitGraph) Dump() {
	fmt.Println("\n> Commit Graph:")
	for _, commit := range c.Commits {
		fmt.Printf("  - %s\n", commit.OID)
		for _, packFile := range commit.PackFiles {
			fmt.Printf("    %s (last: %s)\n", packFile.OID, packFile.MTime)
		}
	}
	fmt.Println()
}

// Write dumps the commit graph.
func (c *CommitGraph) Write(w io.Writer) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Save persists the commit-graph locally.
func (c *CommitGraph) Save() error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/info/commit-graph")
	return c.SaveTo(path)
}

// SaveTo persists the commit-graph to the given path.
func (c *CommitGraph) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Write(f)
}

/*
 * Commit
 */

type Commit struct {
	OID       string       `yaml:"oid" json:"oid"`
	CTime     time.Time    `yaml:"ctime" json:"ctime"`
	MTime     time.Time    `yaml:"mtime" json:"mtime"`
	PackFiles PackFileRefs `yaml:"packfiles" json:"packfiles"`
}

// NewCommit initializes a new empty commit.
func NewCommit() *Commit {
	return &Commit{
		OID:   NewOID(),
		CTime: clock.Now(),
		MTime: clock.Now(),
	}
}

// NewCommitFromPackFiles initializes a new commit referencing the given pack files.
func NewCommitFromPackFiles(packFiles ...*PackFile) *Commit {
	var packFilesRefs []*PackFileRef
	for _, packFile := range packFiles {
		packFilesRefs = append(packFilesRefs, packFile.Ref())
	}
	return &Commit{
		OID:       NewOID(),
		CTime:     clock.Now(),
		MTime:     clock.Now(),
		PackFiles: packFilesRefs,
	}
}

// NewCommitWithOID initializes a new commit with a given OID.
func NewCommitWithOID(oid string) *Commit {
	return &Commit{
		OID:   oid,
		CTime: clock.Now(),
		MTime: clock.Now(),
	}
}

// AppendPackFile appends a new pack file OID in the commit.
func (c *Commit) AppendPackFile(packFile *PackFile) {
	c.PackFiles = append(c.PackFiles, packFile.Ref())
}

// IncludePackFile returns the pack file is present in the commit.
func (c *Commit) IncludePackFile(oid string) (*PackFileRef, bool) {
	for _, packFile := range c.PackFiles {
		if packFile.OID == oid {
			return packFile, true
		}
	}
	return nil, false
}

func (c Commit) String() string {
	return fmt.Sprintf("%s (including %d pack files)", c.OID, len(c.PackFiles))
}
