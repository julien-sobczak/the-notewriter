package remote

import (
	"errors"
)

var (
	ErrObjectNotExist = errors.New("object does not exist")
)

// Remote provides an abstraction in front of remote implementations.
//
// A remote must be able to save different files:
// - info files (ex: index)
// - pack files (ex: files, medias)
// - blob files (ex: medias in various sizes)
//
// A remote is free to save files in any format as long as it can retrieve
// the same field when querying using the same key.
type Remote interface {
	GetObject(key string) ([]byte, error)
	PutObject(key string, content []byte) error
	DeleteObject(key string) error
	GC() error
	// Note: File permissions are not important concerning object. MTime, etc. must be stored inside the object definitions if useful.
}
