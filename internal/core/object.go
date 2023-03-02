package core

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"time"
)

// OIDToPath converts an oid to a file path.
func OIDToPath(oid string) string {
	// We use the first two characters to spread objects into different directories
	// (same as .git/objects/) to avoid having a large unpractical directory.
	return oid[0:2] + "/" + oid
}

type BlobRef struct {
	// OID to locate the blob file in .nt/objects
	OID        string
	Attributes map[string]interface{}
	Tags       []string
}

func (b *BlobRef) Hash() string {
	// TODO
	return ""
}

// Object groups method common to all kinds of managed objects.
// Useful when creating commits in a generic way where a single commit
// groups different kinds of objects inside the same object.
type Object interface {
	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() string
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// SubObjects returns the objects directly contained by this object.
	SubObjects() []StatefulObject
	// Blobs returns the optional blobs associated with this object.
	Blobs() []BlobRef

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error

	// String returns a one-line description
	String() string
}

// StatefulObject to represent the subset of updatable objects persisted in database.
type StatefulObject interface {
	Object

	// State returns the current state.
	State() State
	// ForceState marks the object in the given state
	ForceState(newState State)

	// Save persists to DB.
	Save(tx *sql.Tx) error
}

type BlobFile struct {
	Ref  BlobRef
	Data []byte
}

func NewBlobFile(ref BlobRef, data []byte) *BlobFile {
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
	path := filepath.Join(CurrentConfig().RootDirectory, ".nt/objects", OIDToPath(c.Ref.OID))
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

// Same for other objects

// Command Add
// _file_ = read the file
// _path_ = traverse the path
// . = traverse the work tree
// Same as current Build() but:
// - create blobs in `.nt/objects/`
// - append to `.nt/index`:
