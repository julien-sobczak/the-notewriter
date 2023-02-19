package core

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

/* Commit */

type Commit struct {
	OID string
}

// NewCommitFromObject instantiates a new commit from an object file.
func NewCommitFromObject(r io.Reader) *Commit {
	// TODO
	return &Commit{}
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
