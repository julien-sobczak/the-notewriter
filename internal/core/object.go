package core

import (
	"database/sql"
	"io"
	"time"
)

type Blob struct {
	// OID to locate the blob file in .nt/objects
	OID        string
	Attributes map[string]interface{}
	Tags       []string
}

func (b *Blob) Hash() string {
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
	SubObjects() []Object
	// Blobs returns the optional blobs associated with this object.
	Blobs() []Blob

	// State returns the current state.
	State() State
	// SetTombstone marks the object as to be deleted.
	SetTombstone()

	// Save persists to DB.
	Save(tx *sql.Tx) error

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}

// Same for other objects

// Command Add
// _file_ = read the file
// _path_ = traverse the path
// . = traverse the work tree
// Same as current Build() but:
// - create blobs in `.nt/objects/`
// - append to `.nt/index`:
