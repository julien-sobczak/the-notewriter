package core

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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

/* Index */

// Index
// See https://git-scm.com/docs/index-format for inspiration.
//
// The index file is used to determine if an object is new
// and to quickly locate which the commit file containing the object otherwise.
// Useful when adding or restoring objects.
type Index struct {
	Objects []*IndexObject `yaml:"objects"`
	// Same as objects when searching by OID
	objectsRef map[string]*IndexObject `yaml:"-"`
	// Same as objects when searching by relative path
	filesRef map[string]*IndexObject `yaml:"-"`

	// Mapping between pack files and their commit OID
	PackFiles map[string]string `yaml:"packfiles"`

	// A list of pack files that is known to be orphans
	OrphanPackFiles []*IndexOrphanPackFile `yaml:"orphan_packfiles"`

	// A list of blobs that is known to be orphans
	OrphanBlobs []*IndexOrphanBlob `yaml:"orphan_blobs"`

	StagingArea StagingArea `yaml:"staging"`
}

type IndexObject struct {
	OID   string    `yaml:"oid"`
	Kind  string    `yaml:"kind"`
	MTime time.Time `yaml:"mtime"`
	// The commit (and its packfile) containing the latest version (empty for uncommitted object)
	CommitOID   string `yaml:"commit_oid"`
	PackFileOID string `yaml:"packfile_oid"`
}

type IndexOrphanPackFile struct {
	OID   string    `yaml:"oid"`
	DTime time.Time `yaml:"dtime"`
}

type IndexOrphanBlob struct {
	OID   string    `yaml:"oid"`
	DTime time.Time `yaml:"dtime"`
	// The media that introduced this blob
	MediaOID string `yaml:"media_oid"`
}

type StagingObject struct {
	PackObject
	PreviousCommitOID   string `yaml:"previous_commit_oid"`
	PreviousPackFileOID string `yaml:"previous_packfile_oid"`
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
			commit.PackFiles = append(commit.PackFiles, packFile.OID)
			i.PackFiles[packFile.OID] = commit.OID
			// Start a new pack file
			packFile = NewPackFile()
			objectsInPackFile = 0
		}
	}
	if objectsInPackFile > 0 {
		packFiles = append(packFiles, packFile)
		commit.PackFiles = append(commit.PackFiles, packFile.OID)
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

/* Commit Graph */

// CommitGraph represents a .nt/objects/info/commit-graph file.
// See https://git-scm.com/docs/commit-graph for inspiration.
//
// The commit graph is used to quickly finds commit to download
// and/or diffs between local and remote directories.
// Useful when pulling or pushing commits.
type CommitGraph struct {
	UpdatedAt time.Time `yaml:"updated_at,omitempty"`
	Commits   []*Commit `yaml:"commits,omitempty"`
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

// MissingCommitsFrom returns all commits present in origin and not present in current commit graph.
func (c *CommitGraph) MissingCommitsFrom(origin *CommitGraph) []*Commit {
	var results []*Commit

	for _, commitOrigin := range origin.Commits {
		found := false
		for _, commitLocal := range c.Commits {
			if commitLocal.OID == commitOrigin.OID {
				found = true
				break
			}
		}
		if !found {
			results = append(results, commitOrigin)
		}
	}

	return results
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
	dir := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/info/")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "commit-graph"))
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Write(f)
}

/* Commit */

type Commit struct {
	OID       string    `yaml:"oid"`
	CTime     time.Time `yaml:"ctime"`
	PackFiles []string  `yaml:"packfiles"`
}

type PackFile struct {
	OID         string        `yaml:"oid"`
	CTime       time.Time     `yaml:"ctime"`
	PackObjects []*PackObject `yaml:"objects"`
}

type PackObject struct {
	OID         string     `yaml:"oid"`
	Kind        string     `yaml:"kind"`
	State       State      `yaml:"state"` // (A) added, (D) deleted, (M) modified, (R) renamed
	MTime       time.Time  `yaml:"mtime"`
	Description string     `yaml:"desc"`
	Data        ObjectData `yaml:"data"`
}

// NewPackFile initializes a new empty pack file.
func NewPackFile() *PackFile {
	return &PackFile{
		OID:   NewOID(),
		CTime: clock.Now(),
	}
}

// ReadObject recreates the core object from a commit object.
func (p *PackObject) ReadObject() StatefulObject {
	switch p.Kind {
	case "file":
		file := new(File)
		p.Data.Unmarshal(file)
		return file
	case "flashcard":
		flashcard := new(Flashcard)
		p.Data.Unmarshal(flashcard)
		return flashcard
	case "note":
		note := new(Note)
		p.Data.Unmarshal(note)
		return note
	case "link":
		link := new(Link)
		p.Data.Unmarshal(link)
		return link
	case "media":
		media := new(Media)
		p.Data.Unmarshal(media)
		return media
	case "reminder":
		reminder := new(Reminder)
		p.Data.Unmarshal(reminder)
		return reminder
	}
	return nil
}

// ObjectData serializes any Object to base64 after zlib compression.
type ObjectData []byte // alias to serialize to YAML easily

// NewObjectData creates a compressed-string representation of the object.
func NewObjectData(obj Object) (ObjectData, error) {
	b := new(bytes.Buffer)
	if err := obj.Write(b); err != nil {
		return nil, err
	}
	in := b.Bytes()

	zb := new(bytes.Buffer)
	w := zlib.NewWriter(zb)
	w.Write(in)
	w.Close()
	return ObjectData(zb.Bytes()), nil
}

func (od ObjectData) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(od), nil
}

func (od *ObjectData) UnmarshalYAML(node *yaml.Node) error {
	value := node.Value
	ba, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	*od = ba
	return nil
}

func (od ObjectData) Unmarshal(target interface{}) error {
	if target == nil {
		return fmt.Errorf("cannot unmarshall in nil target")
	}
	src := bytes.NewReader(od)
	dest := new(bytes.Buffer)
	r, err := zlib.NewReader(src)
	if err != nil {
		return err
	}
	io.Copy(dest, r)
	r.Close()

	if f, ok := target.(*File); ok {
		f.Read(dest)
		return nil
	}
	if n, ok := target.(*Note); ok {
		n.Read(dest)
		return nil
	}
	if f, ok := target.(*Flashcard); ok {
		f.Read(dest)
		return nil
	}
	if m, ok := target.(*Media); ok {
		m.Read(dest)
		return nil
	}
	if l, ok := target.(*Link); ok {
		l.Read(dest)
		return nil
	}
	if r, ok := target.(*Reminder); ok {
		r.Read(dest)
		return nil
	}

	return fmt.Errorf("unsupported type %T", target)
}

// NewCommit initializes a new empty commit.
func NewCommit() *Commit {
	return &Commit{
		OID:   NewOID(),
		CTime: clock.Now(),
	}
}

// NewCommitWithOID initializes a new commit with a given OID.
func NewCommitWithOID(oid string) *Commit {
	return &Commit{
		OID:   oid,
		CTime: clock.Now(),
	}
}

func (c Commit) String() string {
	return fmt.Sprintf("%s (including %d pack files)", c.OID, len(c.PackFiles))
}

// NewPackFileFromPath reads a pack file file on disk or returns an empty instance.
func NewPackFileFromPath(path string) (*PackFile, error) {
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return NewPackFile(), nil
	}
	if err != nil {
		return nil, err
	}
	result := new(PackFile)
	if err := result.Read(in); err != nil {
		return nil, err
	}
	in.Close()
	return result, nil
}

// GetPackObject retrieves an object from a pack file.
func (p *PackFile) GetPackObject(oid string) (*PackObject, bool) {
	for _, object := range p.PackObjects {
		if object.OID == oid {
			return object, true
		}
	}
	return nil, false
}

// AppendPackObject registers a new stateful object inside the pack file.
func (p *PackFile) AppendPackObject(obj *PackObject) {
	p.PackObjects = append(p.PackObjects, obj)
}

// Append registers a new staged object inside a pack file.
func (p *PackFile) AppendStagingObject(obj *StagingObject) {
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:         obj.OID,
		Kind:        obj.Kind,
		State:       obj.State,
		MTime:       obj.MTime,
		Description: obj.Description,
		Data:        obj.Data,
	})
}

// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj StatefulObject) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:         obj.UniqueOID(),
		Kind:        obj.Kind(),
		State:       obj.State(),
		MTime:       obj.ModificationTime(),
		Description: obj.String(),
		Data:        data,
	})
	return nil
}

// UnmarshallObject extract a single object from a commit.
func (p *PackFile) UnmarshallObject(oid string, target interface{}) error {
	for _, objEdit := range p.PackObjects {
		if objEdit.OID == oid {
			return objEdit.Data.Unmarshal(target)
		}
	}
	return fmt.Errorf("no object with OID %q", oid)
}

// Merge tries to merge two pack files together by returning a new pack file
// containing the concatenation of both pack files.
func (p *PackFile) Merge(other *PackFile) (*PackFile, bool) {
	if len(p.PackObjects)+len(other.PackObjects) > CurrentConfig().ConfigFile.Core.MaxObjectsPerPackFile {
		return nil, false
	}
	result := NewPackFile()
	result.PackObjects = append(result.PackObjects, p.PackObjects...)
	result.PackObjects = append(result.PackObjects, other.PackObjects...)
	return result, true
}

/* Object */

func (p *PackFile) UniqueOID() string {
	return p.OID
}

// Read populates a pack file from an object file.
func (p *PackFile) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&p)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps a pack file to an object file.
func (p *PackFile) Write(w io.Writer) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Save writes a new pack file inside .nt/objects.
func (p *PackFile) Save() error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/"+OIDToPath(p.OID))
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Write(f)
}

func (p *PackFile) Blobs() []*BlobRef {
	// Blobs are stored outside packfiles.
	return nil
}

/* OID */

var oidGenerator OIDGenerator = &uniqueOIDGenerator{}

func NewOID() string {
	return oidGenerator.NewOID()
}
func NewOIDFromBytes(b []byte) string {
	return oidGenerator.NewOIDFromBytes(b)
}

/* Test */

type OIDGenerator interface {
	NewOID() string
	NewOIDFromBytes(b []byte) string
}

type uniqueOIDGenerator struct{}

// NewOID generates an OID.
// Every call generates a new unique OID.
func (g *uniqueOIDGenerator) NewOID() string {
	// We use the same "format" as Git (=40-length string) but use a content hash only for blob objects.
	// We use a randomly generated ID for other objects that is fixed even if objects are updated.

	// Ex (Git): 5e3f1b351782c017590b4b70fee709bf9c83b050
	// Ex (UUIDv4): 123e4567-e89b-12d3-a456-426655440000

	// Algorithm:
	// Remove `-` + add 8 random characters
	oid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:40]
	return oid
}

// NewOIDFromBytes generates an OID based on bytes.
// The same bytes will generate the same OID.
func (g *uniqueOIDGenerator) NewOIDFromBytes(b []byte) string {
	h := sha1.New()
	h.Write(b)

	// This gets the finalized hash result as a byte
	// slice. The argument to `Sum` can be used to append
	// to an existing byte slice: it usually isn't needed.
	bs := h.Sum(nil)

	// SHA1 values are often printed in hex, for example
	// in git commits. Use the `%x` format verb to convert
	// a hash results to a hex string.
	return fmt.Sprintf("%x\n", bs)
}

type suiteOIDGenerator struct {
	nextOIDs []string
}

func (g *suiteOIDGenerator) NewOID() string {
	return g.nextOID()
}

func (g *suiteOIDGenerator) NewOIDFromBytes(b []byte) string {
	return g.nextOID()
}

func (g *suiteOIDGenerator) nextOID() string {
	if len(g.nextOIDs) > 0 {
		oid, nextOIDs := g.nextOIDs[0], g.nextOIDs[1:]
		g.nextOIDs = nextOIDs
		return oid
	}
	panic("No more OIDs")
}

type fixedOIDGenerator struct {
	oid string
}

func (g *fixedOIDGenerator) NewOID() string {
	return g.oid
}

func (g *fixedOIDGenerator) NewOIDFromBytes(b []byte) string {
	return g.oid
}

type sequenceOIDGenerator struct {
	count int
}

func (g *sequenceOIDGenerator) NewOID() string {
	g.count++
	return fmt.Sprintf("%040d", g.count)
}

func (g *sequenceOIDGenerator) NewOIDFromBytes(b []byte) string {
	return NewOID()
}

// ResetOID restores the original unique OID generator.
// Useful in tests with a defer after overriding the default generator.
func ResetOID() {
	oidGenerator = &uniqueOIDGenerator{}
}
