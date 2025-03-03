package main

// Lite version of internal/core/object.go

import (
	"io"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
)

// Object groups method common to all kinds of managed objects.
// Useful when creating commits in a generic way where a single commit
// groups different kinds of objects inside the same object.
type Object interface {
	// Kind returns the object kind to determine which kind of object to create.
	Kind() string
	// UniqueOID returns the OID of the object.
	UniqueOID() oid.OID
	// ModificationTime returns the last modification time.
	ModificationTime() time.Time

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}

// StatefulObject to represent the subset of updatable objects persisted in database.
type StatefulObject interface {
	Object

	// Save persists to DB
	Save() error

	// Update website/guides/devolopers/presentation.md
}
