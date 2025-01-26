package core

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type OID string

const NilOID = OID("")
const MissingOID string = "4044044044044044044044044044044044044040"

func (o OID) IsNil() bool {
	return string(o) == ""
}

// Prefix returns the first two characters of the OID as used in .nt/objects.
func (o OID) Prefix() string {
	if o.IsNil() {
		return ""
	}
	return string(o)[0:2]
}

// RelativePath returns the relative path inside .nt/objects.
func (o OID) RelativePath() string {
	// We use the first two characters to spread objects into different directories
	// (same as .git/objects/) to avoid having a large unpractical directory.
	return fmt.Sprintf("%s/%s", o.Prefix(), o)
}

// String returns the OID as a string.
func (o OID) String() string {
	return string(o)
}

/* Constructors */

func NewOID() OID {
	return oidGenerator.NewOID()
}
func NewOIDFromBytes(b []byte) OID {
	return oidGenerator.NewOIDFromBytes(b)
}

/* Parser */

// MustParseOID parses an OID or panic if the OID format is not valid.
func MustParseOID(s string) OID {
	if len(s) != 40 {
		panic("Invalid OID")
	}
	return OID(s)
}

// ParseOIDOrNil parses an OID or returns NilOID.
func ParseOIDOrNil(s string) OID {
	if len(s) != 40 {
		return NilOID
	}
	return OID(s)
}

/* Generator */

type OIDGenerator interface {
	NewOID() OID
	NewOIDFromBytes(b []byte) OID
}

var oidGenerator OIDGenerator = &uniqueOIDGenerator{}

/* Test Generator */

type uniqueOIDGenerator struct{}

// NewOID generates an OID.
// Every call generates a new unique OID.
func (g *uniqueOIDGenerator) NewOID() OID {
	// We use the same "format" as Git (=40-length string) but use a content hash only for blob objects.
	// We use a randomly generated ID for other objects that is fixed even if objects are updated.

	// Ex (Git): 5e3f1b351782c017590b4b70fee709bf9c83b050
	// Ex (UUIDv4): 123e4567-e89b-12d3-a456-426655440000

	// Algorithm:
	// Remove `-` + add 8 random characters
	oid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:40]
	return OID(oid)
}

// NewOIDFromBytes generates an OID based on bytes.
// The same bytes will generate the same OID.
func (g *uniqueOIDGenerator) NewOIDFromBytes(b []byte) OID {
	h := sha1.New()
	h.Write(b)

	// This gets the finalized hash result as a byte
	// slice. The argument to `Sum` can be used to append
	// to an existing byte slice: it usually isn't needed.
	bs := h.Sum(nil)

	// SHA1 values are often printed in hex, for example
	// in git commits. Use the `%x` format verb to convert
	// a hash results to a hex string.
	return OID(fmt.Sprintf("%x\n", bs))
}

type suiteOIDGenerator struct {
	nextOIDs []string
}

func (g *suiteOIDGenerator) NewOID() OID {
	return g.nextOID()
}

func (g *suiteOIDGenerator) NewOIDFromBytes(b []byte) OID {
	return g.nextOID()
}

func (g *suiteOIDGenerator) nextOID() OID {
	if len(g.nextOIDs) > 0 {
		oid, nextOIDs := g.nextOIDs[0], g.nextOIDs[1:]
		g.nextOIDs = nextOIDs
		return OID(oid)
	}
	panic("No more OIDs")
}

type fixedOIDGenerator struct {
	oid OID
}

func (g *fixedOIDGenerator) NewOID() OID {
	return g.oid
}

func (g *fixedOIDGenerator) NewOIDFromBytes(b []byte) OID {
	return g.oid
}

type sequenceOIDGenerator struct {
	count int
}

func (g *sequenceOIDGenerator) NewOID() OID {
	g.count++
	return OID(fmt.Sprintf("%040d", g.count))
}

func (g *sequenceOIDGenerator) NewOIDFromBytes(b []byte) OID {
	return NewOID()
}

// ResetOID restores the original unique OID generator.
// Useful in tests with a defer after overriding the default generator.
func ResetOID() {
	oidGenerator = &uniqueOIDGenerator{}
}
