package core

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"gopkg.in/yaml.v3"
)

/*
 * ObjectData
 */


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
	if f, ok := target.(*Study); ok {
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

/*
 * PackFile
 */

type PackFile struct {
	OID         string        `yaml:"oid" json:"oid"`
	CTime       time.Time     `yaml:"ctime" json:"ctime"`
	MTime       time.Time     `yaml:"mtime" json:"mtime"`
	PackObjects []*PackObject `yaml:"objects" json:"objects"`
}

type PackObject struct {
	OID         string     `yaml:"oid" json:"oid"`
	Kind        string     `yaml:"kind" json:"kind"`
	State       State      `yaml:"state" json:"state"` // (A) added, (D) deleted, (M) modified, (R) renamed
	MTime       time.Time  `yaml:"mtime" json:"mtime"`
	Description string     `yaml:"desc" json:"desc"`
	Data        ObjectData `yaml:"data" json:"data"`
}


// NewPackFileRefWithOID initializes a new empty pack file ref with a given OID.
func NewPackFileRefWithOID(oid string) *PackFileRef {
	return &PackFileRef{
		OID:   oid,
		CTime: clock.Now(),
		MTime: clock.Now(),
	}
}

// NewPackFile initializes a new empty pack file.
func NewPackFile() *PackFile {
	return &PackFile{
		OID:   NewOID(),
		CTime: clock.Now(),
		MTime: clock.Now(),
	}
}

// NewPackFileWithOID initializes a new empty pack file with a given OID.
func NewPackFileWithOID(oid string) *PackFile {
	return &PackFile{
		OID:   oid,
		CTime: clock.Now(),
		MTime: clock.Now(),
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
	case "study":
		study := new(Study)
		p.Data.Unmarshal(study)
		return study
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

// Ref returns a ref to the pack file.
func (p *PackFile) Ref() *PackFileRef {
	return &PackFileRef{
		OID:   p.OID,
		CTime: p.CTime,
		MTime: p.MTime,
	}
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
	return p.SaveTo(path)
}

// SaveTo writes a new pack file to the given location.
func (p *PackFile) SaveTo(path string) error {
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

/*
 * PackFileRef
 */

type PackFileRef struct {
	OID   string    `yaml:"oid" json:"oid"`
	CTime time.Time `yaml:"ctime" json:"ctime"`
	MTime time.Time `yaml:"mtime" json:"mtime"`
}

// Convenient type to add methods
type PackFileRefs []*PackFileRef

func (p PackFileRefs) OIDs() []string {
	var results []string
	for _, packFileRef := range p {
		results = append(results, packFileRef.OID)
	}
	return results
}

