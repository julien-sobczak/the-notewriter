package core

import (
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
	Kind() string
	UniqueOID() string
	ModificationTime() time.Time
	Read(r io.Reader) error
	Write(w io.Writer) error
	Blobs() []Blob // OID size tags
}

// Same for other objects

// Command Add
// _file_ = read the file
// _path_ = traverse the path
// . = traverse the work tree
// Same as current Build() but:
// - create blobs in `.nt/objects/`
// - append to `.nt/index`:
