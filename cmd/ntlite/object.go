package main
// Lite version of internal/core/object.go

import (
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewOID generates an OID.
// Every call generates a new unique OID.
func NewOID() string {
	// We use the same "format" as Git (=40-length string) but use a content hash only for blob objects.
	// We use a randomly generated ID for other objects that is fixed even if objects are updated.

	// Ex (Git): 5e3f1b351782c017590b4b70fee709bf9c83b050
	// Ex (UUIDv4): 123e4567-e89b-12d3-a456-426655440000

	// Algorithm:
	// Remove `-` + add 8 random characters
	oid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:40]
	return oid
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

	// Read rereads the object from YAML.
	Read(r io.Reader) error
	// Write writes the object to YAML.
	Write(w io.Writer) error
}

// StatefulObject to represent the subset of updatable objects persisted in database.
type StatefulObject interface {
	Object

	// State returns the current state.
	State() State

	// Save persists to DB
	Save() error

	// Update website/guides/devolopers/presentation.md
}
