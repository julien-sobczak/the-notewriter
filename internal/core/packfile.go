package core

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
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
	if l, ok := target.(*GoLink); ok {
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
	OID              oid.OID       `yaml:"oid" json:"oid"`
	FileRelativePath string        `yaml:"file_relative_path" json:"file_relative_path"`
	FileMTime        time.Time     `yaml:"file_mtime" json:"file_mtime"`
	FileSize         int64         `yaml:"file_size" json:"file_size"`
	CTime            time.Time     `yaml:"ctime" json:"ctime"`
	PackObjects      []*PackObject `yaml:"objects" json:"objects"`
	BlobRefs         []*BlobRef    `yaml:"blobs" json:"blobs"`
}

type PackObject struct {
	OID         oid.OID    `yaml:"oid" json:"oid"`
	Kind        string     `yaml:"kind" json:"kind"`
	CTime       time.Time  `yaml:"ctime" json:"ctime"`
	Description string     `yaml:"desc" json:"desc"`
	Data        ObjectData `yaml:"data" json:"data"`
}

// NewPackFile initializes a new empty pack file.
func NewPackFile(fileObject FileObject) *PackFile {
	return &PackFile{
		OID: fileObject.UniqueOID(),

		// Init file properties
		FileRelativePath: fileObject.FileRelativePath(),
		FileMTime:        fileObject.FileMTime(),
		FileSize:         fileObject.FileSize(),

		// Init pack file properties
		CTime: clock.Now(),
	}
}

// ReadObject recreates the core object from a commit object.
func (p *PackObject) ReadObject() Object {
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
		link := new(GoLink)
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

// NewPackFileFromPath reads a pack file file on disk.
func NewPackFileFromPath(path string) (*PackFile, error) {
	in, err := os.Open(path)
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
func (p *PackFile) Ref() PackFileRef {
	return PackFileRef{
		RelativePath: p.FileRelativePath,
		OID:          p.OID,
		CTime:        p.CTime,
	}
}

// GetPackObject retrieves an object from a pack file.
func (p *PackFile) GetPackObject(oid oid.OID) (*PackObject, bool) {
	for _, object := range p.PackObjects {
		if object.OID == oid {
			return object, true
		}
	}
	return nil, false
}

// AppendPackObject registers a new object inside the pack file.
func (p *PackFile) AppendPackObject(obj *PackObject) {
	p.PackObjects = append(p.PackObjects, obj)
}

// MustAppendObject registers a new object inside the pack file or panic.
func (p *PackFile) MustAppendObject(obj Object) {
	if err := p.AppendObject(obj); err != nil {
		panic(err)
	}
}

// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj Object) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:         obj.UniqueOID(),
		Kind:        obj.Kind(),
		CTime:       obj.ModificationTime(),
		Description: obj.String(),
		Data:        data,
	})
	return nil
}

// AppendBlob registers a new blob inside the pack file.
func (p *PackFile) AppendBlob(blob *BlobRef) error {
	p.BlobRefs = append(p.BlobRefs, blob)
	return nil
}

// AppendBlobs registers new blobs inside the pack file.
func (p *PackFile) AppendBlobs(blobs []*BlobRef) error {
	p.BlobRefs = append(p.BlobRefs, blobs...)
	return nil
}

// UnmarshallObject extract a single object from a commit.
func (p *PackFile) UnmarshallObject(oid oid.OID, target interface{}) error {
	for _, objEdit := range p.PackObjects {
		if objEdit.OID == oid {
			return objEdit.Data.Unmarshal(target)
		}
	}
	return fmt.Errorf("no object with OID %q", oid)
}

// FindFirstBlobWithMimeType returns the first blob with the given mime type.
func (p *PackFile) FindFirstBlobWithMimeType(mimeType string) *BlobRef {
	for _, blob := range p.BlobRefs {
		if blob.MimeType == mimeType {
			return blob
		}
	}
	return nil
}

/* Object */

func (p *PackFile) UniqueOID() oid.OID {
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
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects/"+p.OID.RelativePath()+".pack")
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

/* Interface Dumpable */

func (p *PackFile) ToYAML() string {
	return ToBeautifulYAML(p)
}

func (p *PackFile) ToJSON() string {
	return ToBeautifulJSON(p)
}

func (p *PackFile) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PackFile %s\n\n", p.OID))
	sb.WriteString("## Objects\n\n")
	for _, obj := range p.PackObjects {
		sb.WriteString(fmt.Sprintf("* %s: %s `@oid: %s`\n", obj.Kind, obj.Description, obj.OID))
	}
	return sb.String()
}

/*
 * PackFileRef
 */

type PackFileRef struct {
	RelativePath string    `yaml:"relative_path" json:"relative_path"`
	OID          oid.OID   `yaml:"oid" json:"oid"`
	CTime        time.Time `yaml:"ctime" json:"ctime"`
}

// Convenient type to add methods
type PackFileRefs []PackFileRef

func (p PackFileRefs) OIDs() []oid.OID {
	var results []oid.OID
	for _, packFileRef := range p {
		results = append(results, packFileRef.OID)
	}
	return results
}
