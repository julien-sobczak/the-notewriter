package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

type ObjectRef interface {
	ObjectOID() oid.OID
	ObjectPath() string
	ObjectRelativePath() string
}

type BlobRef struct {
	// OID to locate the blob file in .nt/objects
	OID        oid.OID      `yaml:"oid" json:"oid"`
	MimeType   string       `yaml:"mime" json:"mime"`
	Attributes AttributeSet `yaml:"attributes" json:"attributes"`
	Tags       TagSet       `yaml:"tags" json:"tags"`
}

func (b *BlobRef) ToYAML() string {
	return ToBeautifulYAML(b)
}

func (b *BlobRef) ToJSON() string {
	return ToBeautifulJSON(b)
}

func (b *BlobRef) ToMarkdown() string {
	return fmt.Sprintf("Blob %s %s\n", b.OID, b.MimeType)
}

// ObjectOID returns the OID of the blob.
func (b BlobRef) ObjectOID() oid.OID {
	return b.OID
}

// ObjectPath returns the absolute path to the blob file in .nt/objects/ directory.
func (b BlobRef) ObjectPath() string {
	return BlobPath(b.OID)
}

// ObjectRelativePath returns the relative path to the blob file inside .nt/ directory.
func (b BlobRef) ObjectRelativePath() string {
	return BlobRelativePath(b.OID)
}

// BlobPath returns the path to the blob file in .nt/objects/ directory.
func BlobPath(oid oid.OID) string {
	return filepath.Join(CurrentConfig().RootDirectory, ".nt", BlobRelativePath(oid))
}

// BlobRelativePath returns the path to the blob file in .nt/objects/ directory.
func BlobRelativePath(oid oid.OID) string {
	return "objects/" + oid.RelativePath(".blob")
}

type Dumpable interface {
	ToYAML() string
	ToJSON() string
	ToMarkdown() string
}

// Object groups method common to all kinds of managed objects.
// Useful when creating commits in a generic way where a single commit
// groups different kinds of objects inside the same object.
type Object interface {
	Dumpable

	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() oid.OID
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// Relations returns the relations where the current object is the source.
	Relations() []*Relation

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error

	// String returns a one-line description
	String() string

	// Update website/guides/devolopers/presentation.md
}

// ParseObject to represent the subset of objects that was parsed from a Markdown file.
// For example, studies are not based on Markdown.
type ParsedObject interface {
	Object

	// RelativePath containing the version of this object.
	FileRelativePath() string
}

// StatefulObject to represent the subset of updatable objects persisted in database.
type StatefulObject interface {
	Object

	// Save persists to DB
	Save() error
	// Delete removes from DB
	Delete() error

	// Update website/guides/devolopers/presentation.md
}

// FileObject represents an object present as a file in the repository.
type FileObject interface {
	// UniqueOID of the object representing the file
	UniqueOID() oid.OID

	// Relative path to repository
	FileRelativePath() string
	// Timestamp of last content modification
	FileMTime() time.Time
	// Size of the file
	FileSize() int64
	// MD5 Checksum
	FileHash() string

	Blobs() []*BlobRef
}

type BlobFile struct {
	Ref  *BlobRef
	Data []byte
}

func NewBlobFile(ref *BlobRef, data []byte) *BlobFile {
	return &BlobFile{
		Ref:  ref,
		Data: data,
	}
}

// Read populates a commit from an object file.
func (c *BlobFile) Read(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	c.Data = data
	return nil
}

// Write dumps a commit to an object file.
func (c *BlobFile) Write(w io.Writer) error {
	_, err := w.Write(c.Data)
	if err != nil {
		return err
	}
	return err
}

// Save writes a new file inside .nt/objects.
func (c *BlobFile) Save() error {
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", c.Ref.OID.RelativePath("..blob"))
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Write(f)
}

// Convenient type to add methods
type BlobRefs []BlobRef

// OIDs returns the list of OIDs.
func (r BlobRefs) OIDs() []oid.OID {
	var results []oid.OID
	for _, ref := range r {
		results = append(results, ref.OID)
	}
	return results
}

/* Utility */

func ToBeautifulYAML(obj any) string {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(2)
	_ = bufEncoder.Encode(obj)
	return buf.String()
}

func ToBeautifulJSON(obj any) string {
	output, _ := json.MarshalIndent(obj, "", "  ")
	return string(output)
}
