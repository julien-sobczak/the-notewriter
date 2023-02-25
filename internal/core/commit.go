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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"gopkg.in/yaml.v3"
)

// State describes an object status.
type State string

const (
	None     State = "none"
	Added    State = "added"
	Modified State = "modified"
	Deleted  State = "deleted"
	Renamed  State = "renamed"
)

/* Index */

// Index
// See https://git-scm.com/docs/index-format for inspiration.
//
// The index file is used to determine if an object is new
// and to quickly locate which the commit file containing the object otherwise.
// Useful when adding or restoring objects.
type Index struct {
	Objects     []*IndexObject          `yaml:"objects"`
	objectsRef  map[string]*IndexObject `yaml:"-"`
	StagingArea StagingArea             `yaml:"staging"`
}

type IndexObject struct {
	OID   string    `yaml:"oid"`
	MTime time.Time `yaml:"mtime"`
	// The commit containing the latest version (empty for uncommitted object)
	CommitOID string `yaml:"commit_oid"`
}

type StagingObject struct {
	CommitObject
	PreviousCommitOID string `yaml:"previous_commit_oid"`
}

type StagingArea struct {
	Added    []*StagingObject `yaml:"added"`
	Modified []*StagingObject `yaml:"modified"`
	Deleted  []*StagingObject `yaml:"edited"`
}

// NewIndex instantiates a new index.
func NewIndex() *Index {
	return &Index{
		objectsRef: make(map[string]*IndexObject),
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

// FindCommitContaining returns the commit associated with a given object.
func (i *Index) FindCommitContaining(objectOID string) (string, bool) {
	indexFile, ok := i.objectsRef[objectOID]
	if !ok {
		return "", false
	}
	return indexFile.CommitOID, true
}

// AppendCommit completes the index with object from a commit.
func (i *Index) AppendCommit(c *Commit) {
	for _, objectCommit := range c.Objects {
		objectFile, ok := i.objectsRef[objectCommit.OID]
		if ok {
			// Update index object if updated since
			if objectFile.MTime.Before(objectCommit.MTime) {
				objectFile.MTime = objectCommit.MTime
			}
		} else {
			// Save the object into the index
			objectIndex := &IndexObject{
				OID:       objectCommit.OID,
				MTime:     objectCommit.MTime,
				CommitOID: c.OID,
			}
			i.Objects = append(i.Objects, objectIndex)
			i.objectsRef[objectCommit.OID] = objectIndex
		}
	}
}

// StageObject registers a changed object into the staging area
func (i *Index) StageObject(obj Object) error {
	objData, err := NewObjectData(obj)
	if err != nil {
		return err
	}

	// Update index
	indexObject, ok := i.objectsRef[obj.UniqueOID()]
	if ok {
		indexObject.MTime = obj.ModificationTime()
	} else {
		indexObject := &IndexObject{
			OID:       obj.UniqueOID(),
			MTime:     obj.ModificationTime(),
			CommitOID: "staging",
		}
		i.objectsRef[obj.UniqueOID()] = indexObject
		i.Objects = append(i.Objects, indexObject)
	}

	// Update staging area
	stagingObject := &StagingObject{
		CommitObject: CommitObject{
			OID:   obj.UniqueOID(),
			Kind:  obj.Kind(),
			State: obj.State(),
			MTime: obj.ModificationTime(),
			Data:  objData,
		},
	}
	switch obj.State() {
	case Added:
		i.StagingArea.Added = append(i.StagingArea.Added, stagingObject)
	case Renamed:
		fallthrough
	case Modified:
		i.StagingArea.Modified = append(i.StagingArea.Modified, stagingObject)
	case Deleted:
		i.StagingArea.Deleted = append(i.StagingArea.Deleted, stagingObject)
	}

	return nil
}

// CreateCommit generates a new commit from current changes in the staging area.
func (i *Index) CreateCommitFromStagingArea() *Commit {
	c := NewCommit()

	for _, o := range i.StagingArea.Added {
		c.Append(o.OID, o.Kind, o.State, o.MTime, o.Data)
		i.objectsRef[o.OID].CommitOID = c.OID
	}
	for _, o := range i.StagingArea.Modified {
		c.Append(o.OID, o.Kind, o.State, o.MTime, o.Data)
		i.objectsRef[o.OID].CommitOID = c.OID
	}
	for _, o := range i.StagingArea.Deleted {
		c.Append(o.OID, o.Kind, o.State, o.MTime, o.Data)
		i.objectsRef[o.OID].CommitOID = c.OID
	}

	// Clear the staging area
	i.StagingArea.Added = nil
	i.StagingArea.Modified = nil
	i.StagingArea.Deleted = nil

	return c
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
	UpdatedAt  time.Time `yaml:"updated_at,omitempty"`
	CommitOIDs []string  `yaml:"commits,omitempty"`
}

// NewCommitGraph instantiates a new commit graph.
func NewCommitGraph() *CommitGraph {
	return &CommitGraph{
		UpdatedAt: clock.Now(),
	}
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
func (c *CommitGraph) AppendCommit(childOID, parentOID string) error {
	var head = ""
	if len(c.CommitOIDs) > 0 {
		head = c.CommitOIDs[len(c.CommitOIDs)-1]
	}
	if head != parentOID {
		return fmt.Errorf("invalid head reference %s", head)
	}
	c.UpdatedAt = clock.Now()
	c.CommitOIDs = append(c.CommitOIDs, childOID)
	return nil
}

// LastCommits returns all recent commits.
func (c *CommitGraph) LastCommitsFrom(head string) ([]string, error) {
	var results []string

	found := false
	for _, commitOID := range c.CommitOIDs {
		if found {
			results = append(results, commitOID)
		}
		if commitOID == head {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("unknown commit %s", head)
	}

	return results, nil
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

/* Commit */

type Commit struct {
	OID     string         `yaml:"oid"`
	CTime   time.Time      `yaml:"ctime"`
	Objects []CommitObject `yaml:"objects"`
}

type CommitObject struct {
	OID   string     `yaml:"oid"`
	Kind  string     `yaml:"kind"`
	State State      `yaml:"state"` // (A) added, (D) deleted, (M) modified, (R) renamed
	MTime time.Time  `yaml:"mtime"`
	Data  ObjectData `yaml:"data"`
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

	if c, ok := target.(*Collection); ok {
		c.Read(dest)
		return nil
	}
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

// Append registers a new object inside the commit.
func (c *Commit) AppendObject(obj Object, state State) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	c.Objects = append(c.Objects, CommitObject{
		OID:   obj.UniqueOID(),
		Kind:  obj.Kind(),
		State: state,
		MTime: obj.ModificationTime(),
		Data:  data,
	})
	return nil
}

// Append registers a new object inside the commit.
func (c *Commit) Append(oid string, kind string, state State, mtime time.Time, data ObjectData) {
	c.Objects = append(c.Objects, CommitObject{
		OID:   oid,
		Kind:  kind,
		State: state,
		MTime: mtime,
		Data:  data,
	})
}

func (c *Commit) UniqueOID() string {
	return c.OID
}

// Read populates a commit from an object file.
func (c *Commit) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&c)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps a commit to an object file.
func (c *Commit) Write(w io.Writer) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (c *Commit) Blobs() []Blob {
	// Blobs are stored outside commits.
	return nil
}

// UnmarshallObject extract a single object from a commit.
func (c *Commit) UnmarshallObject(oid string, target interface{}) error {
	for _, objEdit := range c.Objects {
		if objEdit.OID == oid {
			return objEdit.Data.Unmarshal(target)
		}
	}
	return fmt.Errorf("no object with OID %q", oid)
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

// SetNextOIDs configures a predefined list of OID
func SetNextOIDs(oids ...string) {
	oidGenerator = &suiteOIDGenerator{
		nextOIDs: oids,
	}
}

// SetNextOID configures a predefined list of OID
func UseFixedOID(oid string) {
	oidGenerator = &fixedOIDGenerator{
		oid: oid,
	}
}

// ResetOID restores the original unique OID generator.
// Useful in tests with a defer after overriding the default generator.
func ResetOID() {
	oidGenerator = &uniqueOIDGenerator{}
}
